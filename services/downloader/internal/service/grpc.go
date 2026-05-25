package service

import (
	"context"

	proto "github.com/kingbenny101/kbarr/shared/proto"
)

type GrpcServer struct {
	proto.UnimplementedDownloaderServer
	svc *DownloaderService
}

func NewGrpcServer(svc *DownloaderService) *GrpcServer {
	return &GrpcServer{svc: svc}
}

func (s *GrpcServer) AddTorrent(ctx context.Context, req *proto.AddTorrentRequest) (*proto.AddTorrentResponse, error) {
	return s.svc.AddTorrent(ctx, req)
}

func (s *GrpcServer) GetTorrent(ctx context.Context, req *proto.TorrentRequest) (*proto.TorrentResponse, error) {
	return s.svc.GetTorrent(ctx, req)
}

func (s *GrpcServer) ListTorrents(ctx context.Context, req *proto.ListTorrentsRequest) (*proto.ListTorrentsResponse, error) {
	torrents, err := s.svc.ListTorrents(ctx, req)
	if err != nil {
		return &proto.ListTorrentsResponse{Torrents: []*proto.TorrentResponse{}}, nil
	}

	return &proto.ListTorrentsResponse{Torrents: torrents}, nil
}

func (s *GrpcServer) RemoveTorrent(ctx context.Context, req *proto.RemoveTorrentRequest) (*proto.RemoveTorrentResponse, error) {
	return s.svc.RemoveTorrent(ctx, req)
}
