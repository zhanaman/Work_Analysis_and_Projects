package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/partnerbot/i18n"
	"github.com/anonimouskz/pbm-partner-bot/internal/partnerbot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// PartnerHandlers holds all handlers for the Partner bot.
type PartnerHandlers struct {
	userRepo    *storage.UserRepo
	partnerRepo *storage.PartnerRepo
	adminChatID int64 // PBM admin Telegram ID for notifications
}

// New creates a new PartnerHandlers.
func New(userRepo *storage.UserRepo, partnerRepo *storage.PartnerRepo, adminChatID int64) *PartnerHandlers {
	return &PartnerHandlers{
		userRepo:    userRepo,
		partnerRepo: partnerRepo,
		adminChatID: adminChatID,
	}
}

// HandleStart handles the /start command.
func (h *PartnerHandlers) HandleStart(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	lang := middleware.LangFromContext(ctx)
	chatID := update.Message.Chat.ID

	if user == nil {
		return
	}

	// Already approved — welcome back
	if user.IsAuthorized() && user.PartnerID != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("welcome_back", lang),
		})
		return
	}

	// Pending with email — still waiting
	if user.Role == domain.RolePending && user.Email != "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("pending_approval", lang),
		})
		return
	}

	// New user — prompt for email
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   i18n.T("welcome", lang),
	})
}

// HandleHelp handles the /help command.
func (h *PartnerHandlers) HandleHelp(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := middleware.LangFromContext(ctx)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      i18n.T("help", lang),
		ParseMode: models.ParseModeHTML,
	})
}

// HandleStatus shows the partner's own company card.
func (h *PartnerHandlers) HandleStatus(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	lang := middleware.LangFromContext(ctx)
	chatID := update.Message.Chat.ID

	if user == nil || user.PartnerID == nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("no_partner", lang),
		})
		return
	}

	partner, err := h.partnerRepo.GetByID(ctx, *user.PartnerID)
	if err != nil {
		slog.Error("partner-bot: get partner", "error", err, "partner_id", *user.PartnerID)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Error loading partner data.",
		})
		return
	}

	card := formatPartnerCard(partner, "retention")
	buttons := [][]models.InlineKeyboardButton{
		{{Text: i18n.T("btn_upgrade", lang), CallbackData: fmt.Sprintf("pcard:%d:upgrade", partner.ID)}},
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

// HandleLang handles the /lang command — shows language selection keyboard.
func (h *PartnerHandlers) HandleLang(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	lang := middleware.LangFromContext(ctx)
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   i18n.T("lang_choose", lang),
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "🇷🇺 Русский", CallbackData: "lang:ru"},
					{Text: "🇬🇧 English", CallbackData: "lang:en"},
				},
			},
		},
	})
}

// HandleLangCallback handles language switch via inline button.
func (h *PartnerHandlers) HandleLangCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	tgID := update.CallbackQuery.From.ID

	newLang := strings.TrimPrefix(data, "lang:")
	if newLang != "ru" && newLang != "en" {
		return
	}

	if err := h.userRepo.SetLang(ctx, tgID, newLang); err != nil {
		slog.Error("partner-bot: set lang", "error", err)
	}

	lang := i18n.ParseLang(newLang)
	msgKey := "lang_switched_ru"
	if newLang == "en" {
		msgKey = "lang_switched_en"
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            i18n.T(msgKey, lang),
	})

	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      i18n.T(msgKey, lang),
		})
	}
}

// HandleCardCallback handles retention/upgrade toggle on partner card.
func (h *PartnerHandlers) HandleCardCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	lang := middleware.LangFromContext(ctx)

	// Parse "pcard:123:upgrade" or "pcard:123:retention"
	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}

	viewMode := parts[2]

	// Security: verify user owns this partner
	user := middleware.UserFromContext(ctx)
	if user == nil || user.PartnerID == nil || *user.PartnerID != id {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Access denied",
			ShowAlert:       true,
		})
		return
	}

	partner, err := h.partnerRepo.GetByID(ctx, id)
	if err != nil {
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	card := formatPartnerCard(partner, viewMode)

	var buttons [][]models.InlineKeyboardButton
	if viewMode == "retention" {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: i18n.T("btn_upgrade", lang), CallbackData: fmt.Sprintf("pcard:%d:upgrade", id)},
		})
	} else {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{Text: i18n.T("btn_retention", lang), CallbackData: fmt.Sprintf("pcard:%d:retention", id)},
		})
	}

	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      card,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			},
		})
	}
}

