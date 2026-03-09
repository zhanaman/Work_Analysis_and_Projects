package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/anonimouskz/pbm-partner-bot/internal/partnerbot/handlers"
	mw "github.com/anonimouskz/pbm-partner-bot/internal/partnerbot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	// Load configuration
	token := os.Getenv("PARTNER_BOT_TOKEN")
	if token == "" {
		slog.Error("PARTNER_BOT_TOKEN is required")
		os.Exit(1)
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		slog.Error("POSTGRES_DSN is required")
		os.Exit(1)
	}

	adminIDStr := os.Getenv("ADMIN_TELEGRAM_ID")
	adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)

	pbmToken := os.Getenv("TELEGRAM_BOT_TOKEN") // PBM bot token for cross-bot notifications

	// Setup logging
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))

	slog.Info("starting Partner Self-Service Bot")

	// Create context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Connect to PostgreSQL
	db, err := storage.NewPostgres(ctx, dsn)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("connected to PostgreSQL")

	// Create repositories
	userRepo := storage.NewUserRepo(db)
	partnerRepo := storage.NewPartnerRepo(db)

	// Create handlers
	h := handlers.New(userRepo, partnerRepo, adminID, pbmToken)

	// Create bot
	opts := []bot.Option{
		bot.WithMiddlewares(
			mw.Logging(),
			mw.Auth(userRepo, adminID),
		),
		bot.WithDefaultHandler(h.HandleDefaultMessage),
	}

	b, err := bot.New(token, opts...)
	if err != nil {
		slog.Error("failed to create bot", "error", err)
		os.Exit(1)
	}

	// Register command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.HandleStart)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, h.HandleHelp)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/status", bot.MatchTypeExact, h.HandleStatus)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/lang", bot.MatchTypeExact, h.HandleLangCallback)

	// Register callback handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "pcard:", bot.MatchTypePrefix, h.HandleCardCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "lang:", bot.MatchTypePrefix, h.HandleLangCallback)
	// papprove:/preject:/prejectconfirm: callbacks handled by PBM bot

	// Register bot commands for the "/" menu
	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "status", Description: "📊 Карточка компании / Company card"},
			{Command: "lang", Description: "🌐 Язык / Language"},
			{Command: "help", Description: "❓ Помощь / Help"},
		},
	})

	slog.Info("partner bot started, listening for updates...")
	b.Start(ctx)

	slog.Info("partner bot stopped gracefully")
}
