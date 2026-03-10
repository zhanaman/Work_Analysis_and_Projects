package handlers

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// OnboardingHandler handles the multi-step onboarding flow for HPE Partner Advisor bot.
// Flow: role selection → name → [company (Partner)] → email → admin approval.
type OnboardingHandler struct {
	userRepo *storage.UserRepo
	adminID  int64
}

// NewOnboardingHandler creates a new OnboardingHandler.
func NewOnboardingHandler(userRepo *storage.UserRepo, adminID int64) *OnboardingHandler {
	return &OnboardingHandler{
		userRepo: userRepo,
		adminID:  adminID,
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Onboard step encoding: "role" | "name:<role>" | "company:<role>" | "email:<role>"
// where <role> = "partner" | "pbm"

func parseStep(step string) (phase string, role string) {
	parts := strings.SplitN(step, ":", 2)
	phase = parts[0]
	if len(parts) == 2 {
		role = parts[1]
	}
	return
}

// HandleOnboardingMessage processes free-text messages for users in onboarding.
func (h *OnboardingHandler) HandleOnboardingMessage(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || user.OnboardStep == "" {
		return
	}

	chatID := update.Message.Chat.ID
	text := strings.TrimSpace(update.Message.Text)

	// Skip commands
	if strings.HasPrefix(text, "/") {
		return
	}

	// Delete user's input message for clean UI
	b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatID,
		MessageID: update.Message.ID,
	})

	phase, role := parseStep(user.OnboardStep)

	switch phase {
	case "role":
		// Ignore text during role selection — user must tap button
		return
	case "name":
		h.stepName(ctx, b, chatID, user.ID, text, role)
	case "company":
		h.stepCompany(ctx, b, chatID, user.ID, text, role)
	case "email":
		h.stepEmail(ctx, b, chatID, user, text, role)
	}
}

// StartOnboarding sends Step 1: role selection via inline buttons.
func (h *OnboardingHandler) StartOnboarding(ctx context.Context, b *bot.Bot, chatID int64, userID int) {
	// Set onboard_step to "role"
	if err := h.userRepo.SetOnboardData(ctx, userID, "role", "", "", ""); err != nil {
		slog.Error("onboard: set step role", "error", err)
		return
	}

	userDBIDStr := strconv.Itoa(userID)

	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: "👋 <b>Добро пожаловать в HPE Partner Advisor!</b>\n\n" +
			"⬛⬜⬜ <b>Шаг 1</b>\n\n" +
			"Выберите вашу роль:\n\n" +
			"<i>Для отмены: /cancel</i>",
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "🏢 Partner", CallbackData: "onboard_role:" + userDBIDStr + ":partner"},
					{Text: "👔 PBM", CallbackData: "onboard_role:" + userDBIDStr + ":pbm"},
					{Text: "📦 Distri", CallbackData: "onboard_role:" + userDBIDStr + ":distri"},
				},
			},
		},
	})
	if err != nil {
		slog.Error("onboard: send step 1", "error", err)
		return
	}

	if err := h.userRepo.SetOnboardMsgID(ctx, userID, msg.ID); err != nil {
		slog.Error("onboard: save msg id", "error", err)
	}
}

