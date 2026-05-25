package service

import (
	"context"

	proto "github.com/kingbenny101/kbarr/shared/proto"
)

type GRPCServer struct {
	proto.UnimplementedProwlarrServiceServer
	svc *ProwlarrService
}

func NewGRPCServer(svc *ProwlarrService) *GRPCServer {
	return &GRPCServer{svc: svc}
}

func (s *GRPCServer) Search(ctx context.Context, req *proto.ProwlarrSearchRequest) (*proto.ProwlarrSearchResponse, error) {
	results, err := s.svc.Search(req.GetQuery())
	if err != nil {
		return nil, err
	}

	return &proto.ProwlarrSearchResponse{Results: results}, nil
}
