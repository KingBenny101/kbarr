package handlers

import (
	"context"

	"github.com/kingbenny101/kbarr/internal/core/db"
	coreservice "github.com/kingbenny101/kbarr/internal/core/service"
)

func Health() func(context.Context, *struct{}) (*HealthOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		return &HealthOutput{Body: MessageResponse{Message: "KBArr is running"}}, nil
	}
}

func CheckAvailability() func(context.Context, *struct{}) (*struct{}, error) {
	return func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		coreservice.CheckAvailability(ctx, db.DB)
		return nil, nil
	}
}