// HandleRoleCallback processes the role selection inline button.
// Callback format: onboard_role:<user_db_id>:<role>
func (h *OnboardingHandler) HandleRoleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := strings.TrimPrefix(update.CallbackQuery.Data, "onboard_role:")
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return
	}

	userDBID, err := strconv.Atoi(parts[0])
	if err != nil {
		return
	}
	selectedRole := parts[1]

	// Distri = stub
	if selectedRole == "distri" {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})
		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: "🚧 <b>Дистрибьюторский доступ</b>\n\n" +
					"Эта функция в разработке.\n" +
					"Нажмите /start чтобы выбрать другую роль.",
				ParseMode: models.ParseModeHTML,
			})
		}
		// Reset onboard step
		h.userRepo.SetOnboardData(ctx, userDBID, "", "", "", "")
		return
	}

	if selectedRole != "partner" && selectedRole != "pbm" {
		return
	}

	// Advance to name step with role
	if err := h.userRepo.SetOnboardData(ctx, userDBID, "name:"+selectedRole, "", "", ""); err != nil {
		slog.Error("onboard: set step name", "error", err)
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	roleLabel := "🏢 Partner"
	if selectedRole == "pbm" {
		roleLabel = "👔 PBM"
	}

	totalSteps := "3"
	if selectedRole == "partner" {
		totalSteps = "4"
	}

	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text: fmt.Sprintf("⬛⬛⬜ <b>Шаг 2/%s</b>\n"+
				"✅ Роль: %s\n\n"+
				"Введите ваше полное имя:\n\n"+
				"<i>Для отмены: /cancel</i>",
				totalSteps, roleLabel),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// stepName processes name input.
func (h *OnboardingHandler) stepName(ctx context.Context, b *bot.Bot, chatID int64, userID int, name string, role string) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return
	}

	// Determine next step based on role
	var nextStep string
	if role == "partner" {
		nextStep = "company:" + role
	} else {
		nextStep = "email:" + role
	}

	if err := h.userRepo.SetOnboardData(ctx, userID, nextStep, name, "", ""); err != nil {
		slog.Error("onboard: save name", "error", err)
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("onboard: reload user", "error", err)
		return
	}

	if user.OnboardMsgID == nil {
		return
	}

	if role == "partner" {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("⬛⬛⬛⬜ <b>Шаг 3/4</b>\n"+
				"✅ Роль: 🏢 Partner\n"+
				"✅ Имя: %s\n\n"+
				"Введите название вашей компании:\n\n"+
				"<i>Для отмены: /cancel</i>",
				html.EscapeString(name)),
			ParseMode: models.ParseModeHTML,
		})
	} else {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("⬛⬛⬛ <b>Шаг 3/3</b>\n"+
				"✅ Роль: 👔 PBM\n"+
				"✅ Имя: %s\n\n"+
				"Введите ваш HPE email (@hpe.com):\n\n"+
				"<i>Для отмены: /cancel</i>",
				html.EscapeString(name)),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// stepCompany processes company name input (Partner only).
func (h *OnboardingHandler) stepCompany(ctx context.Context, b *bot.Bot, chatID int64, userID int, company string, role string) {
	company = strings.TrimSpace(company)
	if len(company) < 2 || len(company) > 200 {
		return
	}

	// Save company, advance to email step
	if err := h.userRepo.SetOnboardData(ctx, userID, "email:"+role, "", company, ""); err != nil {
		slog.Error("onboard: save company", "error", err)
		return
	}

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("onboard: reload user", "error", err)
		return
	}

	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("⬛⬛⬛⬛ <b>Шаг 4/4</b>\n"+
				"✅ Роль: 🏢 Partner\n"+
				"✅ Имя: %s\n"+
				"✅ Компания: %s\n\n"+
				"Введите ваш корпоративный email:\n\n"+
				"<i>Для отмены: /cancel</i>",
				html.EscapeString(user.FullName),
				html.EscapeString(company)),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// stepEmail processes email input with role-specific validation.
func (h *OnboardingHandler) stepEmail(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, email string, role string) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) || len(email) > 320 {
		h.emailError(ctx, b, chatID, user, role, "❌ Неверный формат email. Попробуйте ещё раз:")
		return
	}

	// Role-specific validation
	emailDomain := email[strings.LastIndex(email, "@")+1:]

	if role == "pbm" {
		if emailDomain != "hpe.com" {
			h.emailError(ctx, b, chatID, user, role, "❌ PBM требует email @hpe.com:")
			return
		}
	} else if role == "partner" {
		// Fuzzy check: email domain should relate to company name
		companyLower := strings.ToLower(user.CompanyName)
		domainBase := strings.TrimSuffix(emailDomain, ".com")
		domainBase = strings.TrimSuffix(domainBase, ".kz")
		domainBase = strings.TrimSuffix(domainBase, ".uz")
		domainBase = strings.TrimSuffix(domainBase, ".ru")
		domainBase = strings.TrimSuffix(domainBase, ".net")
		domainBase = strings.TrimSuffix(domainBase, ".org")
		domainBase = strings.ToLower(domainBase)

		// Check if domain base is contained in company or company in domain
		if !strings.Contains(companyLower, domainBase) && !strings.Contains(domainBase, strings.ReplaceAll(companyLower, " ", "")) {
			h.emailError(ctx, b, chatID, user, role,
				fmt.Sprintf("❌ Домен email (%s) не совпадает с компанией «%s».\nВведите корпоративный email:",
					emailDomain, user.CompanyName))
			return
		}
	}

	// Save email, clear onboard step → pending
	if err := h.userRepo.SetOnboardData(ctx, user.ID, "", user.FullName, user.CompanyName, email); err != nil {
		slog.Error("onboard: save email", "error", err)
		return
	}

	// Update onboarding message to "waiting"
	if user.OnboardMsgID != nil {
		var companyLine string
		if user.CompanyName != "" {
			companyLine = fmt.Sprintf("🏢 %s\n", html.EscapeString(user.CompanyName))
		}
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("✅ <b>Запрос отправлен!</b>\n\n"+
				"👤 %s\n"+
				"%s"+
				"📧 %s\n\n"+
				"⏳ Ожидайте подтверждения администратора.",
				html.EscapeString(user.FullName),
				companyLine,
				html.EscapeString(email)),
			ParseMode: models.ParseModeHTML,
		})
	}

	// Notify admin with pre-tagged role
	h.notifyAdmin(ctx, b, user, email, role)
}

