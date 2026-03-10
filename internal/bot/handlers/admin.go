package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/chart"
	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/rbac"
	"github.com/anonimouskz/pbm-partner-bot/internal/shared/tgapi"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// AdminHandler handles admin-only commands.
type AdminHandler struct {
	userRepo        *storage.UserRepo
	partnerRepo     *storage.PartnerRepo
	partnerBotToken string // Partner bot token for cross-bot message editing

	mu             sync.Mutex
	pendingRejects map[int64]int // admin TG ID → partner user DB ID
}

// NewAdminHandler creates a new AdminHandler.
func NewAdminHandler(userRepo *storage.UserRepo, partnerRepo *storage.PartnerRepo, partnerBotToken string) *AdminHandler {
	return &AdminHandler{
		userRepo:        userRepo,
		partnerRepo:     partnerRepo,
		partnerBotToken: partnerBotToken,
		pendingRejects:  make(map[int64]int),
	}
}

// SetPendingReject marks the admin as waiting for a rejection comment.
func (h *AdminHandler) SetPendingReject(adminTGID int64, partnerUserID int) {
	h.mu.Lock()
	h.pendingRejects[adminTGID] = partnerUserID
	h.mu.Unlock()
}

// TryHandleRejectComment checks if the admin has a pending reject and processes the comment.
// Returns true if the message was handled as a reject comment.
func (h *AdminHandler) TryHandleRejectComment(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.Message == nil || update.Message.Text == "" {
		return false
	}

	adminTGID := update.Message.From.ID

	h.mu.Lock()
	partnerUserID, ok := h.pendingRejects[adminTGID]
	if ok {
		delete(h.pendingRejects, adminTGID)
	}
	h.mu.Unlock()

	if !ok {
		return false
	}

	comment := strings.TrimSpace(update.Message.Text)

	partnerUser, err := h.userRepo.GetByID(ctx, partnerUserID)
	if err != nil {
		slog.Error("reject comment: get user", "error", err)
		return true
	}

	h.userRepo.ResetOnboard(ctx, partnerUserID)

	commentLine := ""
	if comment != "" {
		commentLine = fmt.Sprintf("\n\U0001f4ac %s", comment)
	}

	// Send admin confirmation
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text: fmt.Sprintf("\u274c <b>\u041e\u0442\u043a\u043b\u043e\u043d\u0435\u043d\u043e</b>\n\n\U0001f464 %s\n\U0001f4e7 %s%s",
			partnerUser.FullName, partnerUser.Email, commentLine),
		ParseMode: models.ParseModeHTML,
	})

	// Edit partner's onboarding message via partner bot token
	if partnerUser.OnboardMsgID != nil && h.partnerBotToken != "" {
		tgapi.EditMessageText(h.partnerBotToken, tgapi.EditMessageTextParams{
			ChatID:    partnerUser.TelegramID,
			MessageID: *partnerUser.OnboardMsgID,
			Text:      fmt.Sprintf("\u274c <b>\u0417\u0430\u043f\u0440\u043e\u0441 \u043e\u0442\u043a\u043b\u043e\u043d\u0451\u043d</b>%s\n\n\u041d\u0430\u0436\u043c\u0438\u0442\u0435 /start \u0447\u0442\u043e\u0431\u044b \u043f\u043e\u0434\u0430\u0442\u044c \u0437\u0430\u043d\u043e\u0432\u043e", commentLine),
			ParseMode: "HTML",
		})
	}

	return true
}

