package handlers

import (
	"context"
	"fmt"
	"html"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/rbac"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start handles the /start command for authorized users.
func Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAuthorized() {
		return
	}

	text := fmt.Sprintf("👋 Привет, <b>%s</b>!\n\n"+
		"Я помогу быстро найти информацию по любому партнёру.\n\n",
		html.EscapeString(user.FullName))

	text += rbac.HelpTextForUser(user)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

// Help handles the /help command — shows only accessible commands.
func Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := middleware.UserFromContext(ctx)

	text := "📖 <b>HPE Partner Advisor — Справка</b>\n\n"
	text += rbac.HelpTextForUser(user)
	text += "\n<i>Просто напишите имя партнёра, и я найду его!</i>"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
	})
}

// escapeMarkdownV2 escapes special characters for Telegram MarkdownV2.
func escapeMarkdownV2(s string) string {
	replacer := []string{
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	}
	result := s
	for i := 0; i < len(replacer); i += 2 {
		result = replaceAll(result, replacer[i], replacer[i+1])
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if string(s[i]) == old {
			result += new
		} else {
			result += string(s[i])
		}
	}
	return result
}
