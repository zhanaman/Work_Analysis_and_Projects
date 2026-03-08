package domain

import (
	"testing"
)

func TestParseCertFraction(t *testing.T) {
	tests := []struct {
		input     string
		wantHave  int
		wantNeed  int
	}{
		{"2/3", 2, 3},
		{"0/1", 0, 1},
		{"3/3", 3, 3},
		{"", 0, 0},
		{"invalid", 0, 0},
		{" 1 / 2 ", 1, 2},
	}

	for _, tt := range tests {
		have, need := parseCertFraction(tt.input)
		if have != tt.wantHave || need != tt.wantNeed {
			t.Errorf("parseCertFraction(%q) = (%d, %d), want (%d, %d)",
				tt.input, have, need, tt.wantHave, tt.wantNeed)
		}
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "$0"},
		{500, "$500"},
		{1_500, "$2K"},
		{150_000, "$150K"},
		{1_500_000, "$1.5M"},
		{15_000_000, "$15.0M"},
	}

	for _, tt := range tests {
		got := formatMoney(tt.input)
		if got != tt.want {
			t.Errorf("formatMoney(%.0f) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTierOrder(t *testing.T) {
	if TierOrder(TierBusiness) >= TierOrder(TierSilver) {
		t.Error("Business should be < Silver")
	}
	if TierOrder(TierSilver) >= TierOrder(TierGold) {
		t.Error("Silver should be < Gold")
	}
	if TierOrder(TierGold) >= TierOrder(TierPlatinum) {
		t.Error("Gold should be < Platinum")
	}
}

func TestNextTier(t *testing.T) {
	tests := []struct {
		input Tier
		want  Tier
	}{
		{TierBusiness, TierSilver},
		{TierSilver, TierGold},
		{TierGold, TierPlatinum},
		{TierPlatinum, ""},
	}

	for _, tt := range tests {
		got := NextTier(tt.input)
		if got != tt.want {
			t.Errorf("NextTier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCalculateReadiness_MaxTier(t *testing.T) {
	r := CalculateReadiness("Compute Platinum Partner", nil, CenterCompute)
	if r == nil {
		t.Fatal("expected non-nil readiness")
	}
	if r.CurrentTier != TierPlatinum {
		t.Errorf("CurrentTier = %q, want %q", r.CurrentTier, TierPlatinum)
	}
	if r.NextTier != "" {
		t.Errorf("NextTier = %q, want empty", r.NextTier)
	}
	if !r.IsReady {
		t.Error("expected IsReady = true for max tier")
	}
}

func TestCalculateReadiness_SilverToGold(t *testing.T) {
	tiers := []PartnerTier{
		{
			Center:         CenterCompute,
			Tier:           TierGold,
			CriteriaMet:    false,
			VolumeStatus:   false,
			VolumeActuals:  555_000,
			Threshold:      1_000_000,
			VolumePct:      55,
			CertStatus:     false,
			SalesCertified: "0/2",
			ASECurrent:     "0/2",
			MASECurrent:    "",
			ATPCurrent:     "1/1",
			SRIStatus:      true,
			SRI:            1.76,
			SRIRequired:    0.80,
			GrowthPlanStatus: false,
		},
	}

	r := CalculateReadiness("Compute Silver Partner", tiers, CenterCompute)
	if r == nil {
		t.Fatal("expected non-nil readiness")
	}
	if r.CurrentTier != TierSilver {
		t.Errorf("CurrentTier = %q, want Silver", r.CurrentTier)
	}
	if r.NextTier != TierGold {
		t.Errorf("NextTier = %q, want Gold", r.NextTier)
	}
	if r.IsReady {
		t.Error("expected IsReady = false")
	}

	// Volume gap
	if r.Volume.Met {
		t.Error("volume should not be met")
	}
	if r.Volume.Actuals != 555_000 {
		t.Errorf("Volume.Actuals = %.0f, want 555000", r.Volume.Actuals)
	}
	if r.Volume.Gap != 445_000 {
		t.Errorf("Volume.Gap = %.0f, want 445000", r.Volume.Gap)
	}

	// SRI should be met
	if !r.SRI.Met {
		t.Error("SRI should be met (1.76 >= 0.80)")
	}

	// Certs
	if r.Certs.SalesMet {
		t.Error("Sales should not be met (0/2)")
	}
	if r.Certs.SalesHave != 0 || r.Certs.SalesNeed != 2 {
		t.Errorf("Sales = %d/%d, want 0/2", r.Certs.SalesHave, r.Certs.SalesNeed)
	}

	// Should have blockers
	if len(r.Blockers) == 0 {
		t.Error("expected blockers")
	}
}

func TestCalculateReadiness_Ready(t *testing.T) {
	tiers := []PartnerTier{
		{
			Center:           CenterCompute,
			Tier:             TierSilver,
			CriteriaMet:      true,
			VolumeStatus:     true,
			VolumeActuals:    200_000,
			Threshold:        150_000,
			VolumePct:        133,
			CertStatus:       true,
			SalesCertified:   "1/1",
			ATPCurrent:       "1/1",
			ASECurrent:       "",
			GrowthPlanStatus: true,
		},
	}

	r := CalculateReadiness("Compute Business Partner", tiers, CenterCompute)
	if r == nil {
		t.Fatal("expected non-nil readiness")
	}
	if !r.IsReady {
		t.Error("expected IsReady = true (all criteria met)")
	}
	if len(r.Blockers) != 0 {
		t.Errorf("expected 0 blockers, got %d: %v", len(r.Blockers), r.Blockers)
	}
}

func TestParseMembershipTier(t *testing.T) {
	tests := []struct {
		input string
		want  Tier
	}{
		{"Compute Silver Partner", TierSilver},
		{"Hybrid Cloud Gold", TierGold},
		{"Networking Business Partner", TierBusiness},
		{"Compute Platinum Partner", TierPlatinum},
		{"", TierBusiness},
		{"Unknown", TierBusiness},
	}

	for _, tt := range tests {
		got := parseMembershipTier(tt.input)
		if got != tt.want {
			t.Errorf("parseMembershipTier(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
