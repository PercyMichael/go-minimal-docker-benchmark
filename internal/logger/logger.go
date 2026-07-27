package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

// InitLogger initializes structured JSON logging using Go standard library log/slog
func InitLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	Log = slog.New(handler)
	slog.SetDefault(Log)
}
