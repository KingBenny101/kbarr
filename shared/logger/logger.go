package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

func Init() {
	debug := os.Getenv("KBARR_LOG_LEVEL") == "debug"

	level := zapcore.InfoLevel
	if debug {
		level = zapcore.DebugLevel
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.TimeKey = "time"
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006/01/02 15:04:05")
	config.EncoderConfig.CallerKey = ""

	base, _ := config.Build()
	Log = base.Sugar()
}
