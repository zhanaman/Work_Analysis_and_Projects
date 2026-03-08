package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// PartnerHandler handles partner detail views.
type PartnerHandler struct {
	partnerRepo *storage.PartnerRepo
}

// NewPartnerHandler creates a new PartnerHandler.
func NewPartnerHandler(repo *storage.PartnerRepo) *PartnerHandler {
	return &PartnerHandler{partnerRepo: repo}
}

// HandleCallback processes callback queries like "partner:123".
func (h *PartnerHandler) HandleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	if !strings.HasPrefix(data, "partner:") {
		return
	}

	idStr := strings.TrimPrefix(data, "partner:")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return
	}

	partner, err := h.partnerRepo.GetByID(ctx, id)
	if err != nil {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "❌ Партнёр не найден",
			ShowAlert:       true,
		})
		return
	}

	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	card := formatPartnerCard(partner)

	var chatID int64
	var msgID int
	if update.CallbackQuery.Message.Message != nil {
		chatID = update.CallbackQuery.Message.Message.Chat.ID
		msgID = update.CallbackQuery.Message.Message.ID
	}
	if chatID == 0 {
		return
	}

	buttons := [][]models.InlineKeyboardButton{
		{
			{Text: "🔍 Новый поиск", CallbackData: "menu:search"},
		},
	}

	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    chatID,
		MessageID: msgID,
		Text:      card,
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: models.InlineKeyboardMarkup{
			InlineKeyboard: buttons,
		},
	})
}

// formatPartnerCard builds a complete partner card with inline details.
func formatPartnerCard(p *domain.Partner) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("🏢 <b>%s</b>\n", p.Name))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📍 %s  •  <code>%s</code>\n", p.Country, p.PartyID))

	// Partner type
	ptype := partnerType(p)
	if ptype != "" {
		sb.WriteString(fmt.Sprintf("🏷 %s\n", ptype))
	}

	sb.WriteString("\n")

	// Each center — full inline details
	type centerDef struct {
		center     domain.Center
		membership string
		name       string
	}
	centers := []centerDef{
		{domain.CenterCompute, p.MembershipCompute, "Compute"},
		{domain.CenterHybridCloud, p.MembershipHC, "Hybrid Cloud"},
		{domain.CenterNetworking, p.MembershipNetworking, "Networking"},
	}

	totalGaps := 0
	allReady := true

	for _, ci := range centers {
		if ci.membership == "" {
			continue
		}

		var centerTiers []domain.PartnerTier
		for _, t := range p.Tiers {
			if t.Center == ci.center {
				centerTiers = append(centerTiers, t)
			}
		}

		readiness := domain.CalculateReadiness(ci.membership, centerTiers, ci.center)

		if readiness == nil || readiness.NextTier == "" {
			// Max tier
			sb.WriteString(fmt.Sprintf("<b>%s</b>  %s ✅\n\n", ci.name, tierBadge(ci.membership)))
		} else if readiness.IsReady {
			// Ready for upgrade
			sb.WriteString(fmt.Sprintf("<b>%s</b>  %s → %s ✅\n\n",
				ci.name, tierBadge(ci.membership), tierBadge(string(readiness.NextTier))))
		} else {
			// Has gaps — show ONLY what's failing
			allReady = false
			gapCount := len(readiness.Blockers)
			totalGaps += gapCount

			sb.WriteString(fmt.Sprintf("<b>%s</b>  %s → %s\n",
				ci.name, tierBadge(ci.membership), tierBadge(string(readiness.NextTier))))
			sb.WriteString(fmt.Sprintf("%d gaps:\n", gapCount))

			// Volume — only if FAILED
			if readiness.Volume.Required > 0 && !readiness.Volume.Met {
				sb.WriteString(fmt.Sprintf("  ❌ Volume %s / %s\n",
					formatNumber(readiness.Volume.Actuals),
					formatNumber(readiness.Volume.Required)))
			}

			// SRI — only if FAILED
			if readiness.SRI.Required > 0 && readiness.SRI.Required < domain.SRISentinel && !readiness.SRI.Met {
				sb.WriteString(fmt.Sprintf("  ❌ SRI %.1f / %.1f\n",
					readiness.SRI.Actuals, readiness.SRI.Required))
			}

			// Certs — only FAILED ones
			certGaps := formatCertGapsOnly(readiness)
			if certGaps != "" {
				sb.WriteString(fmt.Sprintf("  ❌ %s\n", certGaps))
			}

			sb.WriteString("\n")
		}
	}

	// FY27 Readiness + L&R
	sb.WriteString("\n")
	if allReady {
		sb.WriteString("🟢 <b>FY27 Ready</b>\n")
	} else {
		sb.WriteString("🔴 <b>FY27 Not Ready</b>\n")
	}

	if p.LRStatus != "" {
		if strings.EqualFold(p.LRStatus, "Yes") {
			sb.WriteString("📋 L&R ✅\n")
		} else {
			sb.WriteString("📋 L&R ❌\n")
		}
	}

	return sb.String()
}

