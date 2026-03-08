package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/bot/middleware"
	"github.com/anonimouskz/pbm-partner-bot/internal/chart"
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

// HandleStats shows the compact stats dashboard hub.
func (h *AdminHandler) HandleStats(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAuthorized() {
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
	text := fmt.Sprintf("⏳ <b>Ожидают одобрения: %d</b>\n\n", len(pending))

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

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: targetID,
		Text:   "🎉 Ваш доступ к боту одобрен! Используйте /help для списка команд.",
	})
}

// HandleChartCallback sends chart images based on callback data.
func (h *AdminHandler) HandleChartCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	user := middleware.UserFromContext(ctx)
	if user == nil || !user.IsAuthorized() {
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
