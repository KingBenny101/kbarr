package handlers

import (
	"context"
	"sync/atomic"

	"github.com/kingbenny101/kbarr/internal/core/db"
	coreservice "github.com/kingbenny101/kbarr/internal/core/service"
)

func Health() func(context.Context, *struct{}) (*HealthOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		return &HealthOutput{Body: MessageResponse{Message: "KBArr is running"}}, nil
	}
}

// avChecking is 1 while a manual availability check is in progress.
// Concurrent triggers are ignored so the scan is never run twice simultaneously.
var avChecking atomic.Int32

func CheckAvailability() func(context.Context, *struct{}) (*struct{}, error) {
	return func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if !avChecking.CompareAndSwap(0, 1) {
			return nil, nil
		}
		go func() {
			defer avChecking.Store(0)
			coreservice.CheckAvailability(context.Background(), db.DB)
		}()
		return nil, nil
	}
}
