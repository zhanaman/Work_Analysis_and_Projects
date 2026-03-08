package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// SRISentinel is the placeholder value Excel uses for "N/A" SRI.
const SRISentinel = 9999999

// TierReadiness describes readiness for the next tier level in one center.
type TierReadiness struct {
	Center      Center `json:"center"`
	CurrentTier Tier   `json:"current_tier"`
	NextTier    Tier   `json:"next_tier"`
	IsReady     bool   `json:"is_ready"` // all criteria met for next tier

	// Individual criteria status
	Volume    GapStatus `json:"volume"`
	SRI       GapStatus `json:"sri"`
	Certs     CertGap   `json:"certs"`
	GrowthPlan bool    `json:"growth_plan"`

	// Human-readable output
	Blockers        []string `json:"blockers"`
	Recommendations []string `json:"recommendations"`
}

// GapStatus tracks actuals vs required for a metric.
type GapStatus struct {
	Actuals  float64 `json:"actuals"`
	Required float64 `json:"required"`
	Pct      float64 `json:"pct"`
	Met      bool    `json:"met"`
	Gap      float64 `json:"gap"` // how much is missing (0 if met)
}

// CertGap tracks certification requirements.
type CertGap struct {
	SalesHave int `json:"sales_have"`
	SalesNeed int `json:"sales_need"`
	SalesMet  bool `json:"sales_met"`

	ATPHave int `json:"atp_have"`
	ATPNeed int `json:"atp_need"`
	ATPMet  bool `json:"atp_met"`

	ASEHave int `json:"ase_have"`
	ASENeed int `json:"ase_need"`
	ASEMet  bool `json:"ase_met"`

	MASEHave int `json:"mase_have"`
	MASENeed int `json:"mase_need"`
	MASEMet  bool `json:"mase_met"`
}