// emailError shows an email validation error in the onboarding message.
func (h *OnboardingHandler) emailError(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, role string, errMsg string) {
	if user.OnboardMsgID == nil {
		return
	}

	var nameFields string
	if role == "partner" {
		nameFields = fmt.Sprintf("✅ Имя: %s\n✅ Компания: %s",
			html.EscapeString(user.FullName), html.EscapeString(user.CompanyName))
	} else {
		nameFields = fmt.Sprintf("✅ Имя: %s", html.EscapeString(user.FullName))
	}

	stepLabel := "Шаг 3/3"
	roleLabel := "👔 PBM"
	if role == "partner" {
		stepLabel = "Шаг 4/4"
		roleLabel = "🏢 Partner"
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: *user.OnboardMsgID,
		Text: fmt.Sprintf("⬛⬛⬛ <b>%s</b>\n"+
			"✅ Роль: %s\n"+
			"%s\n\n"+
			"%s\n\n"+
			"<i>Для отмены: /cancel</i>",
			stepLabel, roleLabel, nameFields, errMsg),
		ParseMode: models.ParseModeHTML,
	})
}

// notifyAdmin sends a rich notification to admin with the requested role pre-tagged.
func (h *OnboardingHandler) notifyAdmin(ctx context.Context, b *bot.Bot, user *domain.User, email string, role string) {
	tgIDStr := fmt.Sprintf("%d", user.TelegramID)

	roleLabel := "🏢 Partner"
	approveText := "🏢 Одобрить (Partner)"
	approveData := "approve:" + tgIDStr
	if role == "pbm" {
		roleLabel = "👔 PBM"
		approveText = "👔 Одобрить (PBM)"
		approveData = "role_pbm:" + tgIDStr
	}

	var companyLine string
	if user.CompanyName != "" {
		companyLine = fmt.Sprintf("🏢 %s\n", html.EscapeString(user.CompanyName))
	}

	text := fmt.Sprintf("🆕 <b>Новый запрос на доступ:</b>\n\n"+
		"👤 %s\n"+
		"%s"+
		"📧 %s\n"+
		"🔗 @%s\n"+
		"🏷 Запрашиваемая роль: %s\n\n"+
		"Выберите действие:",
		html.EscapeString(user.FullName),
		companyLine,
		html.EscapeString(email),
		html.EscapeString(user.Username),
		roleLabel,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    h.adminID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: approveText, CallbackData: approveData},
					{Text: "❌ Отклонить", CallbackData: "reject:" + tgIDStr},
				},
				{
					{Text: "✏️ Другая роль", CallbackData: "chrole:" + tgIDStr},
				},
			},
		},
	})
}
