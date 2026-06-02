package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/kingbenny101/kbarr/shared/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
)

var DB *bun.DB

func Init() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return fmt.Errorf("DB_URL environment variable is required")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbURL)))
	DB = bun.NewDB(sqldb, pgdialect.New())

	if os.Getenv("KBARR_ENV") == "dev" {
		DB.AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	}

	ctx := context.Background()
	var err error

	const maxAttempts = 120
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = DB.DB.PingContext(ctx)
		if err == nil {
			break
		}

		slog.Warn("PostgreSQL not ready", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
	}

	slog.Info("Connected to PostgreSQL")

	DB.RegisterModel((*Medium)(nil), (*Detailed)(nil), (*Episode)(nil), (*Monitor)(nil), (*DownloadQueue)(nil), (*Setting)(nil), (*TorrentBlacklist)(nil))

	models := []any{
		(*Medium)(nil),
		(*Detailed)(nil),
		(*Episode)(nil),
		(*Monitor)(nil),
		(*DownloadQueue)(nil),
		(*Setting)(nil),
		(*TorrentBlacklist)(nil),
	}
	for _, model := range models {
		if _, err := DB.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("failed to create table for %T: %w", model, err)
		}
	}

	slog.Info("Tables ready")

	if err := runMigrations(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	slog.Info("Migrations applied")

	// TODO: update shared/config.EnsureDefaults to accept *bun.DB instead of *sql.DB.
	if err := config.EnsureDefaults(DB); err != nil {
		return fmt.Errorf("failed to initialize settings defaults: %w", err)
	}
	slog.Info("Settings defaults ready")

	return nil
}
