package storage

import (
	"context"
	"fmt"

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
	COALESCE(lang, 'ru'), COALESCE(bot_type, 'pbm'),
	COALESCE(onboard_step, ''), COALESCE(company_name, ''), onboard_msg_id,
	created_at`

// scanUser scans a row into a domain.User.
func scanUser(scanner interface{ Scan(dest ...any) error }) (*domain.User, error) {
	u := &domain.User{}
	err := scanner.Scan(
		&u.ID, &u.TelegramID, &u.Username, &u.FullName, &u.Role,
		&u.RegionFilter, &u.PartnerID,
		&u.Email, &u.EmailVerified,
		&u.Lang, &u.BotType,
		&u.OnboardStep, &u.CompanyName, &u.OnboardMsgID,
		&u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetOrCreate finds a PBM user by (telegram_id, bot_type='pbm'), or creates one with "pending" role.
// Returns the user and whether it was newly created.
func (r *UserRepo) GetOrCreate(ctx context.Context, telegramID int64, username, fullName string) (*domain.User, bool, error) {
	// Try to find existing PBM user
	u, err := r.GetByTelegramIDAndBotType(ctx, telegramID, "pbm")
	if err == nil {
		return u, false, nil
	}

	// Create new PBM user
	sql := `
		INSERT INTO users (telegram_id, username, full_name, role, bot_type)
		VALUES ($1, $2, $3, $4, 'pbm')
		RETURNING ` + userSelectCols
	u, err = scanUser(r.db.Pool.QueryRow(ctx, sql, telegramID, username, fullName, domain.RolePending))
	if err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}

	return u, true, nil
}

// GetOrCreatePartner finds or creates a user for the Partner bot.
// Searches by (telegram_id, bot_type='partner') so PBM and Partner bots have separate records.
func (r *UserRepo) GetOrCreatePartner(ctx context.Context, telegramID int64, username, fullName string) (*domain.User, bool, error) {
	u, err := r.GetByTelegramIDAndBotType(ctx, telegramID, "partner")
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

// GetByTelegramIDAndBotType finds a user by Telegram ID and bot type.
func (r *UserRepo) GetByTelegramIDAndBotType(ctx context.Context, telegramID int64, botType string) (*domain.User, error) {
	sql := `SELECT ` + userSelectCols + ` FROM users WHERE telegram_id = $1 AND bot_type = $2`
	u, err := scanUser(r.db.Pool.QueryRow(ctx, sql, telegramID, botType))
	if err != nil {
		return nil, fmt.Errorf("get user by telegram_id %d bot_type %s: %w", telegramID, botType, err)
	}
	return u, nil
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

// SetRole updates a user's role by DB ID (safe: never touches other bot's record).
func (r *UserRepo) SetRole(ctx context.Context, userID int, role domain.Role) error {
	sql := `UPDATE users SET role = $1 WHERE id = $2`
	tag, err := r.db.Pool.Exec(ctx, sql, role, userID)
	if err != nil {
		return fmt.Errorf("set role for user %d: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user with id %d not found", userID)
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

// ApproveDistri sets a partner user to distributor status.
func (r *UserRepo) ApproveDistri(ctx context.Context, userID int) error {
	sql := `UPDATE users SET role = $1, email_verified = true WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, domain.RoleDistri, userID)
	if err != nil {
		return fmt.Errorf("approve distri user %d: %w", userID, err)
	}
	return nil
}

// SetLang updates a user's language preference by DB ID (safe: never touches other bot's record).
func (r *UserRepo) SetLang(ctx context.Context, userID int, lang string) error {
	sql := `UPDATE users SET lang = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, lang, userID)
	return err
}

// SetRegionFilter updates a PBM user's region filter (e.g. "RMC", "Azerbaijan", or "" for all).
func (r *UserRepo) SetRegionFilter(ctx context.Context, userID int, regionFilter string) error {
	sql := `UPDATE users SET region_filter = $1 WHERE id = $2`
	tag, err := r.db.Pool.Exec(ctx, sql, regionFilter, userID)
	if err != nil {
		return fmt.Errorf("set region filter for user %d: %w", userID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user with id %d not found", userID)
	}
	return nil
}

// ListPending returns PBM bot users awaiting admin approval (completed onboarding).
func (r *UserRepo) ListPending(ctx context.Context) ([]domain.User, error) {
	sql := `SELECT ` + userSelectCols + `
		FROM users
		WHERE role = $1 AND bot_type = 'pbm' AND onboard_step = ''
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
// ListActivePBM returns all approved PBM bot users (not pending/rejected).
func (r *UserRepo) ListActivePBM(ctx context.Context) ([]domain.User, error) {
	sql := `SELECT ` + userSelectCols + `
		FROM users
		WHERE bot_type = 'pbm' AND role NOT IN ($1, $2)
		ORDER BY role ASC, full_name ASC`
	rows, err := r.db.Pool.Query(ctx, sql, domain.RolePending, domain.RoleRejected)
	if err != nil {
		return nil, fmt.Errorf("list active pbm users: %w", err)
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

// SetOnboardData updates onboarding progress (step + optional full_name/company_name/email).
func (r *UserRepo) SetOnboardData(ctx context.Context, userID int, step, fullName, companyName, email string) error {
	sql := `UPDATE users SET onboard_step = $1, full_name = $2, company_name = $3, email = $4 WHERE id = $5`
	_, err := r.db.Pool.Exec(ctx, sql, step, fullName, companyName, email, userID)
	return err
}

// SetOnboardMsgID saves the bot message ID used for inline editing during onboarding.
func (r *UserRepo) SetOnboardMsgID(ctx context.Context, userID int, msgID int) error {
	sql := `UPDATE users SET onboard_msg_id = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, msgID, userID)
	return err
}

// ResetOnboard clears onboarding state for re-registration (after reject).
func (r *UserRepo) ResetOnboard(ctx context.Context, userID int) error {
	sql := `UPDATE users SET onboard_step = '', full_name = '', company_name = '', email = '', 
		onboard_msg_id = NULL, partner_id = NULL, email_verified = false, role = $1 WHERE id = $2`
	_, err := r.db.Pool.Exec(ctx, sql, domain.RolePending, userID)
	return err
}
