package config

import (
	"log/slog"
	"os"
	"strings"
)

func InitLogger() {
	level := slog.LevelInfo
	if envLevel := strings.TrimSpace(strings.ToUpper(os.Getenv("LOG_LEVEL"))); envLevel != "" {
		switch envLevel {
		case "DEBUG":
			level = slog.LevelDebug
		case "INFO":
			level = slog.LevelInfo
		case "WARN":
			level = slog.LevelWarn
		case "ERROR":
			level = slog.LevelError
		}
	} else if os.Getenv("APP_ENV") != "production" {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))
	slog.SetDefault(logger)
}
