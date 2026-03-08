package parser

import (
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
)

// CenterConfig defines how to parse a specific center sheet.
type CenterConfig struct {
	SheetName string
	HeaderRow int           // 1 for Networking, 2 for Compute/HC
	Center    domain.Center // domain.CenterCompute etc.
	Prefix    string        // "Compute", "Hybrid Cloud", "Networking"
}

// CenterConfigs for all 3 current-year center sheets.
var CenterConfigs = []CenterConfig{
	{
		SheetName: "Compute",
		HeaderRow: 2,
		Center:    domain.CenterCompute,
		Prefix:    "Compute",
	},
	{
		SheetName: "HC",
		HeaderRow: 2,
		Center:    domain.CenterHybridCloud,
		Prefix:    "Hybrid Cloud",
	},
	{
		SheetName: "Networking",
		HeaderRow: 1,
		Center:    domain.CenterNetworking,
		Prefix:    "Networking",
	},
}

// columnMap maps semantic field names to column indices.
type columnMap map[string]int

// buildColumnMap creates a mapping from header names to column indices.
// Uses exact header matching based on real Excel column analysis.
func buildColumnMap(headers []string) columnMap {
	cm := make(columnMap)
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		// Store by exact header name (case-preserved)
		cm[h] = i
		// Also store lowercase for fuzzy lookups
		cm[strings.ToLower(h)] = i
	}
	return cm
}

