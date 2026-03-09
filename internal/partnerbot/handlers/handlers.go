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

	// Already approved — personalized welcome back with quick actions (4.4, 4.8)
	if user.IsAuthorized() && user.PartnerID != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text: fmt.Sprintf("\U0001f44b <b>С возвращением, %s!</b>\n\nВыберите действие:", user.FullName),
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: [][]models.InlineKeyboardButton{
					{
						{Text: "\U0001f4ca Карточка компании", CallbackData: fmt.Sprintf("pcard:%d:retention", *user.PartnerID)},
					},
					{
						{Text: "\U0001f310 Язык", CallbackData: "lang:choose"},
						{Text: "\u2753 Помощь", CallbackData: "phelp"},
					},
				},
			},
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

	// New user or reset — start onboarding: Step 1 (full name) with progress bar (4.7)
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      "\U0001f44b Добро пожаловать в Partner Performance Bot!\n\n\u2b1b\u2b1c\u2b1c <b>Шаг 1/3</b>\nВведите ваше имя и фамилию:\n\n<i>Для отмены: /cancel</i>",
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

// HandleCancel handles the /cancel command — exits onboarding (4.1).
func (h *PartnerHandlers) HandleCancel(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	chatID := update.Message.Chat.ID

	if user == nil || user.OnboardStep == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "\u2139\ufe0f Нечего отменять.",
		})
		return
	}

	// Reset onboarding state
	h.userRepo.SetOnboardData(ctx, user.ID, "", "", "", "")

	// Edit the onboarding message
	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text:      "\u274c <b>Регистрация отменена</b>\n\nНажмите /start чтобы начать заново.",
			ParseMode: models.ParseModeHTML,
		})
	}
}

// HandleHelpCallback handles inline help button.
func (h *PartnerHandlers) HandleHelpCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}
	lang := middleware.LangFromContext(ctx)
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:      i18n.T("help", lang),
			ParseMode: models.ParseModeHTML,
		})
	}
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
		// Informative empty state (4.3)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text: "\U0001f50d <b>Компания не привязана</b>\n\n" +
				"Чтобы начать:\n" +
				"1. Нажмите /start\n" +
				"2. Введите имя и название компании\n" +
				"3. После проверки вы увидите карточку\n\n" +
				"\U0001f4a1 Используйте точное название как в документах HPE.",
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// Typing indicator (4.2)
	b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID, Action: models.ChatActionTyping,
	})

	partner, err := h.partnerRepo.GetByID(ctx, *user.PartnerID)
	if err != nil {
		slog.Error("partner-bot: get partner", "error", err, "partner_id", *user.PartnerID)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "\u274c Ошибка загрузки данных. Попробуйте позже.",
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

	// Save name, advance to step 2 with progress bar (4.7)
	h.userRepo.SetOnboardData(ctx, user.ID, "company", name, "", "")

	// Edit the bot's onboarding message
	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("\u2b1b\u2b1b\u2b1c <b>\u0428\u0430\u0433 2/3</b>\n"+
				"\u2705 \u0418\u043c\u044f: %s\n\n"+
				"\u0412\u0432\u0435\u0434\u0438\u0442\u0435 \u043d\u0430\u0437\u0432\u0430\u043d\u0438\u0435 \u0432\u0430\u0448\u0435\u0439 \u043a\u043e\u043c\u043f\u0430\u043d\u0438\u0438:\n\n<i>\u0414\u043b\u044f \u043e\u0442\u043c\u0435\u043d\u044b: /cancel</i>", name),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// onboardStepCompany processes Step 2: company name input.
func (h *PartnerHandlers) onboardStepCompany(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, company string) {
	if len(company) < 2 || len(company) > 200 {
		return
	}

	// Typing indicator (4.2)
	b.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID, Action: models.ChatActionTyping,
	})

	exists, err := h.partnerRepo.ExistsByName(ctx, company)
	if err != nil {
		slog.Error("partner-bot: check company exists", "error", err)
	}

	if !exists {
		// Fuzzy search for suggestions (4.5)
		suggestions, fuzzyErr := h.partnerRepo.SearchByNameFuzzy(ctx, company)
		if fuzzyErr != nil {
			slog.Error("partner-bot: fuzzy search failed", "error", fuzzyErr)
		}

		var buttons [][]models.InlineKeyboardButton
		for _, s := range suggestions {
			// Telegram callback_data max 64 bytes
			cb := fmt.Sprintf("pcompany:%s", s)
			if len(cb) > 64 {
				cb = cb[:64]
			}
			buttons = append(buttons, []models.InlineKeyboardButton{
				{Text: s, CallbackData: cb},
			})
		}

		slog.Info("partner-bot: company not found",
			"company", company,
			"suggestions", len(suggestions),
			"onboard_msg_id", user.OnboardMsgID,
		)

		if user.OnboardMsgID != nil {
			hint := ""
			if len(suggestions) > 0 {
				hint = "\n\n\U0001f4a1 Возможно, вы имели в виду:"
			}

			msgText := fmt.Sprintf("\u2b1b\u2b1b\u2b1c <b>Шаг 2/3</b>\n"+
				"\u2705 Имя: %s\n\n"+
				"\u274c Компания <b>\"%s\"</b> не найдена в базе.%s\n\n"+
				"<i>Введите точное юридическое название (как в документах HPE)\nДля отмены: /cancel</i>",
				user.FullName, company, hint)

			var editErr error
			if len(buttons) > 0 {
				_, editErr = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    chatID,
					MessageID: *user.OnboardMsgID,
					Text:      msgText,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: buttons},
				})
			} else {
				_, editErr = b.EditMessageText(ctx, &bot.EditMessageTextParams{
					ChatID:    chatID,
					MessageID: *user.OnboardMsgID,
					Text:      msgText,
					ParseMode: models.ParseModeHTML,
				})
			}
			if editErr != nil {
				slog.Error("partner-bot: edit message failed", "error", editErr, "msg_id", *user.OnboardMsgID)
			}
		} else {
			slog.Warn("partner-bot: onboard_msg_id is nil, cannot edit message")
		}
		return
	}

	// Save company, advance to step 3 with progress bar (4.7)
	h.userRepo.SetOnboardData(ctx, user.ID, "email", user.FullName, company, "")

	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("\u2b1b\u2b1b\u2b1b <b>Шаг 3/3</b>\n"+
				"\u2705 Имя: %s\n"+
				"\u2705 Компания: %s\n\n"+
				"Введите вашу корпоративную email:\n\n"+
				"<i>Для отмены: /cancel</i>", user.FullName, company),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// HandleCompanySelect handles inline button selection of a suggested company name (4.5).
func (h *PartnerHandlers) HandleCompanySelect(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || user.OnboardStep != "company" {
		return
	}

	company := strings.TrimPrefix(update.CallbackQuery.Data, "pcompany:")
	if company == "" {
		return
	}

	// Accept the selected company
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "\u2705 " + company,
	})

	chatID := update.CallbackQuery.From.ID
	h.userRepo.SetOnboardData(ctx, user.ID, "email", user.FullName, company, "")

	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("\u2b1b\u2b1b\u2b1b <b>Шаг 3/3</b>\n"+
				"\u2705 Имя: %s\n"+
				"\u2705 Компания: %s\n\n"+
				"Введите вашу корпоративную email:\n\n"+
				"<i>Для отмены: /cancel</i>", user.FullName, company),
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

