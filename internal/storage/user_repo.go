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
		RETURNING id, telegram_id, username, full_name, role, COALESCE(region_filter, ''), created_at
	`
	u = &domain.User{}
	err = r.db.Pool.QueryRow(ctx, sql, telegramID, username, fullName, domain.RolePending).
		Scan(&u.ID, &u.TelegramID, &u.Username, &u.FullName, &u.Role, &u.RegionFilter, &u.CreatedAt)
	if err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}

	return u, true, nil
}

// GetByTelegramID finds a user by their Telegram ID.
func (r *UserRepo) GetByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	sql := `
		SELECT id, telegram_id, username, full_name, role, COALESCE(region_filter, ''), created_at
		FROM users
		WHERE telegram_id = $1
	`
	u := &domain.User{}
	err := r.db.Pool.QueryRow(ctx, sql, telegramID).
		Scan(&u.ID, &u.TelegramID, &u.Username, &u.FullName, &u.Role, &u.RegionFilter, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by telegram_id %d: %w", telegramID, err)
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

// ListPending returns all users awaiting admin approval.
func (r *UserRepo) ListPending(ctx context.Context) ([]domain.User, error) {
	sql := `
		SELECT id, telegram_id, username, full_name, role, COALESCE(region_filter, ''), created_at
		FROM users
		WHERE role = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.Pool.Query(ctx, sql, domain.RolePending)
	if err != nil {
		return nil, fmt.Errorf("list pending users: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.Username, &u.FullName, &u.Role, &u.RegionFilter, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