// HandleStats shows the compact stats dashboard hub.
func (h *AdminHandler) HandleStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ViewStats) {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "🚫 У вас нет доступа к аналитике.",
		})
		return
	}

	count, _ := h.partnerRepo.CountAll(ctx)
	dist, _ := h.partnerRepo.TierDistribution(ctx, "")

	var sb strings.Builder
	sb.WriteString("📊 <b>CCA Dashboard</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("🏢 <b>%d</b> партнёров  •  8 стран\n", count))
	sb.WriteString(fmt.Sprintf("💎 %d Platinum  •  🥇 %d Gold  •  🥈 %d Silver\n",
		dist["compute"]["platinum"],
		dist["compute"]["gold"],
		dist["compute"]["silver"]))

	// Data date
	dataDate, importedAt, err := h.partnerRepo.GetLastImportDate(ctx)
	if err == nil {
		if dataDate != "" {
			sb.WriteString(fmt.Sprintf("📅 Данные от: <b>%s</b>\n", dataDate))
		} else {
			sb.WriteString(fmt.Sprintf("📅 Импорт: %s\n", importedAt))
		}
	}

	buttons := [][]models.InlineKeyboardButton{
		{
			{Text: "🌍 Страны", CallbackData: "stats:countries"},
			{Text: "🏅 Тиры", CallbackData: "stats:tiers"},
		},
		{
			{Text: "📈 Gaps", CallbackData: "stats:upgrade"},
			{Text: "💰 Top Volume", CallbackData: "stats:volume"},
		},
		{
			{Text: "📈 Upgrade Pipeline", CallbackData: "chart:pipeline"},
		},
		{
			{Text: "🎯 Low-Hanging Fruit", CallbackData: "chart:fruit"},
		},
		{
			{Text: "⚠️ Retention Risk", CallbackData: "chart:risk"},
			{Text: "🧩 Concentration", CallbackData: "chart:concentration"},
		},
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      sb.String(),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
}

// HandleStatsCallback handles stats drill-down buttons.
func (h *AdminHandler) HandleStatsCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	section := strings.TrimPrefix(update.CallbackQuery.Data, "stats:")

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	// RBAC check for stats
	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ViewStats) {
		return
	}

	var chatID int64
	var msgID int
	if update.CallbackQuery.Message.Message != nil {
		chatID = update.CallbackQuery.Message.Message.Chat.ID
		msgID = update.CallbackQuery.Message.Message.ID
	}
	if chatID == 0 {
		return
	}

	var text string
	switch section {
	case "countries":
		text = h.statsCountries(ctx)
	case "tiers":
		text = h.statsTiers(ctx)
	case "upgrade":
		text = h.statsUpgrade(ctx)
	case "volume":
		text = h.statsVolume(ctx)
	default:
		return
	}

	backBtn := [][]models.InlineKeyboardButton{
		{
			{Text: "🌍 Страны", CallbackData: "stats:countries"},
			{Text: "🏅 Тиры", CallbackData: "stats:tiers"},
		},
		{
			{Text: "📈 Gaps", CallbackData: "stats:upgrade"},
			{Text: "💰 Top Volume", CallbackData: "stats:volume"},
		},
		{
			{Text: "📈 Upgrade Pipeline", CallbackData: "chart:pipeline"},
		},
		{
			{Text: "🎯 Low-Hanging Fruit", CallbackData: "chart:fruit"},
		},
		{
			{Text: "⚠️ Retention Risk", CallbackData: "chart:risk"},
			{Text: "🧩 Concentration", CallbackData: "chart:concentration"},
		},
		{
			{Text: "🔍 Поиск", CallbackData: "menu:search"},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: msgID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: backBtn,
		},
	})
}

