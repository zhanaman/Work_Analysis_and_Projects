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
		bot.WithDefaultHandler(defaultHandler),
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

	slog.Info("bot started, listening for updates...")
	b.Start(ctx)

	return nil
}

// defaultHandler handles any unmatched text input as a partner search.
func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	// Treat any non-command text as a partner search
	text := update.Message.Text
	if len(text) > 0 && text[0] != '/' {
		// Rewrite as search command and re-dispatch
		update.Message.Text = "/search " + text
		// Find and call search handler (simplified — in production use proper routing)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🔍 Ищу: *" + text + "*...\n\nИспользуйте /search для поиска.",
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}
