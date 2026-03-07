package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbot "github.com/anonimouskz/pbm-partner-bot/internal/bot"
	"github.com/anonimouskz/pbm-partner-bot/internal/config"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup structured logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("starting PBM Partner Bot",
		"log_level", cfg.LogLevel.String(),
	)

	// Create context with graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to PostgreSQL
	db, err := storage.NewPostgres(ctx, cfg.PostgresDSN)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Start the bot
	if err := tgbot.Run(ctx, cfg, db); err != nil {
		slog.Error("bot error", "error", err)
		os.Exit(1)
	}

	slog.Info("bot stopped gracefully")
}