// partnerType builds a human-readable partner type string.
func partnerType(p *domain.Partner) string {
	var types []string
	if p.BusinessRelSP != "" {
		types = append(types, p.BusinessRelSP)
	}
	if p.BusinessRelSvc != "" {
		types = append(types, p.BusinessRelSvc)
	}
	if p.BusinessRelSI != "" {
		types = append(types, p.BusinessRelSI)
	}
	return strings.Join(types, " | ")
}

// formatCertsCompact returns "Sales 2/3 ❌ • ASE 2/2 ✅"
func formatCertsCompact(r *domain.TierReadiness) string {
	var parts []string
	add := func(name string, have, need int, met bool) {
		if need > 0 {
			icon := "✅"
			if !met {
				icon = "❌"
			}
			parts = append(parts, fmt.Sprintf("%s %d/%d %s", name, have, need, icon))
		}
	}
	add("Sales", r.Certs.SalesHave, r.Certs.SalesNeed, r.Certs.SalesMet)
	add("ATP", r.Certs.ATPHave, r.Certs.ATPNeed, r.Certs.ATPMet)
	add("ASE", r.Certs.ASEHave, r.Certs.ASENeed, r.Certs.ASEMet)
	add("MASE", r.Certs.MASEHave, r.Certs.MASENeed, r.Certs.MASEMet)
	return strings.Join(parts, " • ")
}

// formatCertGapsOnly returns only FAILED certs, e.g. "MASE 0/1, Sales 0/2"
func formatCertGapsOnly(r *domain.TierReadiness) string {
	var parts []string
	add := func(name string, have, need int, met bool) {
		if need > 0 && !met {
			parts = append(parts, fmt.Sprintf("%s %d/%d", name, have, need))
		}
	}
	add("Sales", r.Certs.SalesHave, r.Certs.SalesNeed, r.Certs.SalesMet)
	add("ATP", r.Certs.ATPHave, r.Certs.ATPNeed, r.Certs.ATPMet)
	add("ASE", r.Certs.ASEHave, r.Certs.ASENeed, r.Certs.ASEMet)
	add("MASE", r.Certs.MASEHave, r.Certs.MASENeed, r.Certs.MASEMet)
	return strings.Join(parts, ", ")
}

// tierBadge returns "🥇 Gold" etc.
func tierBadge(tier string) string {
	t := strings.ToLower(tier)
	switch {
	case strings.Contains(t, "platinum"):
		return "💎 Platinum"
	case strings.Contains(t, "gold"):
		return "🥇 Gold"
	case strings.Contains(t, "silver"):
		return "🥈 Silver"
	case strings.Contains(t, "business"):
		return "🏷 BP"
	default:
		if tier == "" {
			return "—"
		}
		return tier
	}
}

func tierDisplay(tier string) string { return tierBadge(tier) }

func formatNumber(n float64) string {
	if n == 0 {
		return "$0"
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", n/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("$%.0fK", n/1_000)
	}
	return fmt.Sprintf("$%.0f", n)
}
