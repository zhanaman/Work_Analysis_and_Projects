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
	"github.com/anonimouskz/pbm-partner-bot/internal/shared/card"
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

	// Pending with email — still waiting for admin
	if user.Role == domain.RolePending && user.Email != "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   i18n.T("pending_approval", lang),
		})
		return
	}

	// New user or reset — start onboarding: Step 1 (full name)
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "👋 Добро пожаловать в Partner Performance Bot!\n\n📝 <b>Шаг 1/3</b>\nВведите ваше имя и фамилию:",
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Error("partner-bot: send start message", "error", err)
		return
	}

	// Save step + message ID for inline editing
	h.userRepo.SetOnboardData(ctx, user.ID, "name", "", "", "")
	h.userRepo.SetOnboardMsgID(ctx, user.ID, msg.ID)
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
			Text:   "❌ Error loading partner data",
		})
		return
	}

	cardText := card.FormatPartnerCard(partner, "retention")
	buttons := [][]models.InlineKeyboardButton{
		{{Text: i18n.T("btn_upgrade", lang), CallbackData: fmt.Sprintf("pcard:%d:upgrade", partner.ID)}},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      cardText,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
}

// HandleLangCallback handles the language switch callback.
func (h *PartnerHandlers) HandleLangCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	user := middleware.UserFromContext(ctx)
	if user == nil {
		return
	}

	newLang := strings.TrimPrefix(data, "lang:")
	if newLang != "ru" && newLang != "en" {
		return
	}

	if err := h.userRepo.SetLang(ctx, user.ID, newLang); err != nil {
		slog.Error("partner-bot: set lang", "error", err)
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            fmt.Sprintf("🌐 Language: %s", newLang),
	})
}

// HandleCardCallback handles partner card view mode toggle.
func (h *PartnerHandlers) HandleCardCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	lang := middleware.LangFromContext(ctx)
	if user == nil || user.PartnerID == nil {
		return
	}

	data := update.CallbackQuery.Data
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return
	}

	partnerID, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}

	// Validate viewMode
	viewMode := parts[2]
	if viewMode != "retention" && viewMode != "upgrade" {
		viewMode = "retention"
	}

	// Security: only own partner
	if *user.PartnerID != partnerID {
		return
	}

	partner, err := h.partnerRepo.GetByID(ctx, partnerID)
	if err != nil {
		slog.Error("partner-bot: get partner for card", "error", err)
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	cardText := card.FormatPartnerCard(partner, viewMode)

	var buttons [][]models.InlineKeyboardButton
	if viewMode == "retention" {
		buttons = [][]models.InlineKeyboardButton{
			{{Text: i18n.T("btn_upgrade", lang), CallbackData: fmt.Sprintf("pcard:%d:upgrade", partner.ID)}},
		}
	} else {
		buttons = [][]models.InlineKeyboardButton{
			{{Text: i18n.T("btn_retention", lang), CallbackData: fmt.Sprintf("pcard:%d:retention", partner.ID)}},
		}
	}

	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      cardText,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			},
		})
	}
}

// HandleDefaultMessage handles free-text messages — used for onboarding flow.
func (h *PartnerHandlers) HandleDefaultMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	user := middleware.UserFromContext(ctx)
	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Skip commands
	if strings.HasPrefix(text, "/") {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❓ Неизвестная команда. Используйте /help",
		})
		return
	}

	if user == nil {
		return
	}

	// Delete the user's input message to keep chat clean
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: update.Message.ID,
	})

	// State machine for onboarding
	switch user.OnboardStep {
	case "name":
		h.onboardStepName(ctx, b, chatID, user, text)
	case "company":
		h.onboardStepCompany(ctx, b, chatID, user, text)
	case "email":
		h.onboardStepEmail(ctx, b, chatID, user, text)
	}
}

// onboardStepName processes Step 1: full name input.
func (h *PartnerHandlers) onboardStepName(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, name string) {
	if len(name) < 2 || len(name) > 100 {
		return // ignore invalid
	}

	// Save name, advance to step 2
	h.userRepo.SetOnboardData(ctx, user.ID, "company", name, "", "")

	// Edit the bot's onboarding message
	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("📝 <b>Шаг 2/3</b>\n"+
				"✅ Имя: %s\n\n"+
				"Введите название вашей компании:", name),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// onboardStepCompany processes Step 2: company name input.
func (h *PartnerHandlers) onboardStepCompany(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, company string) {
	if len(company) < 2 || len(company) > 200 {
		return
	}

	// Save company, advance to step 3
	h.userRepo.SetOnboardData(ctx, user.ID, "email", user.FullName, company, "")

	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("📝 <b>Шаг 3/3</b>\n"+
				"✅ Имя: %s\n"+
				"✅ Компания: %s\n\n"+
				"Введите вашу корпоративную email:", user.FullName, company),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// onboardStepEmail processes Step 3: email input → send for admin approval.
func (h *PartnerHandlers) onboardStepEmail(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, email string) {
	email = strings.ToLower(strings.TrimSpace(email))

	// Validate email format
	if !emailRegex.MatchString(email) {
		if user.OnboardMsgID != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: *user.OnboardMsgID,
				Text: fmt.Sprintf("📝 <b>Шаг 3/3</b>\n"+
					"✅ Имя: %s\n"+
					"✅ Компания: %s\n\n"+
					"❌ Неверный формат email. Попробуйте ещё раз:", user.FullName, user.CompanyName),
				ParseMode: models.ParseModeHTML,
			})
		}
		return
	}

	// Save email, clear step (done with input)
	h.userRepo.SetOnboardData(ctx, user.ID, "", user.FullName, user.CompanyName, email)

	// Edit partner's message to "waiting for approval"
	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("✅ <b>Запрос на верификацию отправлен!</b>\n\n"+
				"👤 %s\n"+
				"🏢 %s\n"+
				"📧 %s\n\n"+
				"⏳ Ожидайте подтверждения",
				user.FullName, user.CompanyName, email),
			ParseMode: models.ParseModeHTML,
		})
	}

	// Notify admin in PBM bot
	h.notifyAdminPartnerRequest(ctx, b, user, email)
}

