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

// AdminHandler handles admin-only commands.
type AdminHandler struct {
	userRepo    *storage.UserRepo
	partnerRepo *storage.PartnerRepo
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(userRepo *storage.UserRepo, partnerRepo *storage.PartnerRepo) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		partnerRepo: partnerRepo,
	}
}

// HandleStats shows database statistics.
func (h *AdminHandler) HandleStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAuthorized() {
		return
	}

	count, err := h.partnerRepo.CountAll(ctx)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения статистики.",
		})
		return
	}

	text := fmt.Sprintf("📊 *Статистика базы данных*\n\n"+
		"🏢 Партнёров: *%d*", count)

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
	})
}

// HandleUsers shows pending users for admin approval.
func (h *AdminHandler) HandleUsers(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAdmin() {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🚫 Эта команда доступна только администратору.",
		})
		return
	}

	pending, err := h.userRepo.ListPending(ctx)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "❌ Ошибка получения списка пользователей.",
		})
		return
	}

	if len(pending) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "✅ Нет ожидающих одобрения пользователей.",
		})
		return
	}

	var rows [][]models.InlineKeyboardButton
	text := fmt.Sprintf("⏳ *Ожидают одобрения: %d*\n\n", len(pending))

	for _, u := range pending {
		text += fmt.Sprintf("• %s (@%s)\n", u.FullName, u.Username)
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         "✅ " + u.FullName,
				CallbackData: "approve:" + strconv.FormatInt(u.TelegramID, 10),
			},
			{
				Text:         "❌ Reject",
				CallbackData: "reject:" + strconv.FormatInt(u.TelegramID, 10),
			},
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: rows,
		},
	})
}

// HandleApproveCallback processes approve/reject callbacks.
func (h *AdminHandler) HandleApproveCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAdmin() {
		return
	}

	data := update.CallbackQuery.Data
	var action string
	var targetIDStr string

	if strings.HasPrefix(data, "approve:") {
		action = "approve"
		targetIDStr = strings.TrimPrefix(data, "approve:")
	} else if strings.HasPrefix(data, "reject:") {
		action = "reject"
		targetIDStr = strings.TrimPrefix(data, "reject:")
	} else {
		return
	}

	targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil {
		return
	}

	var newRole domain.Role
	var responseText string

	if action == "approve" {
		newRole = domain.RoleUser
		responseText = "✅ Пользователь одобрен!"
	} else {
		// For rejection, we could delete the user or keep as pending
		// For now, we'll just acknowledge
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Пользователь отклонён",
		})
		return
	}

	if err := h.userRepo.SetRole(ctx, targetID, newRole); err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Ошибка: " + err.Error(),
			ShowAlert:       true,
		})
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            responseText,
	})

	// Notify the approved user
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: targetID,
		Text:   "🎉 Ваш доступ к боту одобрен! Используйте /help для списка команд.",
	})
}
