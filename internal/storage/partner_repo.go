package storage

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/jackc/pgx/v5"
)

// PartnerRepo provides access to partner data in PostgreSQL.
type PartnerRepo struct {
	db *Postgres
}

// NewPartnerRepo creates a new PartnerRepo.
func NewPartnerRepo(db *Postgres) *PartnerRepo {
	return &PartnerRepo{db: db}
}

// Search performs a fuzzy search on partner names using trigram similarity.
// regionFilter: "" for all CCA, "RMC" for RMC countries only.
func (r *PartnerRepo) Search(ctx context.Context, query string, regionFilter string) ([]domain.Partner, error) {
	sql := `
		SELECT id, party_id, name, COALESCE(theater, ''), COALESCE(sub_region, ''),
		       COALESCE(hpe_org, ''), COALESCE(country, ''), COALESCE(country_code, ''),
		       COALESCE(membership_compute, ''), COALESCE(membership_hc, ''),
		       COALESCE(membership_networking, '')
		FROM partners
		WHERE (name ILIKE '%' || $1 || '%'
		    OR party_id ILIKE '%' || $1 || '%'
		    OR COALESCE(global_entity_name, '') ILIKE '%' || $1 || '%')
	`

	args := []any{query}
	argIdx := 2

	if regionFilter == "RMC" {
		sql += fmt.Sprintf(` AND country = ANY($%d)`, argIdx)
		args = append(args, domain.RMCCountries)
		argIdx++
	}

	sql += ` ORDER BY similarity(name, $1) DESC LIMIT 10`

	rows, err := r.db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("search partners: %w", err)
	}
	defer rows.Close()

	var partners []domain.Partner
	for rows.Next() {
		var p domain.Partner
		if err := rows.Scan(
			&p.ID, &p.PartyID, &p.Name, &p.Theater, &p.SubRegion,
			&p.HPEOrg, &p.Country, &p.CountryCode,
			&p.MembershipCompute, &p.MembershipHC, &p.MembershipNetworking,
		); err != nil {
			return nil, fmt.Errorf("scan partner: %w", err)
		}
		partners = append(partners, p)
	}
	return partners, rows.Err()
}