// notifyAdminPartnerRequest sends a notification to the PBM admin about a new partner request.
func (h *PartnerHandlers) notifyAdminPartnerRequest(ctx context.Context, b *bot.Bot, user *domain.User, email string) {
	text := fmt.Sprintf(
		"🆕 <b>Запрос на доступ (Partner Bot)</b>\n\n"+
			"👤 %s\n"+
			"📱 @%s\n"+
			"🏢 %s\n"+
			"📧 %s\n\n"+
			"Проверьте и решите:",
		user.FullName, user.Username, user.CompanyName, email,
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

// HandleApproveCallback handles admin approve/reject of partner requests.
// The admin clicks buttons in the partner bot's notification message.
func (h *PartnerHandlers) HandleApproveCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	// Only admin can approve/reject
	if update.CallbackQuery.From.ID != h.adminChatID {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Access denied",
			ShowAlert:       true,
		})
		return
	}

	data := update.CallbackQuery.Data
	isApprove := strings.HasPrefix(data, "papprove:")

	var prefix string
	if isApprove {
		prefix = "papprove:"
	} else {
		prefix = "preject:"
	}

	userIDStr := strings.TrimPrefix(data, prefix)
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return
	}

	// Load the partner user
	partnerUser, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ User not found",
			ShowAlert:       true,
		})
		return
	}

	if isApprove {
		// Approve: update DB
		if err := h.userRepo.ApprovePartner(ctx, userID); err != nil {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Error: " + err.Error(),
				ShowAlert:       true,
			})
			return
		}

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Approved!",
		})

		// Edit admin message
		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("✅ <b>Одобрено</b>\n\n"+
					"👤 %s\n"+
					"🏢 %s\n"+
					"📧 %s",
					partnerUser.FullName, partnerUser.CompanyName, partnerUser.Email),
				ParseMode: models.ParseModeHTML,
			})
		}

		// Edit partner's onboarding message
		if partnerUser.OnboardMsgID != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    partnerUser.TelegramID,
				MessageID: *partnerUser.OnboardMsgID,
				Text: "✅ <b>Доступ подтверждён!</b>\n\n" +
					"Используйте /status для просмотра\n" +
					"карточки вашей компании.",
				ParseMode: models.ParseModeHTML,
			})
		}

	} else {
		// Reject: edit admin message with "no comment" option + ask for comment
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		// Edit admin message — replace buttons with comment prompt
		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("❌ <b>Отклонение</b>\n\n"+
					"👤 %s\n"+
					"📧 %s\n\n"+
					"Напишите причину отказа или нажмите кнопку:",
					partnerUser.FullName, partnerUser.Email),
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: &models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{
						{{Text: "Без комментария", CallbackData: fmt.Sprintf("prejectconfirm:%d:", userID)}},
					},
				},
			})
		}

		// Save pending reject state — admin's next text message will be the comment
		h.userRepo.SetOnboardData(ctx, userID, "rejected", partnerUser.FullName, partnerUser.CompanyName, partnerUser.Email)
	}
}

// HandleRejectConfirm handles the "no comment" reject confirm button.
func (h *PartnerHandlers) HandleRejectConfirm(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	if update.CallbackQuery.From.ID != h.adminChatID {
		return
	}

	data := update.CallbackQuery.Data
	// Format: "prejectconfirm:123:" or "prejectconfirm:123:comment text"
	rest := strings.TrimPrefix(data, "prejectconfirm:")
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) < 1 {
		return
	}

	userID, err := strconv.Atoi(parts[0])
	if err != nil {
		return
	}

	comment := ""
	if len(parts) == 2 {
		comment = parts[1]
	}

	h.executeReject(ctx, b, update, userID, comment)
}

// executeReject performs the actual rejection with optional comment.
func (h *PartnerHandlers) executeReject(ctx context.Context, b *bot.Bot, update *models.Update, userID int, comment string) {
	partnerUser, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	// Reset user for re-registration
	h.userRepo.ResetOnboard(ctx, userID)

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "❌ Rejected",
	})

	// Build reject text
	commentLine := ""
	if comment != "" {
		commentLine = fmt.Sprintf("\n💬 %s", comment)
	}

	// Edit admin message
	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text: fmt.Sprintf("❌ <b>Отклонено</b>\n\n"+
				"👤 %s\n"+
				"📧 %s%s",
				partnerUser.FullName, partnerUser.Email, commentLine),
			ParseMode: models.ParseModeHTML,
		})
	}

	// Edit partner's onboarding message
	if partnerUser.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    partnerUser.TelegramID,
			MessageID: *partnerUser.OnboardMsgID,
			Text: fmt.Sprintf("❌ <b>Запрос отклонён</b>%s\n\n"+
				"Нажмите /start чтобы подать заново",
				commentLine),
			ParseMode: models.ParseModeHTML,
		})
	}
}
