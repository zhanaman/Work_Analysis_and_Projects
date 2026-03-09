package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
)

// UserRepo provides access to user data in PostgreSQL.
type UserRepo struct {
	db *Postgres
}

// NewUserRepo creates a new UserRepo.
func NewUserRepo(db *Postgres) *UserRepo {
	return &UserRepo{db: db}
}

// scanColumns returns the standard SELECT columns for user queries.
const userSelectCols = `id, telegram_id, username, full_name, role,
	COALESCE(region_filter, ''), partner_id,
	COALESCE(email, ''), COALESCE(email_verified, false),
	COALESCE(lang, 'ru'), COALESCE(bot_type, 'pbm'), created_at`

// scanUser scans a row into a domain.User.
func scanUser(scanner interface{ Scan(dest ...any) error }) (*domain.User, error) {
	u := &domain.User{}
	err := scanner.Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FullName, &u.Role,
		&u.RegionFilter, &u.PartnerID,
		&u.Email, &u.EmailVerified,
		&u.Lang, &u.BotType, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetOrCreate finds a user by Telegram ID, or creates one with "pending" role.
// Returns the user and whether it was newly created.
func (r *UserRepo) GetOrCreate(ctx context.Context, telegramID int64, username, fullName string) (*domain.User, bool, error) {
	// Try to find existing
	u, err := r.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return u, false, nil
	}

	// Create new user
	sql := `
		INSERT INTO users (telegram_id, username, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING ` + userSelectCols
	u, err = scanUser(r.db.Pool.QueryRow(ctx, sql, telegramID, username, fullName, domain.RolePending))
	if err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}

	return u, true, nil
}

// GetOrCreatePartner finds or creates a user for the Partner bot.
func (r *UserRepo) GetOrCreatePartner(ctx context.Context, telegramID int64, username, fullName string) (*domain.User, bool, error) {
	u, err := r.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return u, false, nil
	}

	sql := `
		INSERT INTO users (telegram_id, username, full_name, role, bot_type)
		VALUES ($1, $2, $3, $4, 'partner')
		RETURNING ` + userSelectCols
	u, err = scanUser(r.db.Pool.QueryRow(ctx, sql, telegramID, username, fullName, domain.RolePending))
	if err != nil {
		return nil, false, fmt.Errorf("create partner user: %w", err)
	}

	return u, true, nil
}

// GetByTelegramID finds a user by their Telegram ID.
func (r *UserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	sql := `SELECT ` + userSelectCols + ` FROM users WHERE telegram_id = $1`
	u, err := scanUser(r.db.Pool.QueryRow(ctx, sql, telegramID))
	if err != nil {
		return nil, fmt.Errorf("get user by telegram_id %d: %w", telegramID, err)
	}
	return u, nil
}

// GetByID finds a user by their internal DB ID.
func (r *UserRepo) GetByID(ctx context.Context, id int) (*domain.User, error) {
	sql := `SELECT ` + userSelectCols + ` FROM users WHERE id = $1`
	u, err := scanUser(r.db.Pool.QueryRow(ctx, sql, id))
	if err != nil {
		return nil, fmt.Errorf("get user by id %d: %w", id, err)
	}
	return u, nil
}

// SetRole updates a user's role.
func (r *UserRepo) SetRole(ctx context.Context, telegramID int64, role domain.Role) error {
	sql := `UPDATE users SET role = $1 WHERE telegram_id = $2`
	tag, err := r.db.Pool.Exec(ctx, sql, role, telegramID)
	if err != nil {
		return fmt.Errorf("set role for %d: %w", telegramID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user with telegram_id %d not found", telegramID)
	}
	return nil
}

// SetPartnerLink links a user to a partner and sets their email.
func (r *UserRepo) SetPartnerLink(ctx context.Context, userID int, partnerID int, email string) error {
	sql := `UPDATE users SET partner_id = $1, email = $2 WHERE id = $3`
	_, err := r.db.Pool.Exec(ctx, sql, partnerID, email, userID)
	if err != nil {
		return fmt.Errorf("set partner link for user %d: %w", userID, err)
	}
	return nil
}

// ApprovePartner sets a partner user to approved status.
func (r *UserRepo) ApprovePartner(ctx context.Context, userID int) error {
	sql := `UPDATE users SET role = $1, email_verified = true WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, domain.RoleUser, userID)
	if err != nil {
		return fmt.Errorf("approve partner user %d: %w", userID, err)
	}
	return nil
}

// SetLang updates a user's language preference.
func (r *UserRepo) SetLang(ctx context.Context, telegramID int64, lang string) error {
	sql := `UPDATE users SET lang = $1 WHERE telegram_id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, lang, telegramID)
	return err
}

// ListPending returns all users awaiting admin approval.
func (r *UserRepo) ListPending(ctx context.Context) ([]domain.User, error) {
	sql := `SELECT ` + userSelectCols + `
		FROM users
		WHERE role = $1
		ORDER BY created_at ASC`
	rows, err := r.db.Pool.Query(ctx, sql, domain.RolePending)
	if err != nil {
		return nil, fmt.Errorf("list pending users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// ListPendingPartners returns partner bot users awaiting approval.
func (r *UserRepo) ListPendingPartners(ctx context.Context) ([]domain.User, error) {
	sql := `SELECT ` + userSelectCols + `
		FROM users
		WHERE role = $1 AND bot_type = 'partner' AND email IS NOT NULL
		ORDER BY created_at ASC`
	rows, err := r.db.Pool.Query(ctx, sql, domain.RolePending)
	if err != nil {
		return nil, fmt.Errorf("list pending partners: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// MatchPartnerByEmail tries to find partners whose name matches the email domain.
// For example, email "ivan@kazproftech.kz" → search "kazproftech" in partners.name.
func (r *UserRepo) MatchPartnerByEmail(ctx context.Context, email string) ([]struct {
	ID   int
	Name string
}, error) {
	// Extract domain part before TLD
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid email format")
	}
	domainParts := strings.Split(parts[1], ".")
	if len(domainParts) == 0 {
		return nil, fmt.Errorf("invalid email domain")
	}
	// Use the first part of domain as search term (e.g., "kazproftech" from "kazproftech.kz")
	searchTerm := domainParts[0]

	sql := `SELECT id, name FROM partners WHERE LOWER(name) LIKE '%' || LOWER($1) || '%' LIMIT 10`
	rows, err := r.db.Pool.Query(ctx, sql, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("match partner by email: %w", err)
	}
	defer rows.Close()

	var results []struct {
		ID   int
		Name string
	}
	for rows.Next() {
		var p struct {
			ID   int
			Name string
		}
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		results = append(results, p)
	}
	return results, rows.Err()
}
