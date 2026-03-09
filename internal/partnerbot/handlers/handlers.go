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
	"github.com/anonimouskz/pbm-partner-bot/internal/shared/tgapi"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// PartnerHandlers holds all handlers for the Partner bot.
type PartnerHandlers struct {
	userRepo    *storage.UserRepo
	partnerRepo *storage.PartnerRepo
	adminChatID int64  // PBM admin Telegram ID for notifications
	pbmBotToken string // PBM bot token for sending admin notifications
}

// New creates a new PartnerHandlers.
func New(userRepo *storage.UserRepo, partnerRepo *storage.PartnerRepo, adminChatID int64, pbmBotToken string) *PartnerHandlers {
	return &PartnerHandlers{
		userRepo:    userRepo,
		partnerRepo: partnerRepo,
		adminChatID: adminChatID,
		pbmBotToken: pbmBotToken,
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

	exists, err := h.partnerRepo.ExistsByName(ctx, company)
	if err != nil {
		slog.Error("partner-bot: check company exists", "error", err)
	}

	if !exists {
		if user.OnboardMsgID != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: *user.OnboardMsgID,
				Text: fmt.Sprintf("📝 <b>Шаг 2/3</b>\n"+
					"✅ Имя: %s\n\n"+
					"❌ Компания <b>\"%s\"</b> не найдена в базе данных.\n"+
					"Пожалуйста, введите точное юридическое название вашей компании (как в документах HPE):", user.FullName, company),
				ParseMode: models.ParseModeHTML,
			})
		}
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

// notifyAdminPartnerRequest sends a notification to the PBM admin via the PBM bot.
// Uses the PBM bot token so the message appears in the admin's PBM bot chat.
func (h *PartnerHandlers) notifyAdminPartnerRequest(ctx context.Context, b *bot.Bot, user *domain.User, email string) {
	text := fmt.Sprintf(
		"\U0001f195 <b>Запрос на доступ (Partner Bot)</b>\n\n"+
			"\U0001f464 %s\n"+
			"\U0001f4f1 @%s\n"+
			"\U0001f3e2 %s\n"+
			"\U0001f4e7 %s\n\n"+
			"Проверьте и решите:",
		user.FullName, user.Username, user.CompanyName, email,
	)

	_, err := tgapi.SendMessage(h.pbmBotToken, tgapi.SendMessageParams{
		ChatID:    h.adminChatID,
		Text:      text,
		ParseMode: "HTML",
		ReplyMarkup: &tgapi.InlineKeyboardMarkup{
			InlineKeyboard: [][]tgapi.InlineKeyboardButton{
				{
					{Text: "\u2705 Approve", CallbackData: fmt.Sprintf("papprove:%d", user.ID)},
					{Text: "\u274c Reject", CallbackData: fmt.Sprintf("preject:%d", user.ID)},
				},
			},
		},
	})
	if err != nil {
		slog.Error("failed to send admin notification via PBM bot", "error", err)
	}
}

