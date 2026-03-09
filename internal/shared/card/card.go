// Package card provides shared partner card formatting for both PBM and Partner bots.
package card

import (
	"fmt"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
)

// FormatPartnerCard builds a complete partner card with inline details.
// viewMode: "retention" or "upgrade".
func FormatPartnerCard(p *domain.Partner, viewMode string) string {
	// Validate viewMode
	if viewMode != "retention" && viewMode != "upgrade" {
		viewMode = "retention"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("🏢 <b>%s</b>\n", p.Name))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("📍 %s  •  <code>%s</code>\n", p.Country, p.PartyID))

	// Partner type
	ptype := PartnerType(p)
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

		currentTier := domain.ParseMembershipTier(ci.membership)
		var targetTier domain.Tier
		if viewMode == "upgrade" {
			targetTier = domain.NextTier(currentTier)
		} else {
			targetTier = currentTier
		}

		readiness := domain.CalculateReadiness(ci.membership, targetTier, centerTiers, ci.center)

		if targetTier == "" {
			sb.WriteString(fmt.Sprintf("<b>%s</b>  %s ✅\n\n", ci.name, TierBadge(ci.membership)))
		} else if readiness.IsReady {
			if viewMode == "upgrade" {
				sb.WriteString(fmt.Sprintf("<b>%s</b>  %s → %s ✅\n\n",
					ci.name, TierBadge(ci.membership), TierBadge(string(readiness.NextTier))))
			} else {
				sb.WriteString(fmt.Sprintf("<b>%s</b>  %s (Retention) ✅\n\n",
					ci.name, TierBadge(ci.membership)))
			}
		} else {
			allReady = false
			gapCount := len(readiness.Blockers)

			if viewMode == "upgrade" {
				sb.WriteString(fmt.Sprintf("<b>%s</b>  %s → %s\n",
					ci.name, TierBadge(ci.membership), TierBadge(string(readiness.NextTier))))
			} else {
				sb.WriteString(fmt.Sprintf("<b>%s</b>  %s (Retention)\n",
					ci.name, TierBadge(ci.membership)))
			}
			sb.WriteString(fmt.Sprintf("%d gaps:\n", gapCount))

			isNoData := len(readiness.Blockers) > 0 && strings.HasPrefix(readiness.Blockers[0], "No tier data")

			if isNoData {
				for _, b := range readiness.Blockers {
					sb.WriteString(fmt.Sprintf("  ❌ %s\n", b))
				}
			} else {
				if readiness.Volume.Required > 0 && !readiness.Volume.Met {
					sb.WriteString(fmt.Sprintf("  ❌ Volume %s / %s\n",
						FormatNumber(readiness.Volume.Actuals),
						FormatNumber(readiness.Volume.Required)))
				}
				if readiness.SRI.Required > 0 && readiness.SRI.Required < domain.SRISentinel && !readiness.SRI.Met {
					sb.WriteString(fmt.Sprintf("  ❌ SRI %.1f / %.1f\n",
						readiness.SRI.Actuals, readiness.SRI.Required))
				}
				certGaps := FormatCertGapsOnly(readiness)
				if certGaps != "" {
					sb.WriteString(fmt.Sprintf("  ❌ %s\n", certGaps))
				}
				if !readiness.GrowthPlan {
					sb.WriteString("  ❌ Growth Plan not active\n")
				}
			}

			sb.WriteString("\n")
		}
	}

	// FY27 Readiness + L&R
	sb.WriteString("\n")
	if viewMode == "retention" {
		if allReady {
			sb.WriteString("🟢 <b>FY27 Ready</b>\n")
		} else {
			sb.WriteString("🔴 <b>FY27 Not Ready</b>\n")
		}
	} else {
		if allReady {
			sb.WriteString("🟢 <b>Ready to Upgrade</b>\n")
		} else {
			sb.WriteString("🟡 <b>Blocked from Upgrade</b>\n")
		}
	}

	if p.LRStatus != "" {
		if strings.EqualFold(p.LRStatus, "Yes") {
			sb.WriteString("📋 L&R ✅\n")
		} else {
			sb.WriteString("📋 L&R ❌\n")
		}
	}

	refreshDate := ""
	if len(p.Revenue) > 0 {
		refreshDate = p.Revenue[0].RefreshDate
	}
	if refreshDate != "" {
		sb.WriteString(fmt.Sprintf("\n📅 Данные от: %s\n", refreshDate))
	} else if !p.ImportedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("\n📅 Импорт: %s\n", p.ImportedAt.Format("2006-01-02")))
	}

	return sb.String()
}

// PartnerType returns the combined business relationship string.
func PartnerType(p *domain.Partner) string {
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

// FormatCertGapsOnly formats certification gap details.
func FormatCertGapsOnly(r *domain.TierReadiness) string {
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

// TierBadge returns the emoji badge for a tier.
func TierBadge(tier string) string {
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

// FormatNumber formats a dollar amount.
func FormatNumber(n float64) string {
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
