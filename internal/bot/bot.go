package bot

import (
	"context"
	"log/slog"
	"os"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/handlers"
	mw "github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/config"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/rbac"
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
	partnerBotToken := os.Getenv("PARTNER_BOT_TOKEN")
	searchHandler := handlers.NewSearchHandler(partnerRepo)
	partnerHandler := handlers.NewPartnerHandler(partnerRepo)
	adminHandler := handlers.NewAdminHandler(userRepo, partnerRepo, partnerBotToken)
	onboardHandler := handlers.NewOnboardingHandler(userRepo, cfg.AdminTelegramID)

	// Create bot options
	opts := []bot.Option{
		bot.WithMiddlewares(
			mw.Logging(),
			mw.Auth(userRepo, cfg.AdminTelegramID),
		),
		bot.WithDefaultHandler(makeDefaultHandler(searchHandler, adminHandler, onboardHandler)),
	}

	// Initialize bot
	b, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		return err
	}

	// Register command handlers
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, makeStartHandler(onboardHandler))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/cancel", bot.MatchTypeExact, makeCancelHandler(userRepo))
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handlers.Help)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/search", bot.MatchTypePrefix, searchHandler.Handle)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stats", bot.MatchTypeExact, adminHandler.HandleStats)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/users", bot.MatchTypeExact, adminHandler.HandleUsers)

	// Register callback handlers
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "onboard_role:", bot.MatchTypePrefix, onboardHandler.HandleRoleCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "partner:", bot.MatchTypePrefix, partnerHandler.HandleCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "approve:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "role_pbm:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "role_distri:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "reject:", bot.MatchTypePrefix, adminHandler.HandleApproveCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "stats:", bot.MatchTypePrefix, adminHandler.HandleStatsCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "menu:", bot.MatchTypePrefix, handlers.HandleMenuCallback(searchHandler, adminHandler))
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "chart:", bot.MatchTypePrefix, adminHandler.HandleChartCallback)

	// Partner approval callbacks (sent via PBM bot token from partner-bot)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "papprove:", bot.MatchTypePrefix, adminHandler.HandlePartnerApproval)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "pdistri:", bot.MatchTypePrefix, adminHandler.HandlePartnerApproval)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "preject:", bot.MatchTypePrefix, adminHandler.HandlePartnerApproval)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "prejectconfirm:", bot.MatchTypePrefix, adminHandler.HandlePartnerRejectConfirm)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "region:", bot.MatchTypePrefix, adminHandler.HandleRegionCallback)

	// Active user management callbacks
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "chrole:", bot.MatchTypePrefix, adminHandler.HandleRoleChangeMenu)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "chset:", bot.MatchTypePrefix, adminHandler.HandleRoleChangeApply)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "revokeyes:", bot.MatchTypePrefix, adminHandler.HandleRevokeCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "revokeno:", bot.MatchTypePrefix, adminHandler.HandleRevokeCancelCallback)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "revoke:", bot.MatchTypePrefix, adminHandler.HandleRevokeCallback)

	// Register bot commands for the "/" menu
	// Default (all users) = minimal commands (search + help)
	defaultUser := &domain.User{Role: domain.RoleUser}
	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: rbac.TelegramCommandsForUser(defaultUser),
	})
	// Admin gets expanded menu via per-chat scope
	adminUser := &domain.User{Role: domain.RoleAdmin}
	b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: rbac.TelegramCommandsForUser(adminUser),
		Scope:    &models.BotCommandScopeChat{ChatID: cfg.AdminTelegramID},
	})

	slog.Info("bot started, listening for updates...")
	b.Start(ctx)

	return nil
}

// makeStartHandler creates a /start handler that triggers onboarding for new users.
func makeStartHandler(onboardHandler *handlers.OnboardingHandler) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		user := mw.UserFromContext(ctx)
		if user == nil {
			return
		}

		// If user is in onboarding, pending, or rejected → start/restart onboarding
		if user.OnboardStep != "" || user.Role == domain.RolePending || user.Role == domain.RoleRejected {
			onboardHandler.StartOnboarding(ctx, b, update.Message.Chat.ID, user.ID)
			return
		}

		// Authorized user — show normal start
		handlers.Start(ctx, b, update)
	}
}

// makeCancelHandler creates a /cancel handler that resets onboarding.
func makeCancelHandler(userRepo *storage.UserRepo) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		user := mw.UserFromContext(ctx)
		if user == nil {
			return
		}
		if user.OnboardStep != "" {
			userRepo.ResetOnboard(ctx, user.ID)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "🚫 Регистрация отменена. Нажмите /start чтобы начать заново.",
			})
		}
	}
}

// makeDefaultHandler creates a handler that routes onboarding messages or forwards to search.
func makeDefaultHandler(searchHandler *handlers.SearchHandler, adminHandler *handlers.AdminHandler, onboardHandler *handlers.OnboardingHandler) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.Text == "" {
			return
		}

		user := mw.UserFromContext(ctx)

		// Route onboarding messages
		if user != nil && user.OnboardStep != "" {
			onboardHandler.HandleOnboardingMessage(ctx, b, update)
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

		// Check if admin is typing a rejection comment
		if adminHandler.TryHandleRejectComment(ctx, b, update) {
			return
		}

		// Forward to search handler — rewrite message as /search <text>
		update.Message.Text = "/search " + text
		searchHandler.Handle(ctx, b, update)
	}
}
