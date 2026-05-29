package service

import (
	"context"

	indexerpb "github.com/kingbenny101/kbarr/shared/proto/indexer"
)

type IndexerGrpcServer struct {
	indexerpb.UnimplementedIndexerServiceServer
	svc *IndexerService
}

func NewGRPCServer(svc *IndexerService) *IndexerGrpcServer {
	return &IndexerGrpcServer{svc: svc}
}

func (s *IndexerGrpcServer) Search(ctx context.Context, req *indexerpb.ProwlarrSearchRequest) (*indexerpb.ProwlarrSearchResponse, error) {
	results, err := s.svc.Search(req.GetQuery())
	if err != nil {
		return nil, err
	}

	return &indexerpb.ProwlarrSearchResponse{Results: results}, nil
}
