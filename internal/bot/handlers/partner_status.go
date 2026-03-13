package handlers

import (
	"context"
	"fmt"
	"html"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/rbac"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// PartnerStatusHandler handles /status command for Partner users.
// Shows the partner's own company card using their registered company.
type PartnerStatusHandler struct {
	partnerRepo *storage.PartnerRepo
}

// NewPartnerStatusHandler creates a new PartnerStatusHandler.
func NewPartnerStatusHandler(partnerRepo *storage.PartnerRepo) *PartnerStatusHandler {
	return &PartnerStatusHandler{partnerRepo: partnerRepo}
}

// Handle handles the /status command.
func (h *PartnerStatusHandler) Handle(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !rbac.Can(user, rbac.ViewOwnCard) {
		return
	}

	chatID := update.Message.Chat.ID

	// Use PartnerID if available (linked partner record)
	if user.PartnerID != nil {
		partner, err := h.partnerRepo.GetByID(ctx, *user.PartnerID)
		if err == nil {
			card := formatPartnerCard(partner, "retention")
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      card,
				ParseMode: models.ParseModeHTML,
			})
			return
		}
	}

	// Fallback: search by company name
	if user.CompanyName != "" {
		partners, err := h.partnerRepo.Search(ctx, user.CompanyName, "")
		if err == nil && len(partners) > 0 {
			// Find exact match first
			for _, p := range partners {
				if p.Name == user.CompanyName {
					card := formatPartnerCard(&p, "retention")
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID:    chatID,
						Text:      card,
						ParseMode: models.ParseModeHTML,
					})
					return
				}
			}
			// Show closest match
			card := formatPartnerCard(&partners[0], "retention")
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      "🏢 <b>Данные вашей компании</b>\n\n" + card,
				ParseMode: models.ParseModeHTML,
			})
			return
		}
	}

	// Company not found in DB yet
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: fmt.Sprintf(
			"🏢 <b>Ваша компания:</b> %s\n\n"+
				"<i>Данные компании ещё не загружены в базу. "+
				"Обратитесь к администратору.</i>",
			html.EscapeString(user.CompanyName),
		),
		ParseMode: models.ParseModeHTML,
	})
}

// RoleAwareFooter returns a context-appropriate footer line for /help and /start.
func RoleAwareFooter(user *domain.User) string {
	if user == nil {
		return ""
	}
	switch user.Role {
	case domain.RoleUser:
		return "\n<i>Используйте /status чтобы посмотреть данные вашей компании.</i>"
	case domain.RolePBM:
		return "\n<i>Просто напишите имя партнёра, и я найду его в базе!</i>"
	case domain.RoleAdmin:
		return "\n<i>Полный доступ: поиск, аналитика, управление пользователями.</i>"
	case domain.RoleDistri:
		return "\n<i>Используйте /status для просмотра данных вашей компании.</i>"
	default:
		return ""
	}
}