func (h *AdminHandler) statsCountries(ctx context.Context) string {
	matrix, _ := h.partnerRepo.CountryTierMatrix(ctx)

	var sb strings.Builder
	sb.WriteString("🌍 <b>Партнёры по странам</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	// Header
	sb.WriteString("<code>              💎 🥇 🥈 🏷  All</code>\n")

	for _, r := range matrix {
		flag := countryFlag(r.Country)
		// Short country name for alignment
		name := r.Country
		if len(name) > 10 {
			name = name[:10]
		}
		sb.WriteString(fmt.Sprintf("%s <code>%-10s %2d %2d %2d %3d  %3d</code>\n",
			flag, name, r.Plat, r.Gold, r.Silver, r.Biz, r.Total))
	}

	return sb.String()
}

func (h *AdminHandler) statsTiers(ctx context.Context) string {
	dist, _ := h.partnerRepo.TierDistribution(ctx, "")

	var sb strings.Builder
	sb.WriteString("🏅 <b>Тиры по центрам</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	centers := []struct {
		key, name string
	}{
		{"compute", "Compute"},
		{"hybrid_cloud", "Hybrid Cloud"},
		{"networking", "Networking"},
	}

	for _, c := range centers {
		td := dist[c.key]
		total := td["platinum"] + td["gold"] + td["silver"] + td["business"]
		sb.WriteString(fmt.Sprintf("<b>%s</b> <i>(%d)</i>\n", c.name, total))
		sb.WriteString(fmt.Sprintf("  💎 %d  •  🥇 %d  •  🥈 %d  •  🏷 %d\n\n",
			td["platinum"], td["gold"], td["silver"], td["business"]))
	}

	return sb.String()
}

func (h *AdminHandler) statsUpgrade(ctx context.Context) string {
	ready, _ := h.partnerRepo.UpgradeReadyCount(ctx)
	gaps, _ := h.partnerRepo.GapSummaryAll(ctx)

	var sb strings.Builder
	sb.WriteString("📈 <b>Upgrade & Gaps</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	centerNames := map[string]string{
		"compute":      "Compute",
		"hybrid_cloud": "Hybrid Cloud",
		"networking":   "Networking",
	}

	// Ready for upgrade
	hasReady := false
	for _, cKey := range []string{"compute", "hybrid_cloud", "networking"} {
		ur, ok := ready[cKey]
		if !ok {
			continue
		}
		var parts []string
		for _, tier := range []string{"platinum", "gold", "silver"} {
			if cnt, ok := ur[tier]; ok && cnt > 0 {
				hasReady = true
				parts = append(parts, fmt.Sprintf("%s %s: <b>%d</b>",
					tierEmoji(tier), strings.Title(tier), cnt))
			}
		}
		if len(parts) > 0 {
			sb.WriteString(fmt.Sprintf("🟢 <b>%s</b>\n  %s\n", centerNames[cKey], strings.Join(parts, ", ")))
		}
	}
	if !hasReady {
		sb.WriteString("Пока нет партнёров, готовых к апгрейду.\n")
	}

	// Gap summary
	if len(gaps) > 0 {
		sb.WriteString("\n<b>Гэпы до следующего тира</b>\n")
		for _, g := range gaps {
			name := centerNames[g.Center]
			if name == "" {
				name = g.Center
			}
			sb.WriteString(fmt.Sprintf("\n<b>%s</b>\n", name))
			if g.VolumeCount > 0 {
				sb.WriteString(fmt.Sprintf("  💰 Volume: %d партн., gap %s\n",
					g.VolumeCount, formatNumberAdmin(g.VolumeGap)))
			}
			if g.CertGapCount > 0 {
				sb.WriteString(fmt.Sprintf("  📜 Certs: %d партн. с гэпами\n", g.CertGapCount))
			}
		}
	}

	return sb.String()
}

func (h *AdminHandler) statsVolume(ctx context.Context) string {
	top, _ := h.partnerRepo.TopVolumePartners(ctx, 5)

	var sb strings.Builder
	sb.WriteString("💰 <b>Top Volume (Compute)</b>\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, t := range top {
		name := t.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		sb.WriteString(fmt.Sprintf("%d.  %s\n     <b>%s</b>\n",
			i+1, name, formatNumberAdmin(t.Volume)))
	}

	return sb.String()
}

func tierEmoji(tier string) string {
	switch tier {
	case "platinum":
		return "💎"
	case "gold":
		return "🥇"
	case "silver":
		return "🥈"
	default:
		return "🏷"
	}
}

func countryFlag(country string) string {
	flags := map[string]string{
		"Kazakhstan":   "🇰🇿",
		"Uzbekistan":   "🇺🇿",
		"Azerbaijan":   "🇦🇿",
		"Georgia":      "🇬🇪",
		"Kyrgyzstan":   "🇰🇬",
		"Armenia":      "🇦🇲",
		"Tajikistan":   "🇹🇯",
		"Turkmenistan": "🇹🇲",
	}
	if f, ok := flags[country]; ok {
		return f
	}
	return "🌍"
}

func formatNumberAdmin(n float64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("$%.0fK", n/1_000)
	}
	return fmt.Sprintf("$%.0f", n)
}

