package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// SearchHandler holds dependencies for the search handler.
type SearchHandler struct {
	partnerRepo  *storage.PartnerRepo
	activityRepo *storage.ActivityRepo
}

// NewSearchHandler creates a new SearchHandler.
func NewSearchHandler(repo *storage.PartnerRepo, activityRepo *storage.ActivityRepo) *SearchHandler {
	return &SearchHandler{partnerRepo: repo, activityRepo: activityRepo}
}

// Handle processes the /search command.
func (h *SearchHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	query := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/search"))
	if query == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "🔍 Укажите имя партнёра для поиска.\n\nПример: <code>/search Kaspersky</code>\nИли просто напишите имя партнёра.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// Get region filter from user context
	regionFilter := ""
	if user := middleware.UserFromContext(ctx); user != nil {
		regionFilter = user.RegionFilter
	}

	partners, err := h.partnerRepo.Search(ctx, query, regionFilter)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка при поиске. Попробуйте позже.",
		})
		return
	}

	// Log search activity (fire-and-forget)
	if user := middleware.UserFromContext(ctx); user != nil {
		var uid *int
		if user.ID != 0 {
			id := user.ID
			uid = &id
		}
		h.activityRepo.Log(ctx, uid, update.Message.Chat.ID, domain.EventSearch, query, nil, "")
	}

	if len(partners) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      fmt.Sprintf("🔍 По запросу <b>%s</b> ничего не найдено.\n\n💡 Попробуйте другое имя или часть названия.", escapeHTML(query)),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// Build inline keyboard with results
	var rows [][]models.InlineKeyboardButton
	for _, p := range partners {
		tierInfo := bestMembership(p.MembershipCompute, p.MembershipHC, p.MembershipNetworking)
		label := fmt.Sprintf("%s • %s • %s", p.Name, tierInfo, p.Country)
		if len(label) > 60 {
			label = label[:57] + "..."
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         label,
				CallbackData: "partner:" + strconv.Itoa(p.ID),
			},
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      fmt.Sprintf("🔍 Найдено <b>%d</b> по запросу <b>%s</b>:", len(partners), escapeHTML(query)),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	})
}

// bestMembership returns the highest tier across all 3 centers.
func bestMembership(compute, hc, networking string) string {
	bestOrder := -1
	for _, m := range []string{compute, hc, networking} {
		m = strings.ToLower(m)
		order := 0
		switch {
		case strings.Contains(m, "platinum"):
			order = 4
		case strings.Contains(m, "gold"):
			order = 3
		case strings.Contains(m, "silver"):
			order = 2
		case strings.Contains(m, "business"):
			order = 1
		}
		if order > bestOrder {
			bestOrder = order
		}
	}
	switch bestOrder {
	case 4:
		return "💎 Platinum"
	case 3:
		return "🥇 Gold"
	case 2:
		return "🥈 Silver"
	case 1:
		return "🏷 BP"
	default:
		return "—"
	}
}

// escapeHTML escapes <, >, & for Telegram HTML mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
