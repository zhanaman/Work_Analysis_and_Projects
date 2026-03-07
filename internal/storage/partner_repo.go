package storage

import (
	"context"
	"fmt"

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
// Returns up to 10 most relevant matches.
func (r *PartnerRepo) Search(ctx context.Context, query string) ([]domain.Partner, error) {
	sql := `
		SELECT id, name, COALESCE(partner_id_hpe, ''), COALESCE(tier, ''),
		       COALESCE(country, ''), COALESCE(city, ''),
		       COALESCE(compute_cert, ''), COALESCE(networking_cert, ''),
		       COALESCE(hybrid_cloud_cert, ''), COALESCE(storage_cert, ''),
		       COALESCE(revenue_ytd, 0), COALESCE(target, 0),
		       COALESCE(contact_name, ''), COALESCE(contact_email, ''),
		       COALESCE(contact_phone, '')
		FROM partners
		WHERE name ILIKE '%' || $1 || '%'
		   OR partner_id_hpe ILIKE '%' || $1 || '%'
		ORDER BY similarity(name, $1) DESC
		LIMIT 10
	`

	rows, err := r.db.Pool.Query(ctx, sql, query)
	if err != nil {
		return nil, fmt.Errorf("search partners: %w", err)
	}
	defer rows.Close()

	return scanPartners(rows)
}

// GetByID retrieves a single partner by internal ID.
func (r *PartnerRepo) GetByID(ctx context.Context, id int) (*domain.Partner, error) {
	sql := `
		SELECT id, name, COALESCE(partner_id_hpe, ''), COALESCE(tier, ''),
		       COALESCE(country, ''), COALESCE(city, ''),
		       COALESCE(compute_cert, ''), COALESCE(networking_cert, ''),
		       COALESCE(hybrid_cloud_cert, ''), COALESCE(storage_cert, ''),
		       COALESCE(revenue_ytd, 0), COALESCE(target, 0),
		       COALESCE(contact_name, ''), COALESCE(contact_email, ''),
		       COALESCE(contact_phone, '')
		FROM partners
		WHERE id = $1
	`

	row := r.db.Pool.QueryRow(ctx, sql, id)

	var p domain.Partner
	err := row.Scan(
		&p.ID, &p.Name, &p.PartnerID, &p.Tier,
		&p.Country, &p.City,
		&p.ComputeCert, &p.NetworkingCert, &p.HybridCloud, &p.StorageCert,
		&p.RevenueYTD, &p.Target,
		&p.ContactName, &p.ContactEmail, &p.ContactPhone,
	)
	if err != nil {
		return nil, fmt.Errorf("get partner by id %d: %w", id, err)
	}

	return &p, nil
}

// UpsertBatch inserts or updates a batch of partners.
// Uses partner_id_hpe as the conflict key.
func (r *PartnerRepo) UpsertBatch(ctx context.Context, partners []domain.Partner) (int, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	sql := `
		INSERT INTO partners (
			name, partner_id_hpe, tier, country, city,
			compute_cert, networking_cert, hybrid_cloud_cert, storage_cert,
			revenue_ytd, target,
			contact_name, contact_email, contact_phone,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11,
			$12, $13, $14,
			NOW()
		)
		ON CONFLICT (partner_id_hpe) DO UPDATE SET
			name = EXCLUDED.name,
			tier = EXCLUDED.tier,
			country = EXCLUDED.country,
			city = EXCLUDED.city,
			compute_cert = EXCLUDED.compute_cert,
			networking_cert = EXCLUDED.networking_cert,
			hybrid_cloud_cert = EXCLUDED.hybrid_cloud_cert,
			storage_cert = EXCLUDED.storage_cert,
			revenue_ytd = EXCLUDED.revenue_ytd,
			target = EXCLUDED.target,
			contact_name = EXCLUDED.contact_name,
			contact_email = EXCLUDED.contact_email,
			contact_phone = EXCLUDED.contact_phone,
			updated_at = NOW()
	`

	count := 0
	for _, p := range partners {
		_, err := tx.Exec(ctx, sql,
			p.Name, p.PartnerID, p.Tier, p.Country, p.City,
			p.ComputeCert, p.NetworkingCert, p.HybridCloud, p.StorageCert,
			p.RevenueYTD, p.Target,
			p.ContactName, p.ContactEmail, p.ContactPhone,
		)
		if err != nil {
			return count, fmt.Errorf("upsert partner %q: %w", p.Name, err)
		}
		count++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return count, nil
}

// CountAll returns total number of partners.
func (r *PartnerRepo) CountAll(ctx context.Context) (int, error) {
	var count int
	err := r.db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM partners").Scan(&count)
	return count, err
}

func scanPartners(rows pgx.Rows) ([]domain.Partner, error) {
	var partners []domain.Partner
	for rows.Next() {
		var p domain.Partner
		err := rows.Scan(
			&p.ID, &p.Name, &p.PartnerID, &p.Tier,
			&p.Country, &p.City,
			&p.ComputeCert, &p.NetworkingCert, &p.HybridCloud, &p.StorageCert,
			&p.RevenueYTD, &p.Target,
			&p.ContactName, &p.ContactEmail, &p.ContactPhone,
		)
		if err != nil {
			return nil, fmt.Errorf("scan partner row: %w", err)
		}
		partners = append(partners, p)
	}
	return partners, rows.Err()
}