// HandleUsers shows pending users for admin approval.
func (h *AdminHandler) HandleUsers(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ManageUsers) {
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
	text := fmt.Sprintf("⏳ <b>Ожидают одобрения: %d</b>\n\n", len(pending))

	for _, u := range pending {
		text += fmt.Sprintf("• %s (@%s)\n", u.FullName, u.Username)
		tgIDStr := strconv.FormatInt(u.TelegramID, 10)
		rows = append(rows, []models.InlineKeyboardButton{
			{Text: "✅ User", CallbackData: "approve:" + tgIDStr},
			{Text: "👔 PBM", CallbackData: "role_pbm:" + tgIDStr},
			{Text: "📦 Distri", CallbackData: "role_distri:" + tgIDStr},
			{Text: "❌ Отклонить", CallbackData: "reject:" + tgIDStr},
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeHTML,
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
	if !rbac.Can(user, rbac.ManageUsers) {
		return
	}

	data := update.CallbackQuery.Data
	var action string
	var targetIDStr string

	if strings.HasPrefix(data, "approve:") {
		action = "approve"
		targetIDStr = strings.TrimPrefix(data, "approve:")
	} else if strings.HasPrefix(data, "role_pbm:") {
		action = "pbm"
		targetIDStr = strings.TrimPrefix(data, "role_pbm:")
	} else if strings.HasPrefix(data, "role_distri:") {
		action = "distri"
		targetIDStr = strings.TrimPrefix(data, "role_distri:")
	} else if strings.HasPrefix(data, "reject:") {
		action = "reject"
		targetIDStr = strings.TrimPrefix(data, "reject:")
	} else {
		return
	}

	targetTgID, err := strconv.ParseInt(targetIDStr, 10, 64)
	if err != nil {
		return
	}

	// Look up PBM user by telegram_id to get DB ID
	targetUser, err := h.userRepo.GetByTelegramIDAndBotType(ctx, targetTgID, "pbm")
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Пользователь не найден",
			ShowAlert:       true,
		})
		return
	}

	var newRole domain.Role
	var responseText string
	var userMsg string

	switch action {
	case "approve":
		newRole = domain.RoleUser
		responseText = "✅ Пользователь одобрен (User)"
		userMsg = "🎉 Ваш доступ к боту одобрен! Используйте /help для списка команд."
	case "pbm":
		newRole = domain.RolePBM
		responseText = "👔 Назначен PBM"
		userMsg = "🎉 Вы назначены как PBM! Используйте /help для списка команд."
	case "distri":
		newRole = domain.RoleDistri
		responseText = "📦 Назначен Distri"
		userMsg = "🎉 Вы назначены как Дистрибьютор! Используйте /help для списка команд."
	default:
		newRole = domain.RolePending
		responseText = "❌ Пользователь отклонён"
		userMsg = "❌ Ваш запрос на доступ отклонён."
	}

	if err := h.userRepo.SetRole(ctx, targetUser.ID, newRole); err != nil {
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

	// Edit the admin message to remove buttons and show result
	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text: fmt.Sprintf("%s\n\n👤 %s (@%s)",
				responseText, targetUser.FullName, targetUser.Username),
			ParseMode: models.ParseModeHTML,
		})
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: targetTgID,
		Text:   userMsg,
	})
}

