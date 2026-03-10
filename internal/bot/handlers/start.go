package handlers

import (
	"context"
	"fmt"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start handles the /start command.
func Start(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := middleware.UserFromContext(ctx)

	var text string
	if user == nil || !user.IsAuthorized() {
		text = "👋 Добро пожаловать в *HPE Partner Advisor*\\!\n\n" +
			"Ваш запрос на доступ отправлен администратору\\.\n" +
			"Ожидайте подтверждения\\."
	} else {
		text = fmt.Sprintf("👋 Привет, *%s*\\!\n\n"+
			"Я помогу быстро найти информацию по любому партнёру\\.\n\n"+
			"📋 *Команды:*\n"+
			"/search `<имя>` — поиск партнёра\n"+
			"/help — справка\n"+
			"/stats — статистика по базе",
			escapeMarkdownV2(user.FullName),
		)
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
}

// Help handles the /help command.
func Help(ctx context.Context, b *bot.Bot, update *models.Update) {
	text := "📖 *HPE Partner Advisor — Справка*\n\n" +
		"🔍 /search `<имя>` — поиск партнёра по имени\n" +
		"📊 /stats — статистика по базе партнёров\n" +
		"ℹ️ /help — эта справка\n\n" +
		"_Просто напишите имя партнёра, и я найду его\\!_"

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
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
