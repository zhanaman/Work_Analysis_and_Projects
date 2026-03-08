package domain

import "time"

// Center represents an HPE Partner Ready Vantage Sell Track center.
type Center string

const (
	CenterCompute     Center = "compute"
	CenterHybridCloud Center = "hybrid_cloud"
	CenterNetworking  Center = "networking"
)

// Tier represents a partner tier level within a center.
type Tier string

const (
	TierBusiness Tier = "business"
	TierSilver   Tier = "silver"
	TierGold     Tier = "gold"
	TierPlatinum Tier = "platinum"
)

// TierOrder returns numeric order for tier comparison.
func TierOrder(t Tier) int {
	switch t {
	case TierBusiness:
		return 0
	case TierSilver:
		return 1
	case TierGold:
		return 2
	case TierPlatinum:
		return 3
	default:
		return -1
	}
}

// NextTier returns the tier above the given one, or "" if already at max.
func NextTier(t Tier) Tier {
	switch t {
	case TierBusiness:
		return TierSilver
	case TierSilver:
		return TierGold
	case TierGold:
		return TierPlatinum
	default:
		return ""
	}
}

// Partner represents an HPE channel partner with identity and geography.
type Partner struct {
	ID      int    `json:"id"`
	PartyID string `json:"party_id"` // Signing Entity Party ID (unique key)
	Name    string `json:"name"`     // Party Name

	// Geography
	Theater     string `json:"theater"`
	SubRegion   string `json:"sub_region"`
	HPEOrg      string `json:"hpe_org"`      // HPE Organization
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`

	// Partner IDs
	HQPartnerIDT1 string `json:"hq_partner_id_t1"` // T1 Solution Provider
	HQPartnerIDT2 string `json:"hq_partner_id_t2"` // T2 Solution Provider
	HQPartnerIDSP string `json:"hq_partner_id_sp"` // Service Provider

	// Entity info
	CountryEntityID   string `json:"country_entity_id"`
	CountryEntityName string `json:"country_entity_name"`
	GlobalEntityID    string `json:"global_entity_id"`
	GlobalEntityName  string `json:"global_entity_name"`

	// Business relationship
	BusinessRelSP  string `json:"business_rel_sp"`  // Solution Provider
	BusinessRelSvc string `json:"business_rel_svc"` // Service Provider
	BusinessRelSI  string `json:"business_rel_si"`  // System Integrator

	// Agreement
	ActiveAgreement bool   `json:"active_agreement"`
	LRStatus        string `json:"lr_status"`
	LRStatusStart   string `json:"lr_status_start"`
	LRStatusEnd     string `json:"lr_status_end"`
	SPAgreement     bool   `json:"sp_agreement"`

	// Current memberships (per center)
	MembershipCompute    string `json:"membership_compute"`
	MembershipHC         string `json:"membership_hc"`
	MembershipNetworking string `json:"membership_networking"`

	// Special flags
	TriplePlatinumPlus bool `json:"triple_platinum_plus"`
	IntlPartner        bool `json:"intl_partner"`

	// Related data (loaded separately)
	Tiers        []PartnerTier       `json:"tiers,omitempty"`
	Competencies []PartnerCompetency `json:"competencies,omitempty"`
	CompLevels   []PartnerCompLevel  `json:"comp_levels,omitempty"`
	Revenue      []PartnerRevenue    `json:"revenue,omitempty"`
	SCS          *PartnerSCS         `json:"scs,omitempty"`

	// Metadata
	ImportedAt time.Time `json:"imported_at"`
}

// PartnerTier holds tier-specific data for one center + tier level.
type PartnerTier struct {
	ID        int    `json:"id"`
	PartnerID int    `json:"partner_id"`
	Center    Center `json:"center"`
	Tier      Tier   `json:"tier"`

	// Status flags
	CriteriaMet      bool `json:"criteria_met"`
	VolumeStatus     bool `json:"volume_status"`
	GrowthPlanStatus bool `json:"growth_plan_status"`
	GrowthPlanEnd    string `json:"growth_plan_end,omitempty"`
	CertStatus       bool `json:"cert_status"`

	// Volume
	VolumeActuals      float64 `json:"volume_actuals"`
	VolumeActualsTotal float64 `json:"volume_actuals_total"`
	Threshold          float64 `json:"threshold"`
	VolumePct          float64 `json:"volume_pct"` // actuals / threshold %

	// SRI (Gold+)
	SRI         float64 `json:"sri"`
	SRIRequired float64 `json:"sri_required"`
	SRIPct      float64 `json:"sri_pct"`
	SRIStatus   bool    `json:"sri_status"`

	// Certifications ("x/y" format)
	SalesCertified string `json:"sales_certified"`
	ATPCurrent     string `json:"atp_current"`
	ATPActive      int    `json:"atp_active"`
	ATPAtRisk      int    `json:"atp_at_risk"`
	ATPNotCurrent  int    `json:"atp_not_current"`
	ASECurrent     string `json:"ase_current"`
	ASEActive      int    `json:"ase_active"`
	ASEAtRisk      int    `json:"ase_at_risk"`
	ASENotCurrent  int    `json:"ase_not_current"`
	MASECurrent    string `json:"mase_current"`
	MASEActive     int    `json:"mase_active"`
	MASEAtRisk     int    `json:"mase_at_risk"`
	MASENotCurrent int    `json:"mase_not_current"`

	// Dates
	TierStartDate string `json:"tier_start_date,omitempty"`
	TierEndDate   string `json:"tier_end_date,omitempty"`
}

// PartnerCompetency tracks a single competency achievement.
type PartnerCompetency struct {
	ID          int    `json:"id"`
	PartnerID   int    `json:"partner_id"`
	Competency  string `json:"competency"`
	CriteriaMet bool   `json:"criteria_met"`
}

// PartnerCompLevel stores quarterly compensation level per center.
type PartnerCompLevel struct {
	ID        int    `json:"id"`
	PartnerID int    `json:"partner_id"`
	Center    Center `json:"center"`
	Quarter   string `json:"quarter"` // "Q126", "Q226", etc.
	CompLevel string `json:"comp_level"`
	AtRisk    string `json:"at_risk"`
}

// PartnerRevenue holds revenue breakdown per center.
type PartnerRevenue struct {
	ID            int     `json:"id"`
	PartnerID     int     `json:"partner_id"`
	Center        Center  `json:"center"`
	TotalBusiness float64 `json:"total_business"`
	TotalProducts float64 `json:"total_products"`
	TotalServices float64 `json:"total_services"`
	OpsLOB        float64 `json:"ops_lob"`
	AttachSM      float64 `json:"attach_sm"`
	IBSM          float64 `json:"ib_sm"`
	RefreshDate   string  `json:"refresh_date"`
}

// PartnerSCS holds Service Contract Specialist data.
type PartnerSCS struct {
	ID                int     `json:"id"`
	PartnerID         int     `json:"partner_id"`
	Membership        string  `json:"membership"`
	CriteriaMet       bool    `json:"criteria_met"`
	PRVMembershipMet  bool    `json:"prv_membership_met"`
	ThresholdMet      bool    `json:"threshold_met"`
	VolumeActuals     float64 `json:"volume_actuals"`
	Threshold         float64 `json:"threshold"`
	VolumePct         float64 `json:"volume_pct"`
}

// AllCompetencies lists all 14 HPE competencies tracked in the program.
var AllCompetencies = []string{
	"HPE HPC for Enterprise",
	"HPE Solutions for Cloud IT Ops",
	"HPE Data Protection and Disaster Recovery Solutions",
	"HPE Private Cloud Solutions for Business",
	"HPE GreenLake",
	"HPE Solutions for AI",
	"HPE Solutions for Sovereign Cloud",
	"HPE Solutions for Sustainability",
	"HPE Aruba Networking Central",
	"HPE Aruba Networking SD-WAN",
	"HPE Aruba Networking Secure Service Edge",
	"HPE Aruba Networking Data Center",
	"HPE Aruba Networking ClearPass",
	"HPE Aruba Networking Private 5G",
}