// HandleChartCallback sends chart images based on callback data.
func (h *AdminHandler) HandleChartCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ViewCharts) {
		return
	}

	section := strings.TrimPrefix(update.CallbackQuery.Data, "chart:")

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "📊 Generating chart...",
	})

	var chatID int64
	if update.CallbackQuery.Message.Message != nil {
		chatID = update.CallbackQuery.Message.Message.Chat.ID
	}
	if chatID == 0 {
		return
	}

	var chartURL string
	var caption string

	switch section {
	case "pipeline":
		chartURL, caption = h.chartPipeline(ctx)
	case "fruit":
		chartURL, caption = h.chartFruit(ctx)
	case "risk":
		chartURL, caption = h.chartRisk(ctx)
	case "concentration":
		chartURL, caption = h.chartConcentration(ctx)
	default:
		return
	}

	if chartURL == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "❌ Не удалось создать график. Нет данных.",
		})
		return
	}

	navButtons := [][]models.InlineKeyboardButton{
		{
			{Text: "📈 Pipeline", CallbackData: "chart:pipeline"},
			{Text: "🎯 Fruit", CallbackData: "chart:fruit"},
		},
		{
			{Text: "⚠️ Risk", CallbackData: "chart:risk"},
			{Text: "🧩 Concentration", CallbackData: "chart:concentration"},
		},
		{
			{Text: "📋 Dashboard", CallbackData: "menu:stats"},
			{Text: "🔍 Поиск", CallbackData: "menu:search"},
		},
	}

	b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:    chatID,
		Photo:     &models.InputFileString{Data: chartURL},
		Caption:   caption,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: navButtons,
		},
	})
}

func (h *AdminHandler) chartPipeline(ctx context.Context) (string, string) {
	rows, err := h.partnerRepo.UpgradePipeline(ctx)
	if err != nil || len(rows) == 0 {
		return "", ""
	}

	var centers []string
	var ready, cert, vol, deep []int

	for _, r := range rows {
		centers = append(centers, r.Center)
		ready = append(ready, r.Ready)
		cert = append(cert, r.CertBlocked)
		vol = append(vol, r.VolBlocked)
		deep = append(deep, r.DeepGap)
	}

	url := chart.UpgradePipelineChart(centers, ready, cert, vol, deep)
	caption := "📈 <b>Upgrade Pipeline (Blockers)</b>\n" +
		"Shows why partners are not moving to the next tier."

	return url, caption
}

func (h *AdminHandler) chartFruit(ctx context.Context) (string, string) {
	rows, err := h.partnerRepo.LowHangingFruit(ctx, 8)
	if err != nil || len(rows) == 0 {
		return "", ""
	}

	var names []string
	var volumes, gaps []float64

	for _, r := range rows {
		name := r.PartnerName
		if len(name) > 17 {
			name = name[:15] + ".."
		}
		names = append(names, fmt.Sprintf("%s (%s)", name, strings.Title(r.Tier)))
		volumes = append(volumes, r.Volume)
		gaps = append(gaps, r.Gap)
	}

	url := chart.LowHangingFruitChart(names, volumes, gaps)
	caption := "🎯 <b>Low-Hanging Fruit</b>\n" +
		"Partners at 80%-99% of next tier volume. Quick wins!"

	return url, caption
}

func (h *AdminHandler) chartRisk(ctx context.Context) (string, string) {
	safe, volRisk, certRisk, deepRisk, err := h.partnerRepo.RetentionRisk(ctx)
	if err != nil {
		return "", ""
	}

	url := chart.RetentionRiskChart(safe, volRisk, certRisk, deepRisk)

	total := safe + volRisk + certRisk + deepRisk
	caption := fmt.Sprintf("⚠️ <b>FY27 Retention Risk</b>\n"+
		"Total %d Platinum/Gold partners evaluated against their current tier requirements.", total)

	return url, caption
}

func (h *AdminHandler) chartConcentration(ctx context.Context) (string, string) {
	// Standardize on Compute for this general dashboard
	top3, next7, rest, err := h.partnerRepo.VolumeConcentration(ctx, "compute")
	if err != nil || (top3+next7+rest) == 0 {
		return "", ""
	}

	url := chart.ConcentrationChart(top3, next7, rest, "Compute")

	total := top3 + next7 + rest
	top3Pct := 0.0
	if total > 0 {
		top3Pct = (top3 / total) * 100
	}

	caption := fmt.Sprintf("🧩 <b>Compute Revenue Concentration</b>\n"+
		"Top 3 partners control %.1f%% of total volume.", top3Pct)

	return url, caption
}

