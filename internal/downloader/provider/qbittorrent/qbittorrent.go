package qbittorrent

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
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
	"github.com/zeebo/bencode"
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
		buf    bytes.Buffer
		writer = multipart.NewWriter(&buf)
	)

	// uploadedHash holds the infohash computed locally from an uploaded .torrent
	// file. qBittorrent's /torrents/add returns only "Ok." with no id, so without
	// this the caller cannot identify which torrent was just added.
	var uploadedHash string

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
			if h, err := infohashFromTorrent(torrentBytes); err != nil {
				slog.Warn("could not compute infohash from .torrent file", "error", err)
			} else {
				uploadedHash = h
			}
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
	// Also assign a dedicated category so kbarr's torrents are visually grouped and
	// filterable in qBittorrent. We intentionally do not scope FetchTorrents to it:
	// the duplicate-adopt path must still see torrents the user added manually.
	if err := writer.WriteField("category", "kbarr"); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/add", bytes.NewReader(buf.Bytes()))
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

	// Resolution order for the just-added torrent's infohash:
	//   1. an uploaded .torrent file we hashed locally,
	//   2. the btih hash embedded in a magnet link.
	// Both are exact; neither relies on guessing from the client's torrent list.
	if uploadedHash != "" {
		return uploadedHash, nil
	}
	return extractMagnetHash(torrentURL), nil
}

// infohashFromTorrent computes the v1 BitTorrent infohash (hex) of a .torrent
// file: the SHA-1 of the bencoded "info" dictionary. It hashes the dictionary's
// original raw bytes (via bencode.RawMessage) rather than a re-encode, because
// re-encoding can reorder keys and produce a different — and wrong — hash.
func infohashFromTorrent(torrentBytes []byte) (string, error) {
	var meta struct {
		Info bencode.RawMessage `bencode:"info"`
	}
	if err := bencode.DecodeBytes(torrentBytes, &meta); err != nil {
		return "", err
	}
	if len(meta.Info) == 0 {
		return "", fmt.Errorf("torrent file has no info dictionary")
	}
	sum := sha1.Sum(meta.Info)
	return hex.EncodeToString(sum[:]), nil
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
		DLSpeed     int64   `json:"dlspeed"`
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
			DLSpeed:     r.DLSpeed,
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
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qBittorrent returned %s", resp.Status)
	}
	return nil
}

// DefaultSavePath returns qBittorrent's configured default save directory.
func (c *QBittorrentClient) DefaultSavePath(ctx context.Context) (string, error) {
	qbtURL, _, _ := c.qbtConfig()
	if qbtURL == "" {
		return "", fmt.Errorf("qBittorrent URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qbtURL+"/api/v2/app/defaultSavePath", nil)
	if err != nil {
		return "", err
	}
	resp, err := c.qbtClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("qBittorrent returned %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// SetLocation moves a torrent's files to the given save directory.
func (c *QBittorrentClient) SetLocation(ctx context.Context, hash, location string) error {
	qbtURL, _, _ := c.qbtConfig()
	if qbtURL == "" {
		return fmt.Errorf("qBittorrent URL is not configured")
	}
	form := url.Values{"hashes": {hash}, "location": {location}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qbtURL+"/api/v2/torrents/setLocation", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.qbtClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("qBittorrent returned %s", resp.Status)
	}
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

	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
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
