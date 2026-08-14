package handlers

import (
	"context"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/db"
)

type TriggerMonitorSearchInput struct {
	ID uint `path:"id" maximum:"1000000000" minimum:"1" example:"42" doc:"Monitor ID"`
}

func TriggerMonitorSearch() func(context.Context, *TriggerMonitorSearchInput) (*struct{}, error) {
	return func(ctx context.Context, input *TriggerMonitorSearchInput) (*struct{}, error) {
		monitorID := strconv.FormatUint(uint64(input.ID), 10)
		if err := db.UpdateMonitorStatusByID(monitorID, "pending"); err != nil {
			return nil, huma.Error500InternalServerError("failed to trigger search", err)
		}
		return nil, nil
	}
}