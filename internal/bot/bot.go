package bot

import (
	"context"
	"log/slog"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/handlers"
	mw "github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/config"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Run initializes and starts the Telegram bot.
func Run(ctx context.Context, cfg *config.Config, db *storage.Postgres) error {
	// Create repositories
	partnerRepo := storage.NewPartnerRepo(db)
	userRepo := storage.NewUserRepo(db)

	// Create handlers
	searchHandler := handlers.NewSearchHandler(partnerRepo)
	partnerHandler := handlers.NewPartnerHandler(partnerRepo)
	adminHandler := handlers.NewAdminHandler(userRepo, partnerRepo)

	// Create bot options
	opts := []bot.Option{
		bot.WithMiddlewares(
			mw.Logging(),
			mw.Auth(userRepo, cfg.AdminTelegramID),
		),
		bot.WithDefaultHandler(makeDefaultHandler(searchHandler)),
	}

	// Initialize bot
	b, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		return err
	}

	// Register command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handlers.Start)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handlers.Help)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/search", bot.MatchTypePrefix, searchHandler.Handle)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, adminHandler.HandleStats)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/users", bot.MatchTypeExact, adminHandler.HandleUsers)

	// Register callback handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "partner:", bot.MatchTypePrefix, partnerHandler.HandleCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "approve:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "reject:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "stats:", bot.MatchTypePrefix, adminHandler.HandleStatsCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "menu:", bot.MatchTypePrefix, handlers.HandleMenuCallback(searchHandler, adminHandler))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "chart:", bot.MatchTypePrefix, adminHandler.HandleChartCallback)

	// Register bot commands for the "/" menu
	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{Command: "search", Description: "🔍 Поиск партнёра по имени"},
			{Command: "stats", Description: "📊 Аналитика CCA"},
			{Command: "users", Description: "👥 Управление пользователями"},
			{Command: "help", Description: "❓ Список команд"},
		},
	})

	slog.Info("bot started, listening for updates...")
	b.Start(ctx)

	return nil
}

// makeDefaultHandler creates a handler that forwards non-command text to search.
func makeDefaultHandler(searchHandler *handlers.SearchHandler) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.Text == "" {
			return
		}

		text := update.Message.Text
		// Skip if it looks like a command
		if len(text) > 0 && text[0] == '/' {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "❓ Неизвестная команда. Используйте /help для списка команд.",
			})
			return
		}

		// Forward to search handler — rewrite message as /search <text>
		update.Message.Text = "/search " + text
		searchHandler.Handle(ctx, b, update)
	}
}