// HandleDefaultMessage handles free-text messages — used for email verification flow.
func (h *PartnerHandlers) HandleDefaultMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	user := middleware.UserFromContext(ctx)
	lang := middleware.LangFromContext(ctx)
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Skip commands
	if strings.HasPrefix(text, "/") {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❓ Unknown command. " + i18n.T("help", lang),
		})
		return
	}

	if user == nil {
		return
	}

	// If user hasn't linked email yet — treat input as email
	if user.Role == domain.RolePending && user.Email == "" {
		h.handleEmailInput(ctx, b, chatID, user, text, lang)
		return
	}
}

// handleEmailInput processes an email entered by a pending partner user.
func (h *PartnerHandlers) handleEmailInput(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, email string, lang i18n.Lang) {
	email = strings.ToLower(strings.TrimSpace(email))

	// Validate email format
	if !emailRegex.MatchString(email) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("email_invalid", lang),
		})
		return
	}

	// Try to match partner by email domain
	matches, err := h.userRepo.MatchPartnerByEmail(ctx, email)
	if err != nil || len(matches) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("email_company_not_found", lang),
		})
		return
	}

	if len(matches) == 1 {
		// Exact match — link and send for approval
		if err := h.userRepo.SetPartnerLink(ctx, user.ID, matches[0].ID, email); err != nil {
			slog.Error("partner-bot: set partner link", "error", err)
			return
		}
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("email_sent", lang),
		})
		// Notify PBM admin
		h.notifyAdminPartnerRequest(ctx, b, user, email, matches[0].Name, matches[0].ID)
		return
	}

	// Multiple matches — ask user to choose
	var rows [][]models.InlineKeyboardButton
	for _, m := range matches {
		label := m.Name
		if len(label) > 50 {
			label = label[:47] + "..."
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: label, CallbackData: fmt.Sprintf("pickpartner:%d:%s", m.ID, email)},
		})
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   i18n.T("email_multiple_companies", lang),
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	})
}

// HandlePickPartner handles the callback when user selects a company from multiple matches.
func (h *PartnerHandlers) HandlePickPartner(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	lang := middleware.LangFromContext(ctx)
	user := middleware.UserFromContext(ctx)
	if user == nil {
		return
	}

	// Parse "pickpartner:42:email@domain.com"
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return
	}

	partnerID, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	email := parts[2]

	if err := h.userRepo.SetPartnerLink(ctx, user.ID, partnerID, email); err != nil {
		slog.Error("partner-bot: set partner link (pick)", "error", err)
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// Get partner name for the notification
	partner, _ := h.partnerRepo.GetByID(ctx, partnerID)
	partnerName := "Unknown"
	if partner != nil {
		partnerName = partner.Name
	}

	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      i18n.T("email_sent", lang),
		})
	}

	h.notifyAdminPartnerRequest(ctx, b, user, email, partnerName, partnerID)
}

// notifyAdminPartnerRequest sends a notification to the PBM admin about a new partner request.
func (h *PartnerHandlers) notifyAdminPartnerRequest(ctx context.Context, b *bot.Bot, user *domain.User, email, partnerName string, partnerID int) {
	text := fmt.Sprintf(
		"🆕 <b>Partner verification request</b>\n\n"+
			"👤 %s (@%s)\n"+
			"📧 %s\n"+
			"🏢 %s\n\n"+
			"Approve or reject:",
		user.FullName, user.Username, email, partnerName,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    h.adminChatID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "✅ Approve", CallbackData: fmt.Sprintf("papprove:%d", user.ID)},
					{Text: "❌ Reject", CallbackData: fmt.Sprintf("preject:%d", user.ID)},
				},
			},
		},
	})
}