// GetByID retrieves a partner with all related data.
func (r *PartnerRepo) GetByID(ctx context.Context, id int) (*domain.Partner, error) {
	p := &domain.Partner{}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT id, party_id, name, COALESCE(theater, ''), COALESCE(sub_region, ''),
		       COALESCE(hpe_org, ''), COALESCE(country, ''), COALESCE(country_code, ''),
		       COALESCE(hq_partner_id_t1, ''), COALESCE(hq_partner_id_t2, ''),
		       COALESCE(hq_partner_id_sp, ''),
		       COALESCE(country_entity_id, ''), COALESCE(global_entity_id, ''),
		       COALESCE(global_entity_name, ''),
		       COALESCE(business_rel_sp, ''), COALESCE(business_rel_svc, ''),
		       COALESCE(business_rel_si, ''),
		       COALESCE(active_agreement, false), COALESCE(lr_status, ''),
		       COALESCE(membership_compute, ''), COALESCE(membership_hc, ''),
		       COALESCE(membership_networking, ''),
		       COALESCE(triple_platinum_plus, false), COALESCE(intl_partner, false),
		       imported_at
		FROM partners WHERE id = $1
	`, id).Scan(
		&p.ID, &p.PartyID, &p.Name, &p.Theater, &p.SubRegion,
		&p.HPEOrg, &p.Country, &p.CountryCode,
		&p.HQPartnerIDT1, &p.HQPartnerIDT2, &p.HQPartnerIDSP,
		&p.CountryEntityID, &p.GlobalEntityID, &p.GlobalEntityName,
		&p.BusinessRelSP, &p.BusinessRelSvc, &p.BusinessRelSI,
		&p.ActiveAgreement, &p.LRStatus,
		&p.MembershipCompute, &p.MembershipHC, &p.MembershipNetworking,
		&p.TriplePlatinumPlus, &p.IntlPartner,
		&p.ImportedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get partner by id %d: %w", id, err)
	}

	// Load tiers
	tierRows, err := r.db.Pool.Query(ctx, `
		SELECT id, center, tier, criteria_met, volume_status, growth_plan_status,
		       cert_status, COALESCE(volume_actuals, 0), COALESCE(volume_actuals_total, 0),
		       COALESCE(threshold, 0), COALESCE(volume_pct, 0),
		       COALESCE(sri, 0), COALESCE(sri_required, 0), COALESCE(sri_pct, 0),
		       COALESCE(sri_status, false),
		       COALESCE(sales_certified, ''), COALESCE(atp_current, ''),
		       COALESCE(ase_current, ''), COALESCE(mase_current, '')
		FROM partner_tiers WHERE partner_id = $1
		ORDER BY center, CASE tier
			WHEN 'business' THEN 0
			WHEN 'silver' THEN 1
			WHEN 'gold' THEN 2
			WHEN 'platinum' THEN 3
		END
	`, id)
	if err != nil {
		return nil, fmt.Errorf("load tiers: %w", err)
	}
	defer tierRows.Close()

	for tierRows.Next() {
		var t domain.PartnerTier
		if err := tierRows.Scan(
			&t.ID, &t.Center, &t.Tier, &t.CriteriaMet, &t.VolumeStatus,
			&t.GrowthPlanStatus, &t.CertStatus,
			&t.VolumeActuals, &t.VolumeActualsTotal, &t.Threshold, &t.VolumePct,
			&t.SRI, &t.SRIRequired, &t.SRIPct, &t.SRIStatus,
			&t.SalesCertified, &t.ATPCurrent, &t.ASECurrent, &t.MASECurrent,
		); err != nil {
			return nil, fmt.Errorf("scan tier: %w", err)
		}
		t.PartnerID = id
		p.Tiers = append(p.Tiers, t)
	}

	// Load competencies
	compRows, err := r.db.Pool.Query(ctx, `
		SELECT competency, criteria_met
		FROM partner_competencies WHERE partner_id = $1
		ORDER BY competency
	`, id)
	if err != nil {
		return nil, fmt.Errorf("load competencies: %w", err)
	}
	defer compRows.Close()

	for compRows.Next() {
		var c domain.PartnerCompetency
		if err := compRows.Scan(&c.Competency, &c.CriteriaMet); err != nil {
			return nil, fmt.Errorf("scan competency: %w", err)
		}
		c.PartnerID = id
		p.Competencies = append(p.Competencies, c)
	}

	return p, nil
}

// UpsertFromParsed inserts or updates partners from parsed Excel data.
// Returns (inserted, updated, error).
func (r *PartnerRepo) UpsertFromParsed(ctx context.Context, partners map[string]*domain.Partner) (int, int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inserted, updated := 0, 0

	for _, p := range partners {
		// Upsert partner
		var partnerID int
		err := tx.QueryRow(ctx, `
			INSERT INTO partners (
				party_id, name, theater, sub_region, hpe_org, country, country_code,
				hq_partner_id_t1, hq_partner_id_t2, hq_partner_id_sp,
				country_entity_id, country_entity_name, global_entity_id, global_entity_name,
				business_rel_sp, business_rel_svc, business_rel_si,
				active_agreement, lr_status, sp_agreement,
				membership_compute, membership_hc, membership_networking,
				triple_platinum_plus, intl_partner, imported_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
				$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
				$21, $22, $23, $24, $25, NOW()
			)
			ON CONFLICT (party_id) DO UPDATE SET
				name = EXCLUDED.name,
				theater = EXCLUDED.theater,
				sub_region = EXCLUDED.sub_region,
				hpe_org = EXCLUDED.hpe_org,
				country = EXCLUDED.country,
				country_code = EXCLUDED.country_code,
				hq_partner_id_t1 = EXCLUDED.hq_partner_id_t1,
				hq_partner_id_t2 = EXCLUDED.hq_partner_id_t2,
				hq_partner_id_sp = EXCLUDED.hq_partner_id_sp,
				country_entity_id = EXCLUDED.country_entity_id,
				country_entity_name = EXCLUDED.country_entity_name,
				global_entity_id = EXCLUDED.global_entity_id,
				global_entity_name = EXCLUDED.global_entity_name,
				business_rel_sp = EXCLUDED.business_rel_sp,
				business_rel_svc = EXCLUDED.business_rel_svc,
				business_rel_si = EXCLUDED.business_rel_si,
				active_agreement = EXCLUDED.active_agreement,
				lr_status = EXCLUDED.lr_status,
				sp_agreement = EXCLUDED.sp_agreement,
				membership_compute = EXCLUDED.membership_compute,
				membership_hc = EXCLUDED.membership_hc,
				membership_networking = EXCLUDED.membership_networking,
				triple_platinum_plus = EXCLUDED.triple_platinum_plus,
				intl_partner = EXCLUDED.intl_partner,
				imported_at = NOW()
			RETURNING id
		`, p.PartyID, p.Name, p.Theater, p.SubRegion, p.HPEOrg, p.Country, p.CountryCode,
			p.HQPartnerIDT1, p.HQPartnerIDT2, p.HQPartnerIDSP,
			p.CountryEntityID, p.CountryEntityName, p.GlobalEntityID, p.GlobalEntityName,
			p.BusinessRelSP, p.BusinessRelSvc, p.BusinessRelSI,
			p.ActiveAgreement, p.LRStatus, p.SPAgreement,
			p.MembershipCompute, p.MembershipHC, p.MembershipNetworking,
			p.TriplePlatinumPlus, p.IntlPartner,
		).Scan(&partnerID)
		if err != nil {
			slog.Warn("upsert partner failed", "party_id", p.PartyID, "name", p.Name, "error", err)
			continue
		}

		// Upsert tiers
		for _, t := range p.Tiers {
			_, err := tx.Exec(ctx, `
				INSERT INTO partner_tiers (
					partner_id, center, tier,
					criteria_met, volume_status, growth_plan_status, cert_status,
					volume_actuals, volume_actuals_total, threshold, volume_pct,
					sri, sri_required, sri_pct, sri_status,
					sales_certified, atp_current, atp_active, atp_at_risk, atp_not_current,
					ase_current, ase_active, ase_at_risk, ase_not_current,
					mase_current, mase_active, mase_at_risk, mase_not_current,
					tier_start_date, tier_end_date
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
					$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
					$21, $22, $23, $24, $25, $26, $27, $28,
					NULLIF($29, '')::date, NULLIF($30, '')::date
				)
				ON CONFLICT (partner_id, center, tier) DO UPDATE SET
					criteria_met = EXCLUDED.criteria_met,
					volume_status = EXCLUDED.volume_status,
					growth_plan_status = EXCLUDED.growth_plan_status,
					cert_status = EXCLUDED.cert_status,
					volume_actuals = EXCLUDED.volume_actuals,
					volume_actuals_total = EXCLUDED.volume_actuals_total,
					threshold = EXCLUDED.threshold,
					volume_pct = EXCLUDED.volume_pct,
					sri = EXCLUDED.sri,
					sri_required = EXCLUDED.sri_required,
					sri_pct = EXCLUDED.sri_pct,
					sri_status = EXCLUDED.sri_status,
					sales_certified = EXCLUDED.sales_certified,
					atp_current = EXCLUDED.atp_current,
					atp_active = EXCLUDED.atp_active,
					atp_at_risk = EXCLUDED.atp_at_risk,
					atp_not_current = EXCLUDED.atp_not_current,
					ase_current = EXCLUDED.ase_current,
					ase_active = EXCLUDED.ase_active,
					ase_at_risk = EXCLUDED.ase_at_risk,
					ase_not_current = EXCLUDED.ase_not_current,
					mase_current = EXCLUDED.mase_current,
					mase_active = EXCLUDED.mase_active,
					mase_at_risk = EXCLUDED.mase_at_risk,
					mase_not_current = EXCLUDED.mase_not_current,
					tier_start_date = EXCLUDED.tier_start_date,
					tier_end_date = EXCLUDED.tier_end_date
			`, partnerID, string(t.Center), string(t.Tier),
				t.CriteriaMet, t.VolumeStatus, t.GrowthPlanStatus, t.CertStatus,
				t.VolumeActuals, t.VolumeActualsTotal, t.Threshold, t.VolumePct,
				t.SRI, t.SRIRequired, t.SRIPct, t.SRIStatus,
				t.SalesCertified, t.ATPCurrent, t.ATPActive, t.ATPAtRisk, t.ATPNotCurrent,
				t.ASECurrent, t.ASEActive, t.ASEAtRisk, t.ASENotCurrent,
				t.MASECurrent, t.MASEActive, t.MASEAtRisk, t.MASENotCurrent,
				t.TierStartDate, t.TierEndDate,
			)
			if err != nil {
				slog.Warn("upsert tier failed", "partner", p.Name, "center", t.Center, "tier", t.Tier, "error", err)
			}
		}

		// Upsert competencies
		for _, c := range p.Competencies {
			_, err := tx.Exec(ctx, `
				INSERT INTO partner_competencies (partner_id, competency, criteria_met)
				VALUES ($1, $2, $3)
				ON CONFLICT (partner_id, competency) DO UPDATE SET
					criteria_met = EXCLUDED.criteria_met
			`, partnerID, c.Competency, c.CriteriaMet)
			if err != nil {
				slog.Warn("upsert competency failed", "partner", p.Name, "competency", c.Competency, "error", err)
			}
		}

		// Upsert comp levels
		for _, cl := range p.CompLevels {
			_, err := tx.Exec(ctx, `
				INSERT INTO partner_comp_levels (partner_id, center, quarter, comp_level, at_risk)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (partner_id, center, quarter) DO UPDATE SET
					comp_level = EXCLUDED.comp_level,
					at_risk = EXCLUDED.at_risk
			`, partnerID, string(cl.Center), cl.Quarter, cl.CompLevel, cl.AtRisk)
			if err != nil {
				slog.Warn("upsert comp level failed", "partner", p.Name, "error", err)
			}
		}

		inserted++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit tx: %w", err)
	}

	return inserted, updated, nil
}

// CountAll returns total number of partners.
func (r *PartnerRepo) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM partners").Scan(&count)
	return count, err
}

// TierDistribution returns counts by center and membership level.
func (r *PartnerRepo) TierDistribution(ctx context.Context, regionFilter string) (map[string]map[string]int, error) {
	sql := `SELECT
		CASE
			WHEN membership_compute != '' THEN 'compute'
			ELSE ''
		END as center,
		membership_compute as membership
		FROM partners WHERE membership_compute != ''`

	if regionFilter == "RMC" {
		sql += ` AND country = ANY($1)`
	}

	// This is simplified — for a proper implementation, we'd query each center's membership
	// For now, return a placeholder structure
	result := make(map[string]map[string]int)
	for _, center := range []string{"compute", "hybrid_cloud", "networking"} {
		result[center] = make(map[string]int)

		membershipCol := "membership_compute"
		switch center {
		case "hybrid_cloud":
			membershipCol = "membership_hc"
		case "networking":
			membershipCol = "membership_networking"
		}

		query := fmt.Sprintf(`
			SELECT COALESCE(%s, 'None'), COUNT(*)
			FROM partners
			WHERE %s IS NOT NULL AND %s != ''
		`, membershipCol, membershipCol, membershipCol)

		var args []any
		if regionFilter == "RMC" {
			query += ` AND country = ANY($1)`
			args = append(args, domain.RMCCountries)
		}
		query += fmt.Sprintf(` GROUP BY %s ORDER BY COUNT(*) DESC`, membershipCol)

		rows, err := r.db.Pool.Query(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("tier distribution for %s: %w", center, err)
		}

		for rows.Next() {
			var membership string
			var count int
			if err := rows.Scan(&membership, &count); err != nil {
				rows.Close()
				return nil, err
			}
			// Normalize membership name to just the tier
			membership = strings.ToLower(membership)
			for _, tier := range []string{"platinum", "gold", "silver", "business"} {
				if strings.Contains(membership, tier) {
					result[center][tier] += count
					break
				}
			}
		}
		rows.Close()
	}

	return result, nil
}

// LogImport records an import operation with optional data date.
func (r *PartnerRepo) LogImport(ctx context.Context, filename string, sheets []string, totalPartners, ccaPartners int, durationMs int) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO import_log (filename, data_date, sheets_parsed, partners_total, partners_cca, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, filename, ExtractDateFromFilename(filename), sheets, totalPartners, ccaPartners, durationMs)
	return err
}

// GetLastImportDate returns the data_date and imported_at from the most recent import.
func (r *PartnerRepo) GetLastImportDate(ctx context.Context) (dataDate string, importedAt string, err error) {
	err = r.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(data_date::text, ''),
		       TO_CHAR(imported_at, 'YYYY-MM-DD HH24:MI')
		FROM import_log ORDER BY id DESC LIMIT 1
	`).Scan(&dataDate, &importedAt)
	return
}

