package middleware

import (
	"context"
	"log/slog"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/partnerbot/i18n"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// ctxKey is a private type for context keys to avoid collisions.
type ctxKey string

const (
	userCtxKey ctxKey = "user"
	langCtxKey ctxKey = "lang"
)

// UserFromContext extracts the domain.User from the request context.
func UserFromContext(ctx context.Context) *domain.User {
	u, _ := ctx.Value(userCtxKey).(*domain.User)
	return u
}

// LangFromContext extracts the user language from context.
func LangFromContext(ctx context.Context) i18n.Lang {
	l, ok := ctx.Value(langCtxKey).(i18n.Lang)
	if !ok {
		return i18n.LangRU
	}
	return l
}

// Auth creates a middleware for the Partner bot.
// Registers new users with bot_type="partner" and blocks pending users.
// The adminID user is auto-promoted to admin so their approve/reject callbacks work.
func Auth(userRepo *storage.UserRepo, adminID int64) bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			tgUser := extractTelegramUser(update)
			if tgUser == nil {
				next(ctx, b, update)
				return
			}

			user, _, err := userRepo.GetOrCreatePartner(ctx, tgUser.ID, tgUser.Username, fullName(tgUser))
			if err != nil {
				slog.Error("partner auth: get or create user", "error", err, "telegram_id", tgUser.ID)
				next(ctx, b, update)
				return
			}

			// Auto-promote admin by Telegram ID (needed for approve/reject callbacks)
			if tgUser.ID == adminID && user.Role != domain.RoleAdmin {
				if err := userRepo.SetRole(ctx, user.ID, domain.RoleAdmin); err != nil {
					slog.Error("partner auth: failed to promote admin", "error", err)
				}
				user.Role = domain.RoleAdmin
			}

			// Inject lang into context
			lang := i18n.ParseLang(user.Lang)
			ctx = context.WithValue(ctx, langCtxKey, lang)

				// Allow /start and /cancel for everyone (including pending and onboarding)
			if update.Message != nil && (update.Message.Text == "/start" || update.Message.Text == "/cancel") {
				ctx = context.WithValue(ctx, userCtxKey, user)
				next(ctx, b, update)
				return
			}

			// User is in an onboarding step — allow text messages and pcompany: callbacks through
			if user.OnboardStep != "" {
				// Allow pcompany: callbacks (company suggestion buttons)
				if update.CallbackQuery != nil && strings.HasPrefix(update.CallbackQuery.Data, "pcompany:") {
					ctx = context.WithValue(ctx, userCtxKey, user)
					next(ctx, b, update)
					return
				}
				if update.Message == nil || update.Message.Text == "" {
					return // silently ignore non-text during onboarding
				}
				ctx = context.WithValue(ctx, userCtxKey, user)
				next(ctx, b, update)
				return
			}

			// Block pending users who have filed a request (waiting for admin)
			if user.Role == domain.RolePending {
				chatID := extractChatID(update)
				if chatID != 0 {
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: chatID,
						Text:   i18n.T("pending_approval", lang),
					})
				}
				return
			}

			ctx = context.WithValue(ctx, userCtxKey, user)
			next(ctx, b, update)
		}
	}
}

// Logging creates a logging middleware for the partner bot.
func Logging() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			if update.Message != nil {
				slog.Info("partner-bot update",
					"update_id", update.ID,
					"user_id", update.Message.From.ID,
					"username", update.Message.From.Username,
					"text", update.Message.Text,
				)
			} else if update.CallbackQuery != nil {
				slog.Info("partner-bot update",
					"update_id", update.ID,
					"user_id", update.CallbackQuery.From.ID,
					"username", update.CallbackQuery.From.Username,
					"callback_data", update.CallbackQuery.Data,
				)
			}
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
