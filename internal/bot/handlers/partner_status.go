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

	partner, err := h.resolvePartner(ctx, user)
	if err != nil || partner == nil {
		// Company not found in DB
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
		return
	}

	card := formatPartnerCard(partner, "retention")

	// Inline buttons — same as PBM view
	buttons := [][]models.InlineKeyboardButton{
		{
			{Text: "📈 Как повысить статус?", CallbackData: fmt.Sprintf("partner:%d:upgrade", partner.ID)},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      card,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
}

// resolvePartner finds the partner record with full Tiers/Revenue data.
// Tries PartnerID first (exact link), then falls back to name search + GetByID.
func (h *PartnerStatusHandler) resolvePartner(ctx context.Context, user *domain.User) (*domain.Partner, error) {
	// Path 1: user has an explicit PartnerID link
	if user.PartnerID != nil {
		return h.partnerRepo.GetByID(ctx, *user.PartnerID)
	}

	// Path 2: search by company name, then load full data via GetByID
	if user.CompanyName == "" {
		return nil, nil
	}

	results, err := h.partnerRepo.Search(ctx, user.CompanyName, "")
	if err != nil || len(results) == 0 {
		return nil, err
	}

	// Prefer exact name match
	var matchID int
	for _, p := range results {
		if p.Name == user.CompanyName {
			matchID = p.ID
			break
		}
	}
	// Fallback to closest similarity match
	if matchID == 0 {
		matchID = results[0].ID
	}

	// Load full data (Tiers, Revenue, Competencies, etc.)
	return h.partnerRepo.GetByID(ctx, matchID)
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
