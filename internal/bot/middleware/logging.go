package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Logging creates a middleware that logs every incoming update.
func Logging() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			start := time.Now()

			user := extractTelegramUser(update)
			attrs := []any{
				"update_id", update.ID,
			}
			if user != nil {
				attrs = append(attrs,
					"user_id", user.ID,
					"username", user.Username,
				)
			}
			if update.Message != nil && update.Message.Text != "" {
				text := update.Message.Text
				if len(text) > 50 {
					text = text[:50] + "..."
				}
				attrs = append(attrs, "text", text)
			}
			if update.CallbackQuery != nil {
				attrs = append(attrs, "callback_data", update.CallbackQuery.Data)
			}

			slog.Info("incoming update", attrs...)

			next(ctx, b, update)

			slog.Debug("update processed",
				"update_id", update.ID,
				"duration", time.Since(start),
			)
		}
	}
}
