package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/kingbenny101/kbarr/internal/config"
	dlprovider "github.com/kingbenny101/kbarr/internal/downloader/provider"
	"github.com/uptrace/bun"
)

func init() {
	dlprovider.Register("qbittorrent", func(db *bun.DB) dlprovider.TorrentClient {
		jar, _ := cookiejar.New(nil)
		return &QBittorrentClient{
			db:        db,
			qbtClient: &http.Client{Timeout: 30 * time.Second, Jar: jar},
			httpClient: &http.Client{
				Timeout: 60 * time.Second,
				CheckRedirect: func(req *http.Request, _ []*http.Request) error {
					if req.URL.Scheme == "magnet" {
						return http.ErrUseLastResponse
					}
					return nil
				},
			},
		}
	})
}

type QBittorrentClient struct {
	db         *bun.DB
	qbtClient  *http.Client
	httpClient *http.Client
}

func (c *QBittorrentClient) qbtConfig() (qbtURL, username, password string) {
	qbtURL = strings.TrimRight(config.Get(c.db, "qbittorrentUrl", "http://localhost:8080"), "/")
	username = config.Get(c.db, "qbittorrentUsername", "")
	password = config.Get(c.db, "qbittorrentPassword", "")
	return
}

func (c *QBittorrentClient) Login(ctx context.Context) error {
	qbtURL, username, password := c.qbtConfig()
	if username == "" {
		return nil
	}
	resp, err := c.qbtClient.PostForm(qbtURL+"/api/v2/auth/login", url.Values{
		"username": {username},
		"password": {password},
	})
	if err != nil {
		return fmt.Errorf("qBittorrent login failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("qBittorrent login returned %s", resp.Status)
	}
	return nil
}

func (c *QBittorrentClient) AddTorrent(ctx context.Context, torrentURL string) (string, error) {
	qbtURL, _, _ := c.qbtConfig()
	if qbtURL == "" {
		return "", fmt.Errorf("qBittorrent URL is not configured")
	}

	var (
		buf    strings.Builder
		writer = multipart.NewWriter(&buf)
	)

	effectiveURL := torrentURL
	if !strings.HasPrefix(torrentURL, "magnet:") {
		magnetURL, torrentBytes, err := c.resolveURL(ctx, torrentURL)
		if err != nil {
			return "", fmt.Errorf("failed to resolve torrent URL: %w", err)
		}
		if magnetURL != "" {
			effectiveURL = magnetURL
		} else {
			part, err := writer.CreateFormFile("torrents", "file.torrent")
			if err != nil {
				return "", err
			}
			if _, err := part.Write(torrentBytes); err != nil {
				return "", err
			}
			effectiveURL = ""
		}
	}

	if effectiveURL != "" {
		if err := writer.WriteField("urls", effectiveURL); err != nil {
			return "", err
		}
	}
	if err := writer.WriteField("tags", "kbarr"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/add", strings.NewReader(buf.String()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.qbtClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("qBittorrent returned %s", resp.Status)
	}

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(respBody))
	prefix := torrentURL
	if len(prefix) > 80 {
		prefix = prefix[:80]
	}
	slog.Info("qBittorrent add response", "status", resp.StatusCode, "body", bodyStr, "url_prefix", prefix)
	if bodyStr == "Fails." {
		return "", fmt.Errorf("qBittorrent rejected the torrent (Fails.)")
	}

	var jsonResp struct {
		AddedIDs []string `json:"added_torrent_ids"`
	}
	if err := json.Unmarshal([]byte(bodyStr), &jsonResp); err == nil && len(jsonResp.AddedIDs) > 0 {
		return jsonResp.AddedIDs[0], nil
	}

	return extractMagnetHash(torrentURL), nil
}

func (c *QBittorrentClient) FetchTorrents(ctx context.Context, hash, category string) ([]dlprovider.TorrentInfo, error) {
	qbtURL, _, _ := c.qbtConfig()
	if qbtURL == "" {
		return nil, fmt.Errorf("qBittorrent URL is not configured")
	}

	query := url.Values{}
	if category != "" {
		query.Set("category", category)
	}
	if hash != "" {
		query.Set("hashes", hash)
	}

	endpoint := qbtURL + "/api/v2/torrents/info"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.qbtClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qBittorrent returned %s", resp.Status)
	}

	var raw []struct {
		Hash        string  `json:"hash"`
		Name        string  `json:"name"`
		ContentPath string  `json:"content_path"`
		State       string  `json:"state"`
		Size        int64   `json:"size"`
		Progress    float64 `json:"progress"`
		ETA         int64   `json:"eta"`
		SavePath    string  `json:"save_path"`
		Category    string  `json:"category"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	items := make([]dlprovider.TorrentInfo, len(raw))
	for i, r := range raw {
		items[i] = dlprovider.TorrentInfo{
			Hash:        r.Hash,
			Name:        r.Name,
			ContentPath: r.ContentPath,
			State:       r.State,
			Size:        r.Size,
			Progress:    r.Progress,
			ETA:         r.ETA,
			SavePath:    r.SavePath,
			Category:    r.Category,
		}
	}
	return items, nil
}

func (c *QBittorrentClient) DeleteTorrent(ctx context.Context, hash string, deleteFiles bool) error {
	qbtURL, _, _ := c.qbtConfig()
	deleteFilesStr := "false"
	if deleteFiles {
		deleteFilesStr = "true"
	}
	form := url.Values{"hashes": {hash}, "deleteFiles": {deleteFilesStr}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/delete", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.qbtClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// resolveURL fetches an HTTP URL. If the server redirects to a magnet link,
// it returns (magnetURL, nil, nil). Otherwise it returns ("", bytes, nil).
func (c *QBittorrentClient) resolveURL(ctx context.Context, torrentURL string) (magnetURL string, torrentBytes []byte, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torrentURL, nil)
	if err != nil {
		return "", nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 == 3 {
		loc := resp.Header.Get("Location")
		if strings.HasPrefix(loc, "magnet:") {
			return loc, nil, nil
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	if trimmed := bytes.TrimSpace(b); bytes.HasPrefix(trimmed, []byte("<?xml")) || bytes.HasPrefix(trimmed, []byte("<error")) {
		return "", nil, fmt.Errorf("indexer error response: %s", strings.TrimSpace(string(b)))
	}

	return "", b, nil
}

func extractMagnetHash(magnetURL string) string {
	parsed, err := url.Parse(magnetURL)
	if err != nil {
		return ""
	}
	xt := parsed.Query().Get("xt")
	if strings.HasPrefix(xt, "urn:btih:") {
		return strings.TrimPrefix(xt, "urn:btih:")
	}
	return ""
}
