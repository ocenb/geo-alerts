package logger

import (
	"io"
	"log/slog"
	"os"

	"github.com/ocenb/geo-alerts/internal/config"
)

func New(cfg config.LogConfig, environment string) *slog.Logger {
	logLevel := slog.Level(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: environment == "local" || logLevel == slog.LevelDebug,
	}

	var handler slog.Handler

	if cfg.Handler == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(slog.String("env", environment))
}

func NewDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))

}
