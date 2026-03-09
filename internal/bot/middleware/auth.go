package middleware

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ctxKey is a private type for context keys to avoid collisions.
type ctxKey string

const userCtxKey ctxKey = "user"

// UserFromContext extracts the domain.User from the request context.
func UserFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userCtxKey).(*domain.User)
	return u
}

// Auth creates a middleware that loads the user from DB and injects it into context.
// Pending users get a "waiting for approval" message.
// Unknown users are auto-registered with "pending" role.
func Auth(userRepo *storage.UserRepo, adminID int64) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			// Extract telegram user from update
			tgUser := extractTelegramUser(update)
			if tgUser == nil {
				// No user info — skip auth (e.g., channel posts)
				next(ctx, b, update)
				return
			}

			// Get or create user in DB
			user, isNew, err := userRepo.GetOrCreate(ctx, tgUser.ID, tgUser.Username, fullName(tgUser))
			if err != nil {
				slog.Error("auth middleware: get or create user", "error", err, "telegram_id", tgUser.ID)
				next(ctx, b, update)
				return
			}

			// Auto-promote admin by Telegram ID
			if tgUser.ID == adminID && user.Role != domain.RoleAdmin {
				if err := userRepo.SetRole(ctx, user.ID, domain.RoleAdmin); err != nil {
				slog.Error("auth middleware: failed to promote admin", "error", err, "telegram_id", tgUser.ID)
			}
				user.Role = domain.RoleAdmin
			}

			// Notify admin about new user registrations
			if isNew && tgUser.ID != adminID {
				slog.Info("new user registered",
					"telegram_id", tgUser.ID,
					"username", tgUser.Username,
					"full_name", fullName(tgUser),
				)
				notifyAdminNewUser(ctx, b, adminID, tgUser)
			}

			// Block pending users (except /start)
			if user.Role == domain.RolePending {
				if update.Message != nil && update.Message.Text == "/start" {
					// Allow /start to go through for pending users
				} else {
					chatID := extractChatID(update)
					if chatID != 0 {
						b.SendMessage(ctx, &bot.SendMessageParams{
							ChatID: chatID,
							Text:   "⏳ Ваш запрос на доступ ожидает одобрения администратора.",
						})
					}
					return
				}
			}

			// Inject user into context
			ctx = context.WithValue(ctx, userCtxKey, user)
			next(ctx, b, update)
		}
	}
}

func extractTelegramUser(update *models.Update) *models.User {
	if update.Message != nil {
		return update.Message.From
	}
	if update.CallbackQuery != nil {
		return &update.CallbackQuery.From
	}
	return nil
}

func extractChatID(update *models.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		if update.CallbackQuery.Message.Message != nil {
			return update.CallbackQuery.Message.Message.Chat.ID
		}
	}
	return 0
}

func fullName(u *models.User) string {
	name := u.FirstName
	if u.LastName != "" {
		name += " " + u.LastName
	}
	return name
}

func notifyAdminNewUser(ctx context.Context, b *bot.Bot, adminID int64, u *models.User) {
	text := "🆕 *Новый пользователь запрашивает доступ:*\n\n" +
		"👤 " + fullName(u) + "\n" +
		"🔗 @" + u.Username + "\n" +
		"🆔 `" + fmt.Sprintf("%d", u.ID) + "`"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    adminID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
	})
}
