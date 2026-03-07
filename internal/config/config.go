package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration.
type Config struct {
	TelegramToken   string
	AdminTelegramID int64
	PostgresDSN     string
	LogLevel        slog.Level
}

// Load reads configuration from environment variables.
// It returns an error if any required variable is missing.
func Load() (*Config, error) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_BOT_TOKEN is required")
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	adminIDStr := os.Getenv("ADMIN_TELEGRAM_ID")
	if adminIDStr == "" {
		return nil, fmt.Errorf("ADMIN_TELEGRAM_ID is required")
	}
	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("ADMIN_TELEGRAM_ID must be a number: %w", err)
	}

	logLevel := parseLogLevel(os.Getenv("LOG_LEVEL"))

	return &Config{
		TelegramToken:   token,
		AdminTelegramID: adminID,
		PostgresDSN:     dsn,
		LogLevel:        logLevel,
	}, nil
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
