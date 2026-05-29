package service

import (
	"context"

	downloader "github.com/kingbenny101/kbarr/shared/proto/downloader"
)

type GrpcServer struct {
	downloader.UnimplementedDownloaderServer
	svc *DownloaderService
}

func NewGrpcServer(svc *DownloaderService) *GrpcServer {
	return &GrpcServer{svc: svc}
}

func (s *GrpcServer) AddTorrent(ctx context.Context, req *downloader.AddTorrentRequest) (*downloader.AddTorrentResponse, error) {
	return s.svc.AddTorrent(ctx, req)
}

func (s *GrpcServer) GetTorrent(ctx context.Context, req *downloader.TorrentRequest) (*downloader.TorrentResponse, error) {
	return s.svc.GetTorrent(ctx, req)
}

func (s *GrpcServer) ListTorrents(ctx context.Context, req *downloader.ListTorrentsRequest) (*downloader.ListTorrentsResponse, error) {
	torrents, err := s.svc.ListTorrents(ctx, req)
	if err != nil {
		return &downloader.ListTorrentsResponse{Torrents: []*downloader.TorrentResponse{}}, nil
	}

	return &downloader.ListTorrentsResponse{Torrents: torrents}, nil
}

func (s *GrpcServer) RemoveTorrent(ctx context.Context, req *downloader.RemoveTorrentRequest) (*downloader.RemoveTorrentResponse, error) {
	return s.svc.RemoveTorrent(ctx, req)
}
