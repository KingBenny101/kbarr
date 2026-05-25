package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	dbgen "github.com/kingbenny101/kbarr/services/core/internal/db/generated"
	"github.com/kingbenny101/kbarr/shared/config"
)

var (
	Pool    *pgxpool.Pool
	DB      *sql.DB
	Queries *dbgen.Queries
)

func Init() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return fmt.Errorf("DB_URL environment variable is required")
	}

	ctx := context.Background()
	var err error

	const maxAttempts = 120
	connected := false
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		poolConfig, openErr := pgxpool.ParseConfig(dbURL)
		if openErr != nil {
			err = openErr
		} else {
			if poolConfig.ConnConfig.RuntimeParams == nil {
				poolConfig.ConnConfig.RuntimeParams = make(map[string]string)
			}
			poolConfig.ConnConfig.RuntimeParams["search_path"] = "public"

			pool, poolErr := pgxpool.NewWithConfig(ctx, poolConfig)
			if poolErr != nil {
				err = poolErr
			} else {
				err = pool.Ping(ctx)
				if err == nil {
					Pool = pool
					DB = stdlib.OpenDBFromPool(pool)
					connected = true
					break
				}
				pool.Close()
			}
		}

		slog.Warn("PostgreSQL not ready", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
		time.Sleep(1 * time.Second)
	}

	if !connected {
		return fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
	}

	slog.Info("Connected to PostgreSQL")

	if err := runMigrations(dbURL); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	Queries = dbgen.New(DB)

	slog.Info("Tables ready")

	if err := config.EnsureDefaults(DB); err != nil {
		return fmt.Errorf("failed to initialize settings defaults: %w", err)
	}
	slog.Info("Settings defaults ready")

	return nil
}

func runMigrations(dbURL string) error {
	migrationsPath, err := filepath.Abs("../../migrations/core")
	if err != nil {
		return fmt.Errorf("failed to resolve migrations path: %w", err)
	}

	migrationSource := "file://" + filepath.ToSlash(migrationsPath)
	m, err := migrate.New(migrationSource, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migration runner: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Warn("Failed to close migration source", "error", srcErr)
		}
		if dbErr != nil {
			slog.Warn("Failed to close migration database handle", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