// ExtractDateFromFilename extracts a date pattern (YYYY-MM-DD or YYYY_MM_DD) from a filename.
func ExtractDateFromFilename(filename string) *string {
	// Look for YYYY-MM-DD or YYYY_MM_DD patterns
	re := regexp.MustCompile(`(\d{4})[-_](\d{2})[-_](\d{2})`)
	match := re.FindStringSubmatch(filename)
	if len(match) >= 4 {
		date := match[1] + "-" + match[2] + "-" + match[3]
		return &date
	}
	return nil
}

// CountryStats returns partner count by country, sorted descending.
func (r *PartnerRepo) CountryStats(ctx context.Context) ([]CountryStat, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT COALESCE(country, 'Unknown'), COUNT(*)
		FROM partners
		GROUP BY country
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CountryStat
	for rows.Next() {
		var s CountryStat
		if err := rows.Scan(&s.Country, &s.Count); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

// CountryStat holds country name and partner count.
type CountryStat struct {
	Country string
	Count   int
}

// CountryTierMatrix returns a matrix of country × tier membership counts (uses compute center).
func (r *PartnerRepo) CountryTierMatrix(ctx context.Context) ([]CountryTierRow, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT country,
			COUNT(*) FILTER (WHERE LOWER(membership_compute) LIKE '%platinum%') as plat,
			COUNT(*) FILTER (WHERE LOWER(membership_compute) LIKE '%gold%') as gold,
			COUNT(*) FILTER (WHERE LOWER(membership_compute) LIKE '%silver%') as silver,
			COUNT(*) FILTER (WHERE LOWER(membership_compute) LIKE '%business%') as biz,
			COUNT(*) as total
		FROM partners
		WHERE membership_compute IS NOT NULL AND membership_compute != ''
		GROUP BY country
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CountryTierRow
	for rows.Next() {
		var r CountryTierRow
		if err := rows.Scan(&r.Country, &r.Plat, &r.Gold, &r.Silver, &r.Biz, &r.Total); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, nil
}

// CountryTierRow holds one row of country × tier data.
type CountryTierRow struct {
	Country                        string
	Plat, Gold, Silver, Biz, Total int
}

// UpgradeReadyCount returns how many partners have criteria_met for each tier in each center.
func (r *PartnerRepo) UpgradeReadyCount(ctx context.Context) (map[string]map[string]int, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT center, tier, COUNT(DISTINCT partner_id)
		FROM partner_tiers
		WHERE criteria_met = true AND tier != 'business'
		GROUP BY center, tier
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var center, tier string
		var count int
		if err := rows.Scan(&center, &tier, &count); err != nil {
			return nil, err
		}
		if result[center] == nil {
			result[center] = make(map[string]int)
		}
		result[center][tier] = count
	}
	return result, nil
}

// TopVolumePartners returns the top N partners by volume in any center.
func (r *PartnerRepo) TopVolumePartners(ctx context.Context, limit int) ([]TopPartner, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT p.name, pt.center, pt.volume_actuals
		FROM partner_tiers pt
		JOIN partners p ON p.id = pt.partner_id
		WHERE pt.tier = 'platinum' AND pt.volume_actuals > 0
		ORDER BY pt.volume_actuals DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TopPartner
	for rows.Next() {
		var t TopPartner
		if err := rows.Scan(&t.Name, &t.Center, &t.Volume); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

// TopPartner holds a partner name and volume.
type TopPartner struct {
	Name   string
	Center string
	Volume float64
}

// GapSummary holds aggregate gap data for a center.
type GapSummary struct {
	Center       string
	VolumeGap    float64
	VolumeCount  int
	CertGapCount int
}

// GapSummaryAll returns aggregate volume and cert gaps per center.
func (r *PartnerRepo) GapSummaryAll(ctx context.Context) ([]GapSummary, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT center,
			COALESCE(SUM(CASE WHEN volume_status = false AND threshold < 9999999 THEN threshold - volume_actuals ELSE 0 END), 0),
			COUNT(*) FILTER (WHERE volume_status = false AND threshold < 9999999),
			COUNT(*) FILTER (WHERE cert_status = false)
		FROM partner_tiers
		WHERE tier = (
			CASE
				WHEN center = 'compute' THEN 'gold'
				WHEN center = 'hybrid_cloud' THEN 'gold'
				WHEN center = 'networking' THEN 'silver'
			END
		)
		GROUP BY center
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []GapSummary
	for rows.Next() {
		var g GapSummary
		if err := rows.Scan(&g.Center, &g.VolumeGap, &g.VolumeCount, &g.CertGapCount); err != nil {
			return nil, err
		}
		result = append(result, g)
	}
	return result, nil
}


// unused but satisfies interface
var _ = pgx.Rows(nil)
