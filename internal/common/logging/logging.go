// Package logging provides structured logging setup for the application.
// It uses slog with environment-aware handlers (JSON for production, Text for development).
package logging

import (
	"log/slog"
	"os"
)

// Config holds the configuration for the application logger.
type Config struct {
	Level       string
	Environment string
	ServiceName string
}

// NewLogger initializes a new slog.Logger instance with the given configuration.
func NewLogger(cfg Config) *slog.Logger {
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
		slog.String("env", cfg.Environment),
	)
}
