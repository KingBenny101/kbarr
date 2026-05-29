package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var DB *bun.DB

func Init() error {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return fmt.Errorf("DB_URL environment variable is required")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dbURL)))
	DB = bun.NewDB(sqldb, pgdialect.New())

	ctx := context.Background()
	var err error
	const maxAttempts = 120
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = DB.DB.PingContext(ctx)
		if err == nil {
			slog.Info("Connected to PostgreSQL")
			return nil
		}

		slog.Warn("PostgreSQL not ready", "attempt", attempt, "maxAttempts", maxAttempts, "error", err)
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("failed to connect to database after %d attempts: %w", maxAttempts, err)
}