// HandlePartnerApproval handles admin approval/rejection of partner bot users.
// Callback format: "papprove:123" or "preject:123" where 123 is partner user DB ID.
func (h *AdminHandler) HandlePartnerApproval(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ManageUsers) {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Access denied",
			ShowAlert:       true,
		})
		return
	}

	data := update.CallbackQuery.Data
	isApprove := strings.HasPrefix(data, "papprove:")
	isDistri := strings.HasPrefix(data, "pdistri:")

	var prefix string
	switch {
	case isApprove:
		prefix = "papprove:"
	case isDistri:
		prefix = "pdistri:"
	default:
		prefix = "preject:"
	}

	userIDStr := strings.TrimPrefix(data, prefix)
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return
	}

	partnerUser, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ User not found",
			ShowAlert:       true,
		})
		return
	}

	if isApprove || isDistri {
		var approveErr error
		var roleLabel string
		if isDistri {
			approveErr = h.userRepo.ApproveDistri(ctx, userID)
			roleLabel = "📦 Distri"
		} else {
			approveErr = h.userRepo.ApprovePartner(ctx, userID)
			roleLabel = "✅ Partner"
		}

		if approveErr != nil {
			b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Error: " + approveErr.Error(),
				ShowAlert:       true,
			})
			return
		}

		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            roleLabel + " Approved!",
		})

		// Edit admin message (PBM bot's own message)
		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("%s <b>Одобрено</b>\n\n"+
					"👤 %s\n🏢 %s\n📧 %s",
					roleLabel, partnerUser.FullName, partnerUser.CompanyName, partnerUser.Email),
				ParseMode: models.ParseModeHTML,
			})
		}

		// Edit partner's onboarding message via partner bot token
		if partnerUser.OnboardMsgID != nil && h.partnerBotToken != "" {
			if err := tgapi.EditMessageText(h.partnerBotToken, tgapi.EditMessageTextParams{
				ChatID:    partnerUser.TelegramID,
				MessageID: *partnerUser.OnboardMsgID,
				Text:      "✅ <b>Доступ подтверждён!</b>\n\nИспользуйте /status для просмотра\nкарточки вашей компании.",
				ParseMode: "HTML",
			}); err != nil {
				slog.Error("failed to edit partner message", "error", err)
			}
		}

	} else {
		// Reject: ask for comment
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		if update.CallbackQuery.Message.Message != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("❌ <b>Отклонение</b>\n\n"+
					"👤 %s\n📧 %s\n\n"+
					"Напишите причину отказа или нажмите кнопку:",
					partnerUser.FullName, partnerUser.Email),
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{
						{{Text: "Без комментария", CallbackData: fmt.Sprintf("prejectconfirm:%d:", userID)}},
					},
				},
			})
		}

		h.userRepo.SetOnboardData(ctx, userID, "rejected", partnerUser.FullName, partnerUser.CompanyName, partnerUser.Email)

		// Track pending reject for comment input
		h.SetPendingReject(update.CallbackQuery.From.ID, userID)
	}
}

// HandlePartnerRejectConfirm handles the "no comment" reject confirm.
func (h *AdminHandler) HandlePartnerRejectConfirm(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if !rbac.Can(user, rbac.ManageUsers) {
		return
	}

	data := update.CallbackQuery.Data
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

	partnerUser, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return
	}

	h.userRepo.ResetOnboard(ctx, userID)

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		Text:            "❌ Rejected",
	})

	commentLine := ""
	if comment != "" {
		commentLine = fmt.Sprintf("\n💬 %s", comment)
	}

	// Edit admin message
	if update.CallbackQuery.Message.Message != nil {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text: fmt.Sprintf("❌ <b>Отклонено</b>\n\n👤 %s\n📧 %s%s",
				partnerUser.FullName, partnerUser.Email, commentLine),
			ParseMode: models.ParseModeHTML,
		})
	}

	// Edit partner's onboarding message via partner bot token
	if partnerUser.OnboardMsgID != nil && h.partnerBotToken != "" {
		tgapi.EditMessageText(h.partnerBotToken, tgapi.EditMessageTextParams{
			ChatID:    partnerUser.TelegramID,
			MessageID: *partnerUser.OnboardMsgID,
			Text:      fmt.Sprintf("❌ <b>Запрос отклонён</b>%s\n\nНажмите /start чтобы подать заново", commentLine),
			ParseMode: "HTML",
		})
	}
}
