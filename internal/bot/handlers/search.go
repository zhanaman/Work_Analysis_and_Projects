package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// SearchHandler holds dependencies for the search handler.
type SearchHandler struct {
	partnerRepo *storage.PartnerRepo
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(repo *storage.PartnerRepo) *SearchHandler {
	return &SearchHandler{partnerRepo: repo}
}

// Handle processes the /search command.
func (h *SearchHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	query := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/search"))
	if query == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🔍 Укажите имя партнёра для поиска.\n\nПример: `/search Kaspersky`",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	}

	partners, err := h.partnerRepo.Search(ctx, query)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка при поиске. Попробуйте позже.",
		})
		return
	}

	if len(partners) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("🔍 По запросу *%s* ничего не найдено.", escapeMarkdownV2(query)),
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	// Build inline keyboard with results
	var rows [][]models.InlineKeyboardButton
	for _, p := range partners {
		label := fmt.Sprintf("%s • %s • %s", p.Name, p.Tier, p.Country)
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         label,
				CallbackData: "partner:" + strconv.Itoa(p.ID),
			},
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("🔍 Найдено *%d* партнёров по запросу *%s*:", len(partners), escapeMarkdownV2(query)),
		ParseMode: models.ParseModeMarkdown,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	})
}