// get retrieves a cell value by column header name.
func (cm columnMap) get(row []string, header string) string {
	idx, ok := cm[header]
	if !ok {
		// Try case-insensitive
		idx, ok = cm[strings.ToLower(header)]
	}
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// partnerColumns maps the shared identity columns (same across all center sheets).
var partnerColumns = struct {
	Theater           string
	SubRegion         string
	HPEOrg            string
	Country           string
	CountryCode       string
	HQPartnerIDT1    string
	HQPartnerIDT2    string
	HQPartnerIDSP    string
	PartyID           string
	PartyName         string
	CountryEntityID   string
	CountryEntityName string
	GlobalEntityID    string
	GlobalEntityName  string
	BusinessRelSP     string
	BusinessRelSvc    string
	BusinessRelSI     string
	ActiveAgreement   string
	LRStatus          string
	LRStatusStart     string
	LRStatusEnd       string
	SPAgreement       string
	MembershipCompute string
	MembershipHC      string
	MembershipNet     string
	TriplePlatinum    string
	IntlPartner       string
}{
	Theater:           "Theater",
	SubRegion:         "Sub-Region",
	HPEOrg:            "HPE Organization",
	Country:           "Country",
	CountryCode:       "Party Country Code",
	HQPartnerIDT1:    "HQ Partner ID (T1 Solution Provider)",
	HQPartnerIDT2:    "HQ Partner ID (T2 Solution Provider)",
	HQPartnerIDSP:    "HQ Partner ID (Service Provider)",
	PartyID:           "Signing Entity Party ID",
	PartyName:         "Party Name",
	CountryEntityID:   "Country Entity ID",
	CountryEntityName: "Country Entity Name",
	GlobalEntityID:    "Global Entity ID",
	GlobalEntityName:  "Global Entity Name",
	BusinessRelSP:     "Current Business Relationship (Solution Provider)",
	BusinessRelSvc:    "Current Business Relationship (Service Provider)",
	BusinessRelSI:     "Current Business Relationship (System Integrator)",
	ActiveAgreement:   "Active Agreement",
	LRStatus:          "L&R Status",
	LRStatusStart:     "L&R Status Start Date",
	LRStatusEnd:       "L&R Status End Date",
	SPAgreement:       "Service Provider Agreement",
	MembershipCompute: "Current Membership Compute",
	MembershipHC:      "Current Membership Hybrid Cloud",
	MembershipNet:     "Current Membership Networking",
	TriplePlatinum:    "Triple Platinum Plus",
	IntlPartner:       "International Partner membership",
}

// tierColumnTemplate defines the column name pattern for tier data.
// Use fmt.Sprintf with (prefix, tierName) to build column names.
type tierColumnTemplate struct {
	CriteriaMet       string // "%s %s Criteria Met"
	TierStartDate     string // "%s %s Final (start date)"
	TierEndDate       string // "%s %s Final (end date)" or "%s %s final (end date)"
	VolumeStatus      string // "%s %s Volume Status"
	GrowthPlanStatus  string // "%s %s Growth Plan Status" (or "Satus" for Gold typo)
	GrowthPlanEnd     string // "%s %s Growth Plan (end date)"
	VolumeActuals     string // "%s %s Volume Actuals ($)"
	VolumeActualsAlt  string // "%s %s Volume Actuals (Rolling 36 Months)" (BP only)
	VolumeActualsTotal string
	Threshold         string // "%s %s Threshold ($)"
	VolumePct         string // "%s %s Volume Actuals / Threshold (Percentage)"
	// Alt name for volume pct (no space before "Volume" in some columns)
	VolumePctAlt      string

	// SRI (Gold/Platinum only)
	SRIStatus    string
	SRI          string
	SRIRequired  string
	SRIPct       string

	// Certifications
	CertStatus      string
	SalesCertified  string
	ATPCurrent      string
	ATPActive       string
	ATPAtRisk       string
	ATPNotCurrent   string
	ASECurrent      string
	ASEActive       string
	ASEAtRisk       string
	ASENotCurrent   string
	MASECurrent     string
	MASEActive      string
	MASEAtRisk      string
	MASENotCurrent  string
}

// buildTierColumnName creates the expected column header for a given center prefix and tier.
func buildTierColumnName(prefix, tierName, suffix string) string {
	return prefix + " " + tierName + " " + suffix
}

// tierNames maps Tier to the display name used in column headers.
var tierNames = map[domain.Tier]string{
	domain.TierBusiness: "Business Partner",
	domain.TierSilver:   "Silver",
	domain.TierGold:     "Gold",
	domain.TierPlatinum: "Platinum",
}

// mapPartnerFromRow extracts the Partner identity from a row.
func mapPartnerFromRow(row []string, cm columnMap) domain.Partner {
	pc := partnerColumns
	return domain.Partner{
		PartyID:              cm.get(row, pc.PartyID),
		Name:                 cm.get(row, pc.PartyName),
		Theater:              cm.get(row, pc.Theater),
		SubRegion:            cm.get(row, pc.SubRegion),
		HPEOrg:               cm.get(row, pc.HPEOrg),
		Country:              cm.get(row, pc.Country),
		CountryCode:          cm.get(row, pc.CountryCode),
		HQPartnerIDT1:       cm.get(row, pc.HQPartnerIDT1),
		HQPartnerIDT2:       cm.get(row, pc.HQPartnerIDT2),
		HQPartnerIDSP:       cm.get(row, pc.HQPartnerIDSP),
		CountryEntityID:      cm.get(row, pc.CountryEntityID),
		CountryEntityName:    cm.get(row, pc.CountryEntityName),
		GlobalEntityID:       cm.get(row, pc.GlobalEntityID),
		GlobalEntityName:     cm.get(row, pc.GlobalEntityName),
		BusinessRelSP:        cm.get(row, pc.BusinessRelSP),
		BusinessRelSvc:       cm.get(row, pc.BusinessRelSvc),
		BusinessRelSI:        cm.get(row, pc.BusinessRelSI),
		ActiveAgreement:      parseBool(cm.get(row, pc.ActiveAgreement)),
		LRStatus:             cm.get(row, pc.LRStatus),
		LRStatusStart:        cm.get(row, pc.LRStatusStart),
		LRStatusEnd:          cm.get(row, pc.LRStatusEnd),
		SPAgreement:          parseBool(cm.get(row, pc.SPAgreement)),
		MembershipCompute:    cm.get(row, pc.MembershipCompute),
		MembershipHC:         cm.get(row, pc.MembershipHC),
		MembershipNetworking: cm.get(row, pc.MembershipNet),
		TriplePlatinumPlus:   parseBool(cm.get(row, pc.TriplePlatinum)),
		IntlPartner:          parseBool(cm.get(row, pc.IntlPartner)),
	}
}

// mapTierFromRow extracts a single tier's data from a row.
func mapTierFromRow(row []string, cm columnMap, prefix string, tier domain.Tier) domain.PartnerTier {
	tn := tierNames[tier]

	// Helper to build column name and get value
	col := func(suffix string) string {
		name := buildTierColumnName(prefix, tn, suffix)
		return cm.get(row, name)
	}

	// Business Partner has slightly different column naming
	pt := domain.PartnerTier{
		Center: domain.Center(strings.ToLower(strings.ReplaceAll(prefix, " ", "_"))),
		Tier:   tier,

		CriteriaMet:      parseBool(col("Criteria Met")),
		VolumeStatus:     parseBool(col("Volume Status")),
		GrowthPlanStatus: parseBool(col("Growth Plan Status")),
		GrowthPlanEnd:    col("Growth Plan (end date)"),
		CertStatus:       parseBool(col("Certification Status")),

		VolumeActuals:      parseMoney(col("Volume Actuals ($)")),
		Threshold:          parseMoney(col("Threshold ($)")),
		VolumePct:          parsePct(col("Volume Actuals / Threshold (Percentage)")),

		SalesCertified: col("Sales"),
		ATPCurrent:     col("ATP - CURRENT"),
		ASECurrent:     col("ASE Certified Individuals - CURRENT"),
		MASECurrent:    col("MASE Certified Individuals - CURRENT"),
	}

	// Business Partner tier uses "Volume Actuals (Rolling 36 Months)" instead
	if tier == domain.TierBusiness && pt.VolumeActuals == 0 {
		pt.VolumeActuals = parseMoney(col("Volume Actuals (Rolling 36 Months)"))
	}

	// Volume Actuals Total (includes SP direct orders)
	pt.VolumeActualsTotal = parseMoney(col("Volume Actuals Total ($)"))

	// SRI (Gold/Platinum)
	if tier == domain.TierGold || tier == domain.TierPlatinum {
		pt.SRIStatus = parseBool(col("SRI Status"))
		pt.SRI = parseFloat(col("SRI"))
		pt.SRIRequired = parseFloat(col("SRI Required"))
		pt.SRIPct = parsePct(col("SRI Actuals / SRI Required (Percentage)"))
	}

	// Cert detail counts
	pt.ATPActive = parseInt(col("ATP - CURRENT (active)"))
	pt.ATPAtRisk = parseInt(col("ATP - AT RISK (transitional)"))
	pt.ATPNotCurrent = parseInt(col("ATP – Not Current"))
	pt.ASEActive = parseInt(col("ASE Certified Individuals - CURRENT (active)"))
	pt.ASEAtRisk = parseInt(col("ASE Certified Individuals - AT RISK (transitional)"))
	pt.ASENotCurrent = parseInt(col("ASE Certified Individuals - NOT CURRENT"))
	pt.MASEActive = parseInt(col("MASE Certified Individuals - CURRENT (active)"))
	pt.MASEAtRisk = parseInt(col("MASE Certified Individuals - AT RISK (transitional)"))
	pt.MASENotCurrent = parseInt(col("MASE Certified Individuals - NOT CURRENT"))

	// Dates
	pt.TierStartDate = col("Final (start date)")
	pt.TierEndDate = col("Final (end date)")
	if pt.TierEndDate == "" {
		pt.TierEndDate = col("final (end date)") // Platinum uses lowercase
	}

	// Handle Gold typo: "Growth Plan Satus" instead of "Status"
	if tier == domain.TierGold && !pt.GrowthPlanStatus {
		pt.GrowthPlanStatus = parseBool(col("Growth Plan Satus"))
	}

	return pt
}

// mapCompetenciesFromRow extracts all 14 competency flags from a row.
func mapCompetenciesFromRow(row []string, cm columnMap) []domain.PartnerCompetency {
	competencyColumns := map[string]string{
		"HPE HPC for Enterprise":                                "HPE HPC for Enterprise Competency Criteria Met",
		"HPE Solutions for Cloud IT Ops":                        "HPE Solutions for Cloud IT Ops Competency Criteria Met",
		"HPE Data Protection and Disaster Recovery Solutions":   "HPE Data Protection and Disaster Recovery Solutions Competency Criteria Met",
		"HPE Private Cloud Solutions for Business":              "HPE Private Cloud Solutions for Business Competency Criteria Met",
		"HPE GreenLake":                                         "HPE GreenLake Competency Criteria Met",
		"HPE Solutions for AI":                                  "HPE Solutions for AI Competency Criteria Met",
		"HPE Solutions for Sovereign Cloud":                     "HPE Solutions for Sovereign Cloud Competency Criteria Met",
		"HPE Solutions for Sustainability":                      "HPE Solutions for Sustainability Competency Criteria Met",
		"HPE Aruba Networking Central":                          "HPE Networking Central Criteria Met",
		"HPE Aruba Networking SD-WAN":                           "HPE Networking SD-WAN Criteria Met",
		"HPE Aruba Networking Secure Service Edge":              "HPE Networking Secure Service Edge Criteria Met",
		"HPE Aruba Networking Data Center":                      "HPE Networking Data Center Competency Criteria Met",
		"HPE Aruba Networking ClearPass":                        "HPE Networking ClearPass Competency Criteria Met",
		"HPE Aruba Networking Private 5G":                       "HPE Networking Private 5G Competency Criteria Met",
	}

	// Some competency columns include the FY26 transition text
	compWithTransition := map[string]string{
		"HPE HPC for Enterprise":                              "HPE HPC for Enterprise Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Data Protection and Disaster Recovery Solutions":  "HPE Data Protection and Disaster Recovery Solutions Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Private Cloud Solutions for Business":             "HPE Private Cloud Solutions for Business Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Solutions for Cloud IT Ops":                       "HPE Solutions for Cloud IT Ops Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Solutions for Sustainability":                     "HPE Solutions for Sustainability Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Solutions for AI":                                 "HPE Solutions for AI Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE Solutions for Sovereign Cloud":                    "HPE Solutions for Sovereign Cloud Competency Criteria Met (incl. competencies from FY26 transition)",
		"HPE GreenLake":                                        "HPE GreenLake Competency Criteria Met (incl. competencies from FY26 transition)",
	}

	var comps []domain.PartnerCompetency
	for name, colName := range competencyColumns {
		val := cm.get(row, colName)
		if val == "" {
			// Try with transition suffix
			if alt, ok := compWithTransition[name]; ok {
				val = cm.get(row, alt)
			}
		}
		comps = append(comps, domain.PartnerCompetency{
			Competency:  name,
			CriteriaMet: parseBool(val),
		})
	}
	return comps
}

// mapCompLevelsFromRow extracts quarterly comp levels for a center.
func mapCompLevelsFromRow(row []string, cm columnMap, prefix string) []domain.PartnerCompLevel {
	quarters := []struct {
		q    string
		comp string
		risk string
	}{
		{"Q126", prefix + " Comp Q126", prefix + " Comp@Risk Q126"},
		{"Q226", prefix + " Comp Q226", prefix + " Comp@Risk Q226"},
		{"Q326", prefix + " Comp Q326", prefix + " Comp@Risk  Q326"}, // note double space in original
		{"Q426", prefix + " Comp Q426", prefix + " Comp@Risk  Q426"},
	}

	center := domain.Center(strings.ToLower(strings.ReplaceAll(prefix, " ", "_")))
	var levels []domain.PartnerCompLevel
	for _, q := range quarters {
		compLevel := cm.get(row, q.comp)
		atRisk := cm.get(row, q.risk)
		if compLevel != "" || atRisk != "" {
			levels = append(levels, domain.PartnerCompLevel{
				Center:    center,
				Quarter:   q.q,
				CompLevel: compLevel,
				AtRisk:    atRisk,
			})
		}
	}
	return levels
}
