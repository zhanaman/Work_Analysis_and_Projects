package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// HandleApproveCallback handles admin approval/rejection of partner users.
// Callback format: "papprove:123" or "preject:123" where 123 is the user DB ID.
func (h *PartnerHandlers) HandleApproveCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
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

	// Load the user
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		slog.Error("partner-bot: get user for approval", "error", err, "user_id", userID)
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ User not found",
			ShowAlert:       true,
		})
		return
	}

	if isApprove {
		if err := h.userRepo.ApprovePartner(ctx, userID); err != nil {
			slog.Error("partner-bot: approve partner", "error", err)
			return
		}

		// Notify the partner
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: user.TelegramID,
			Text:   "🎉 Ваш доступ одобрен!\n\nИспользуйте /status чтобы посмотреть карточку вашей компании.\n\n🎉 Your access has been approved!\nUse /status to view your company card.",
		})

		// Update admin message
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Approved!",
		})

		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("✅ <b>Approved</b>\n\n👤 %s (@%s)\n📧 %s\n🏢 Partner ID: %d",
					user.FullName, user.Username, user.Email, safePartnerID(user.PartnerID)),
				ParseMode: models.ParseModeHTML,
			})
		}
	} else {
		// Reject — set role back to "rejected" (could reuse pending or add new role)
		if err := h.userRepo.SetRole(ctx, user.TelegramID, domain.RolePending); err != nil {
			slog.Error("partner-bot: reject partner", "error", err)
		}

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: user.TelegramID,
			Text:   "❌ Ваш запрос на доступ был отклонён.\nОбратитесь к вашему HPE PBM.\n\n❌ Your access request was rejected.\nPlease contact your HPE PBM.",
		})

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Rejected",
		})

		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("❌ <b>Rejected</b>\n\n👤 %s (@%s)\n📧 %s",
					user.FullName, user.Username, user.Email),
				ParseMode: models.ParseModeHTML,
			})
		}
	}
}

func safePartnerID(pid *int) int {
	if pid == nil {
		return 0
	}
	return *pid
}
