package db

import (
	"context"
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
	"github.com/uptrace/bun"
)

func InsertDetailed(d models.Detailed) (int64, error) {
	if err := ensureDB(); err != nil {
		return 0, err
	}
	ctx := context.Background()
	episodes := d.Episodes
	d.Episodes = nil

	if _, err := DB.NewInsert().Model(&d).Returning("id").Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to insert detailed media: %w", err)
	}

	for i := range episodes {
		episodes[i].DetailedID = d.ID
		if _, err := DB.NewInsert().Model(&episodes[i]).Exec(ctx); err != nil {
			return 0, fmt.Errorf("failed to insert episode for detailed media %d: %w", d.ID, err)
		}
	}

	return int64(d.ID), nil
}

func GetDetailedByLibraryID(libraryID uint) (models.Detailed, error) {
	if err := ensureDB(); err != nil {
		return models.Detailed{}, err
	}
	var d models.Detailed
	if err := DB.NewSelect().Model(&d).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Scan(context.Background()); err != nil {
		return models.Detailed{}, fmt.Errorf("failed to get detailed media: %w", err)
	}
	return d, nil
}

type EpisodeQueryParams struct {
	Types     []int
	SortField string // "ep_no" | "title"
	SortOrder string // "asc" | "desc"
	Page      int
	Limit     int
}

type EpisodeQueryResult struct {
	Episodes     []models.Episode `json:"episodes"`
	Total        int              `json:"total"`
	Page         int              `json:"page"`
	Limit        int              `json:"limit"`
	PresentTypes []int            `json:"present_types"`
}

func QueryEpisodesByLibraryID(libraryID uint, p EpisodeQueryParams) (EpisodeQueryResult, error) {
	if err := ensureDB(); err != nil {
		return EpisodeQueryResult{}, err
	}
	ctx := context.Background()

	var d models.Detailed
	if err := DB.NewSelect().Model(&d).
		Where("library_id = ? AND deleted_at IS NULL", libraryID).
		Scan(ctx); err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to get detailed record: %w", err)
	}

	var rawTypes []struct {
		Type int64 `bun:"type"`
	}
	_ = DB.NewSelect().
		TableExpr("episodes").
		ColumnExpr("DISTINCT type").
		Where("detailed_id = ?", d.ID).
		Where("deleted_at IS NULL").
		Scan(ctx, &rawTypes)
	presentTypes := make([]int, 0, len(rawTypes))
	for _, rt := range rawTypes {
		presentTypes = append(presentTypes, int(rt.Type))
	}

	q := DB.NewSelect().Model((*models.Episode)(nil)).
		Where("detailed_id = ?", d.ID).
		Where("deleted_at IS NULL")

	if len(p.Types) > 0 {
		q = q.Where("type IN (?)", bun.In(p.Types))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to count episodes: %w", err)
	}

	sortCol := "ep_no"
	if p.SortField == "title" {
		sortCol = "title"
	}
	sortDir := "ASC"
	if p.SortOrder == "desc" {
		sortDir = "DESC"
	}

	if sortCol == "ep_no" {
		q = q.OrderExpr(
			"CASE WHEN ep_no ~ '^[0-9]+(\\.[0-9]+)?$' THEN CAST(ep_no AS NUMERIC) END " + sortDir + " NULLS LAST, ep_no " + sortDir,
		)
	} else {
		q = q.OrderExpr("? "+sortDir, bun.Ident(sortCol))
	}

	if p.Limit > 0 {
		q = q.Limit(p.Limit).Offset((p.Page - 1) * p.Limit)
	}

	var episodes []models.Episode
	if err := q.Scan(ctx, &episodes); err != nil {
		return EpisodeQueryResult{}, fmt.Errorf("failed to query episodes: %w", err)
	}

	return EpisodeQueryResult{
		Episodes:     episodes,
		Total:        total,
		Page:         p.Page,
		Limit:        p.Limit,
		PresentTypes: presentTypes,
	}, nil
}

func DeleteDetailedByLibraryID(libraryID uint) error {
	if err := ensureDB(); err != nil {
		return err
	}
	if _, err := DB.NewDelete().Model((*models.Detailed)(nil)).
		Where("library_id = ?", libraryID).
		Exec(context.Background()); err != nil {
		return fmt.Errorf("failed to delete detailed media: %w", err)
	}
	return nil
}
