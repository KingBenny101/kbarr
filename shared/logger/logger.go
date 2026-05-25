package logger

import (
	"log/slog"
	"os"
)

func Init() {
	level := slog.LevelInfo
	if os.Getenv("KBARR_LOG_LEVEL") == "debug" {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}
