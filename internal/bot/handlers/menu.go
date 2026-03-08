package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandleMenuCallback returns a handler for inline menu buttons (menu:search, menu:stats).
func HandleMenuCallback(search *SearchHandler, admin *AdminHandler) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		if update.CallbackQuery == nil {
			return
		}

		action := strings.TrimPrefix(update.CallbackQuery.Data, "menu:")

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		var chatID int64
		var msgID int
		if update.CallbackQuery.Message.Message != nil {
			chatID = update.CallbackQuery.Message.Message.Chat.ID
			msgID = update.CallbackQuery.Message.Message.ID
		}
		if chatID == 0 {
			return
		}

		switch action {
		case "search":
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: msgID,
				Text:      "🔍 Напишите имя партнёра для поиска:",
				ParseMode: models.ParseModeHTML,
			})
		case "stats":
			admin.handleStatsDashboardEdit(ctx, b, chatID, msgID)
		}
	}
}

// handleStatsDashboardEdit renders the stats hub by editing a message.
func (h *AdminHandler) handleStatsDashboardEdit(ctx context.Context, b *bot.Bot, chatID int64, msgID int) {
	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAuthorized() {
		return
	}

	count, _ := h.partnerRepo.CountAll(ctx)
	dist, _ := h.partnerRepo.TierDistribution(ctx, "")

	var sb strings.Builder
	sb.WriteString("📊 <b>CCA Dashboard</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("🏢 <b>%d</b> партнёров  •  8 стран\n", count))
	sb.WriteString(fmt.Sprintf("💎 %d Platinum  •  🥇 %d Gold  •  🥈 %d Silver\n",
		dist["compute"]["platinum"],
		dist["compute"]["gold"],
		dist["compute"]["silver"]))

	buttons := statsButtons()

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: msgID,
		Text:      sb.String(),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
}

func statsButtons() [][]models.InlineKeyboardButton {
	return [][]models.InlineKeyboardButton{
		{
			{Text: "🌍 Страны", CallbackData: "stats:countries"},
			{Text: "🏅 Тиры", CallbackData: "stats:tiers"},
		},
		{
			{Text: "📈 Gaps", CallbackData: "stats:upgrade"},
			{Text: "💰 Top Volume", CallbackData: "stats:volume"},
		},
	}
}
