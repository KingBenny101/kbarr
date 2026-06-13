package handlers

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/kingbenny101/kbarr/internal/core/db"
	"github.com/kingbenny101/kbarr/internal/core/service"
	"github.com/kingbenny101/kbarr/internal/models"
)

func GetMonitoredList() func(context.Context, *struct{}) (*MonitorListOutput, error) {
	return func(ctx context.Context, _ *struct{}) (*MonitorListOutput, error) {
		monitors, err := db.GetAllMonitored()
		if err != nil {
			slog.Error("Failed to fetch monitored list", "error", err)
			return nil, huma.Error500InternalServerError("failed to fetch monitored list", err)
		}
		return &MonitorListOutput{Body: monitors}, nil
	}
}

func AddMonitor() func(context.Context, *AddMonitorInput) (*struct{}, error) {
	return func(ctx context.Context, input *AddMonitorInput) (*struct{}, error) {
		slog.Info("AddMonitor called", "title", input.Body.Title)
		db.InsertMonitor(input.Body.ToModel())
		// Mark the new monitor downloaded if its file is already on disk, so the
		// indexer does not redundantly re-download it.
		service.ReconcileLibraryAvailability(ctx, db.DB, input.Body.LibraryID)
		return nil, nil
	}
}

func DeleteMonitor() func(context.Context, *MonitorIDInput) (*struct{}, error) {
	return func(ctx context.Context, input *MonitorIDInput) (*struct{}, error) {
		slog.Info("Delete monitor request", "id", input.ID)
		if err := db.DeleteMonitor(input.ID); err != nil {
			slog.Error("Failed to delete monitor entry", "error", err)
			return nil, huma.Error500InternalServerError("failed to delete monitor entry", err)
		}
		return nil, nil
	}
}

func GetMonitorsByLibraryID() func(context.Context, *LibraryIDInput) (*MonitorListOutput, error) {
	return func(ctx context.Context, input *LibraryIDInput) (*MonitorListOutput, error) {
		monitors, err := db.GetMonitorsByLibraryID(input.ID)
		if err != nil {
			slog.Error("Failed to fetch monitored items for library ID", "id", input.ID, "error", err)
			return nil, huma.Error500InternalServerError("failed to fetch monitored items", err)
		}
		return &MonitorListOutput{Body: monitors}, nil
	}
}

func Unmonitor() func(context.Context, *UnmonitorInput) (*struct{}, error) {
	return func(ctx context.Context, input *UnmonitorInput) (*struct{}, error) {
		if err := db.UnmonitorByDetails(input.Body.LibraryID, input.Body.ExternalID); err != nil {
			return nil, huma.Error500InternalServerError("failed to unmonitor", err)
		}
		return nil, nil
	}
}

func BulkAddMonitor() func(context.Context, *BulkAddMonitorInput) (*struct{}, error) {
	return func(ctx context.Context, input *BulkAddMonitorInput) (*struct{}, error) {
		slog.Info("BulkAddMonitor called", "items", len(input.Body))
		monitors := make([]models.Monitor, len(input.Body))
		libraryIDs := map[uint]bool{}
		for i, m := range input.Body {
			monitors[i] = m.ToModel()
			libraryIDs[m.LibraryID] = true
		}
		db.InsertMonitorsBulk(monitors)
		// Mark any newly-monitored episodes already on disk as downloaded so the
		// indexer skips them rather than re-downloading.
		for libraryID := range libraryIDs {
			service.ReconcileLibraryAvailability(ctx, db.DB, libraryID)
		}
		return nil, nil
	}
}

func UnmonitorSeason() func(context.Context, *UnmonitorSeasonInput) (*struct{}, error) {
	return func(ctx context.Context, input *UnmonitorSeasonInput) (*struct{}, error) {
		if err := db.DeleteMonitorsBySeason(input.Body.LibraryID, input.Body.Season); err != nil {
			return nil, huma.Error500InternalServerError("failed to unmonitor season", err)
		}
		return nil, nil
	}
}

