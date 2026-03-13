package handlers

import (
	"context"
	"fmt"
	"strings"
	"strconv"
	"time"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/rbac"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const reportDays = 7     // default lookback period
const reportLimit = 50   // max entries in /report @user detail view

// HandleReport handles /report command (admin only).
// Usage:
//   /report          — Summary: per-user event counts for last 7 days
//   /report 123456789 — Detailed events for a specific Telegram user ID
func (h *AdminHandler) HandleReport(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !rbac.Can(user, rbac.ViewStats) {
		return
	}

	chatID := update.Message.Chat.ID
	arg := strings.TrimSpace(strings.TrimPrefix(update.Message.Text, "/report"))

	if arg == "" {
		h.handleReportSummary(ctx, b, chatID)
	} else {
		telegramID, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    chatID,
				Text:      "❌ Укажите Telegram ID пользователя.\n\nПример: <code>/report 123456789</code>",
				ParseMode: models.ParseModeHTML,
			})
			return
		}
		h.handleReportByUser(ctx, b, chatID, telegramID)
	}
}

// handleReportSummary shows a per-user activity summary for the last N days.
func (h *AdminHandler) handleReportSummary(ctx context.Context, b *bot.Bot, chatID int64) {
	summaries, err := h.activityRepo.GetUserSummary(ctx, reportDays)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при получении отчёта.",
		})
		return
	}

	if len(summaries) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   fmt.Sprintf("📊 Нет активности за последние %d дней.", reportDays),
		})
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 <b>Активность за %d дней</b>\n", reportDays))
	sb.WriteString(fmt.Sprintf("<i>%d активных пользователей</i>\n\n", len(summaries)))

	for _, s := range summaries {
		name := s.FullName
		if name == "" {
			name = fmt.Sprintf("tg:%d", s.TelegramID)
		}
		handle := ""
		if s.Username != "" {
			handle = " (@" + s.Username + ")"
		}
		sb.WriteString(fmt.Sprintf(
			"👤 <b>%s</b>%s\n  🔍 <code>%d</code> запросов  👁 <code>%d</code> карточек\n  🕐 %s\n  <code>/report %d</code>\n\n",
			escapeHTML(name), escapeHTML(handle),
			s.SearchCount, s.ViewCount,
			s.LastActive.In(time.UTC).Format("02.01 15:04"),
			s.TelegramID,
		))
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeHTML,
	})
}

// handleReportByUser shows detailed activity log for a specific user.
func (h *AdminHandler) handleReportByUser(ctx context.Context, b *bot.Bot, chatID int64, telegramID int64) {
	entries, err := h.activityRepo.GetByUser(ctx, telegramID, reportLimit)
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Ошибка при получении отчёта.",
		})
		return
	}

	if len(entries) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    chatID,
			Text:      fmt.Sprintf("📊 Нет активности для пользователя <code>%d</code>.", telegramID),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	// Get display name from first entry
	name := entries[0].FullName
	if name == "" {
		name = fmt.Sprintf("tg:%d", telegramID)
	}
	handle := ""
	if entries[0].Username != "" {
		handle = " (@" + entries[0].Username + ")"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>%s</b>%s\n<i>Последние %d событий</i>\n\n",
		escapeHTML(name), escapeHTML(handle), len(entries)))

	for _, e := range entries {
		ts := e.CreatedAt.In(time.UTC).Format("02.01 15:04")
		switch e.EventType {
		case domain.EventSearch:
			sb.WriteString(fmt.Sprintf("🔍 <code>%s</code>  %s\n", escapeHTML(e.Query), ts))
		case domain.EventPartnerView:
			sb.WriteString(fmt.Sprintf("👁 %s  %s\n", escapeHTML(e.PartnerName), ts))
		}
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      sb.String(),
		ParseMode: models.ParseModeHTML,
	})
}