// CalculateReadiness computes readiness for the target tier in a given center.
// tiers should contain all tier rows for one partner in one center.
func CalculateReadiness(membership string, targetTier Tier, tiers []PartnerTier, center Center) *TierReadiness {
	currentTier := ParseMembershipTier(membership)
	if targetTier == "" {
		// Already at max tier
		return &TierReadiness{
			Center:      center,
			CurrentTier: currentTier,
			NextTier:    "",
			IsReady:     true,
		}
	}

	// Find the target tier data
	var targetTierData *PartnerTier
	for i := range tiers {
		if tiers[i].Tier == targetTier {
			targetTierData = &tiers[i]
			break
		}
	}

	if targetTierData == nil {
		return &TierReadiness{
			Center:      center,
			CurrentTier: currentTier,
			NextTier:    targetTier,
			Blockers:    []string{"No tier data available for " + string(targetTier)},
		}
	}

	r := &TierReadiness{
		Center:      center,
		CurrentTier: currentTier,
		NextTier:    targetTier,
	}

	// Volume gap
	r.Volume = GapStatus{
		Actuals:  targetTierData.VolumeActuals,
		Required: targetTierData.Threshold,
		Pct:      targetTierData.VolumePct,
		Met:      targetTierData.VolumeStatus,
	}
	if !r.Volume.Met && r.Volume.Required > 0 {
		r.Volume.Gap = r.Volume.Required - r.Volume.Actuals
		if r.Volume.Gap < 0 {
			r.Volume.Gap = 0
		}
		r.Blockers = append(r.Blockers, fmt.Sprintf("Volume: %s / %s (%s) — need +%s",
			formatMoney(r.Volume.Actuals), formatMoney(r.Volume.Required),
			formatPct(r.Volume.Pct), formatMoney(r.Volume.Gap)))
		r.Recommendations = append(r.Recommendations,
			fmt.Sprintf("Focus pipeline to close volume gap of %s", formatMoney(r.Volume.Gap)))
	}

	// SRI gap (Gold/Platinum only)
	if targetTier == TierGold || targetTier == TierPlatinum {
		r.SRI = GapStatus{
			Actuals:  targetTierData.SRI,
			Required: targetTierData.SRIRequired,
			Pct:      targetTierData.SRIPct,
			Met:      targetTierData.SRIStatus,
		}
		if !r.SRI.Met && r.SRI.Required > 0 && r.SRI.Required < SRISentinel {
			r.SRI.Gap = r.SRI.Required - r.SRI.Actuals
			if r.SRI.Gap < 0 {
				r.SRI.Gap = 0
			}
			r.Blockers = append(r.Blockers, fmt.Sprintf("SRI: %.2f / %.2f required",
				r.SRI.Actuals, r.SRI.Required))
			r.Recommendations = append(r.Recommendations,
				"Increase services attach rate to improve SRI")
		}
	}

	// Cert gap
	salesH, salesN := parseCertFraction(targetTierData.SalesCertified)
	atpH, atpN := parseCertFraction(targetTierData.ATPCurrent)
	aseH, aseN := parseCertFraction(targetTierData.ASECurrent)
	maseH, maseN := parseCertFraction(targetTierData.MASECurrent)

	r.Certs = CertGap{
		SalesHave: salesH, SalesNeed: salesN, SalesMet: salesH >= salesN,
		ATPHave: atpH,   ATPNeed: atpN,   ATPMet: atpH >= atpN,
		ASEHave: aseH,   ASENeed: aseN,   ASEMet: aseH >= aseN,
		MASEHave: maseH, MASENeed: maseN, MASEMet: maseH >= maseN,
	}

	if !targetTierData.CertStatus {
		var certBlockers []string
		if !r.Certs.SalesMet && salesN > 0 {
			certBlockers = append(certBlockers, fmt.Sprintf("Sales: %d/%d", salesH, salesN))
		}
		if !r.Certs.ATPMet && atpN > 0 {
			certBlockers = append(certBlockers, fmt.Sprintf("ATP: %d/%d", atpH, atpN))
		}
		if !r.Certs.ASEMet && aseN > 0 {
			certBlockers = append(certBlockers, fmt.Sprintf("ASE: %d/%d", aseH, aseN))
		}
		if !r.Certs.MASEMet && maseN > 0 {
			certBlockers = append(certBlockers, fmt.Sprintf("MASE: %d/%d", maseH, maseN))
		}
		if len(certBlockers) > 0 {
			r.Blockers = append(r.Blockers, "Certs: "+strings.Join(certBlockers, ", "))
			r.Recommendations = append(r.Recommendations,
				"Enroll team members in required certification programs")
		}
	}

	// Growth plan
	r.GrowthPlan = targetTierData.GrowthPlanStatus
	if !r.GrowthPlan {
		r.Blockers = append(r.Blockers, "Growth Plan not active")
		r.Recommendations = append(r.Recommendations,
			"Submit and activate Growth Plan for "+string(targetTier)+" tier")
	}

	// Overall readiness
	r.IsReady = targetTierData.CriteriaMet

	return r
}

// ParseMembershipTier extracts tier from membership string like "Silver Partner" or "Compute Gold".
func ParseMembershipTier(membership string) Tier {
	m := strings.ToLower(membership)
	switch {
	case strings.Contains(m, "platinum"):
		return TierPlatinum
	case strings.Contains(m, "gold"):
		return TierGold
	case strings.Contains(m, "silver"):
		return TierSilver
	case strings.Contains(m, "business"):
		return TierBusiness
	default:
		return TierBusiness
	}
}

// parseCertFraction parses "2/3" into have=2, need=3.
func parseCertFraction(s string) (have, need int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0
	}
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	h, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	n, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return h, n
}

func formatMoney(v float64) string {
	if v >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", v/1_000_000)
	}
	if v >= 1_000 {
		return fmt.Sprintf("$%.0fK", v/1_000)
	}
	return fmt.Sprintf("$%.0f", v)
}

func formatPct(v float64) string {
	return fmt.Sprintf("%.0f%%", v)
}
