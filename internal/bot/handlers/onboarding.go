package handlers

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"regexp"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// OnboardingHandler handles the 2-step onboarding flow for HPE Partner Advisor bot.
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

	switch user.OnboardStep {
	case "name":
		h.stepName(ctx, b, chatID, user.ID, text)
	case "email":
		h.stepEmail(ctx, b, chatID, user, text)
	}
}

// StartOnboarding sends the initial onboarding message (Step 1: Name).
func (h *OnboardingHandler) StartOnboarding(ctx context.Context, b *bot.Bot, chatID int64, userID int) {
	msg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text: "👋 <b>Добро пожаловать в HPE Partner Advisor!</b>\n\n" +
			"⬛⬜ <b>Шаг 1/2</b>\n\n" +
			"Введите ваше полное имя:\n\n" +
			"<i>Для отмены: /cancel</i>",
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		slog.Error("onboard: send step 1", "error", err)
		return
	}

	if err := h.userRepo.SetOnboardMsgID(ctx, userID, msg.ID); err != nil {
		slog.Error("onboard: save msg id", "error", err)
	}
}

// stepName processes Step 1: full name input.
func (h *OnboardingHandler) stepName(ctx context.Context, b *bot.Bot, chatID int64, userID int, name string) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return
	}

	// Save name, advance to email step
	if err := h.userRepo.SetOnboardData(ctx, userID, "email", name, "", ""); err != nil {
		slog.Error("onboard: save name", "error", err)
		return
	}

	// Reload user to get OnboardMsgID
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("onboard: reload user", "error", err)
		return
	}

	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("⬛⬛ <b>Шаг 2/2</b>\n"+
				"✅ Имя: %s\n\n"+
				"Введите ваш email:\n\n"+
				"<i>Для отмены: /cancel</i>",
				html.EscapeString(name)),
			ParseMode: models.ParseModeHTML,
		})
	}
}

// stepEmail processes Step 2: email input, completes onboarding.
func (h *OnboardingHandler) stepEmail(ctx context.Context, b *bot.Bot, chatID int64, user *domain.User, email string) {
	email = strings.TrimSpace(strings.ToLower(email))

	if !emailRegex.MatchString(email) || len(email) > 320 {
		if user.OnboardMsgID != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: *user.OnboardMsgID,
				Text: fmt.Sprintf("⬛⬛ <b>Шаг 2/2</b>\n"+
					"✅ Имя: %s\n\n"+
					"❌ Неверный формат email. Попробуйте ещё раз:\n\n"+
					"<i>Для отмены: /cancel</i>",
					html.EscapeString(user.FullName)),
				ParseMode: models.ParseModeHTML,
			})
		}
		return
	}

	// Save email, clear onboard step → pending
	if err := h.userRepo.SetOnboardData(ctx, user.ID, "", user.FullName, "", email); err != nil {
		slog.Error("onboard: save email", "error", err)
		return
	}

	// Update onboarding message to "waiting"
	if user.OnboardMsgID != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: *user.OnboardMsgID,
			Text: fmt.Sprintf("✅ <b>Запрос отправлен!</b>\n\n"+
				"👤 %s\n"+
				"📧 %s\n\n"+
				"⏳ Ожидайте подтверждения администратора.",
				html.EscapeString(user.FullName),
				html.EscapeString(email)),
			ParseMode: models.ParseModeHTML,
		})
	}

	// Notify admin with role buttons
	h.notifyAdminOnboardComplete(ctx, b, user, email)
}

// notifyAdminOnboardComplete sends a rich notification to admin after onboarding.
func (h *OnboardingHandler) notifyAdminOnboardComplete(ctx context.Context, b *bot.Bot, user *domain.User, email string) {
	tgIDStr := fmt.Sprintf("%d", user.TelegramID)
	text := fmt.Sprintf("🆕 <b>Новый запрос на доступ:</b>\n\n"+
		"👤 %s\n"+
		"📧 %s\n"+
		"🔗 @%s\n"+
		"🆔 <code>%s</code>\n\n"+
		"Выберите роль:",
		html.EscapeString(user.FullName),
		html.EscapeString(email),
		html.EscapeString(user.Username),
		tgIDStr,
	)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    h.adminID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "✅ User", CallbackData: "approve:" + tgIDStr},
					{Text: "👔 PBM", CallbackData: "role_pbm:" + tgIDStr},
					{Text: "📦 Distri", CallbackData: "role_distri:" + tgIDStr},
					{Text: "❌ Отклонить", CallbackData: "reject:" + tgIDStr},
				},
			},
		},
	})
}
