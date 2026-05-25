package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	proto "github.com/kingbenny101/kbarr/shared/proto"
)

type DownloaderService struct {
	qbtURL    string
	qbtClient *http.Client
}

func NewDownloaderService(qbtURL string) *DownloaderService {
	return &DownloaderService{
		qbtURL:    strings.TrimRight(strings.TrimSpace(qbtURL), "/"),
		qbtClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type torrentInfo struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	State    string  `json:"state"`
	Size     int64   `json:"size"`
	Progress float32 `json:"progress"`
	ETA      int64   `json:"eta"`
	SavePath string  `json:"save_path"`
	Category string  `json:"category"`
}

func (s *DownloaderService) AddTorrent(ctx context.Context, req *proto.AddTorrentRequest) (*proto.AddTorrentResponse, error) {
	if req == nil {
		return &proto.AddTorrentResponse{Success: false, Error: "request is nil"}, nil
	}
	if s.qbtURL == "" {
		return &proto.AddTorrentResponse{Success: false, Error: "qBittorrent URL is not configured"}, nil
	}

	body := &strings.Builder{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("urls", req.GetMagnetUrl()); err != nil {
		return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
	}
	if req.GetSavePath() != "" {
		if err := writer.WriteField("savepath", req.GetSavePath()); err != nil {
			return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
		}
	}
	if req.GetCategory() != "" {
		if err := writer.WriteField("category", req.GetCategory()); err != nil {
			return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
		}
	}
	if err := writer.Close(); err != nil {
		return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.qbtURL+"/api/v2/torrents/add", strings.NewReader(body.String()))
	if err != nil {
		return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.qbtClient.Do(httpReq)
	if err != nil {
		return &proto.AddTorrentResponse{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &proto.AddTorrentResponse{Success: false, Error: fmt.Sprintf("qBittorrent returned %s", resp.Status)}, nil
	}

	return &proto.AddTorrentResponse{Hash: extractMagnetHash(req.GetMagnetUrl()), Success: true}, nil
}

func (s *DownloaderService) GetTorrent(ctx context.Context, req *proto.TorrentRequest) (*proto.TorrentResponse, error) {
	if req == nil {
		return &proto.TorrentResponse{}, nil
	}

	items, err := s.fetchTorrents(ctx, req.GetHash(), "")
	if err != nil || len(items) == 0 {
		return &proto.TorrentResponse{Hash: req.GetHash()}, nil
	}

	return mapTorrent(items[0]), nil
}

func (s *DownloaderService) ListTorrents(ctx context.Context, req *proto.ListTorrentsRequest) ([]*proto.TorrentResponse, error) {
	category := ""
	if req != nil {
		category = req.GetCategory()
	}

	items, err := s.fetchTorrents(ctx, "", category)
	if err != nil {
		return []*proto.TorrentResponse{}, nil
	}

	out := make([]*proto.TorrentResponse, 0, len(items))
	for _, item := range items {
		out = append(out, mapTorrent(item))
	}
	return out, nil
}

func (s *DownloaderService) RemoveTorrent(ctx context.Context, req *proto.RemoveTorrentRequest) (*proto.RemoveTorrentResponse, error) {
	if req == nil {
		return &proto.RemoveTorrentResponse{Success: false, Error: "request is nil"}, nil
	}
	if s.qbtURL == "" {
		return &proto.RemoveTorrentResponse{Success: false, Error: "qBittorrent URL is not configured"}, nil
	}

	form := url.Values{}
	form.Set("hashes", req.GetHash())
	form.Set("deleteFiles", fmt.Sprintf("%t", req.GetDeleteFiles()))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.qbtURL+"/api/v2/torrents/delete", strings.NewReader(form.Encode()))
	if err != nil {
		return &proto.RemoveTorrentResponse{Success: false, Error: err.Error()}, nil
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.qbtClient.Do(httpReq)
	if err != nil {
		return &proto.RemoveTorrentResponse{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &proto.RemoveTorrentResponse{Success: false, Error: fmt.Sprintf("qBittorrent returned %s", resp.Status)}, nil
	}

	return &proto.RemoveTorrentResponse{Success: true}, nil
}

func (s *DownloaderService) fetchTorrents(ctx context.Context, hash string, category string) ([]torrentInfo, error) {
	if s.qbtURL == "" {
		return nil, fmt.Errorf("qBittorrent URL is not configured")
	}

	query := url.Values{}
	if category != "" {
		query.Set("category", category)
	}
	if hash != "" {
		query.Set("hashes", hash)
	}

	endpoint := s.qbtURL + "/api/v2/torrents/info"
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.qbtClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("qBittorrent returned %s", resp.Status)
	}

	var items []torrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	return items, nil
}

func mapTorrent(item torrentInfo) *proto.TorrentResponse {
	return &proto.TorrentResponse{
		Hash:     item.Hash,
		Name:     item.Name,
		State:    item.State,
		Size:     item.Size,
		Progress: item.Progress,
		Eta:      item.ETA,
		SavePath: item.SavePath,
		Category: item.Category,
	}
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
