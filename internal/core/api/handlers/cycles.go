package handlers

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

type CycleStatus struct {
	Service        string     `json:"service"`
	Cycle          string     `json:"cycle"`
	DisplayName    string     `json:"display_name"`
	State          string     `json:"state"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastDurationMs int64      `json:"last_duration_ms"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
}

type CyclesOutput struct {
	Body struct {
		Cycles []CycleStatus `json:"cycles"`
	}
}

// GetCycles reports the last/next run times of every background cycle.
// Pure DB read: service liveness is reported separately by GET /api/workers.
func GetCycles() func(context.Context, *struct{}) (*CyclesOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*CyclesOutput, error) {
		var rows = []CycleStatus{}
		if err := db.DB.NewRaw(
			"SELECT service, cycle, display_name, state, last_started_at, last_finished_at, last_duration_ms, next_run_at FROM cycle_status ORDER BY next_run_at",
		).Scan(ctx, &rows); err != nil {
			return nil, huma.Error500InternalServerError("failed to read cycle status", err)
		}
		return &CyclesOutput{Body: struct {
			Cycles []CycleStatus `json:"cycles"`
		}{Cycles: rows}}, nil
	}
}
