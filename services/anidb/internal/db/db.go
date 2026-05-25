package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var (
	Pool  *pgxpool.Pool
	SQLDB *sql.DB
)

func Init() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return fmt.Errorf("DB_URL environment variable is required")
	}

	ctx := context.Background()
	var err error

	const maxAttempts = 120
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
					SQLDB = stdlib.OpenDBFromPool(pool)
					slog.Info("Connected to PostgreSQL")
					return nil
				}
				pool.Close()
			}
		}

		slog.Warn("PostgreSQL not ready", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to open database after %d attempts: %w", maxAttempts, err)
}
