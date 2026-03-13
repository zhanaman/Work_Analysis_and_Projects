package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
)

// ActivityRepo provides access to the activity_log table.
type ActivityRepo struct {
	db *Postgres
}

// NewActivityRepo creates a new ActivityRepo.
func NewActivityRepo(db *Postgres) *ActivityRepo {
	return &ActivityRepo{db: db}
}

// Log records a user action. Errors are logged but not propagated — logging must never break UX.
func (r *ActivityRepo) Log(ctx context.Context, userID *int, telegramID int64, eventType domain.EventType, query string, partnerID *int, partnerName string) {
	sql := `
		INSERT INTO activity_log (user_id, telegram_id, event_type, query, partner_id, partner_name)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Pool.Exec(ctx, sql, userID, telegramID, string(eventType), query, partnerID, partnerName)
	if err != nil {
		slog.Warn("activity_log: failed to log event", "type", eventType, "telegram_id", telegramID, "error", err)
	}
}

// GetByUser returns the most recent events for a specific Telegram user.
func (r *ActivityRepo) GetByUser(ctx context.Context, telegramID int64, limit int) ([]domain.ActivityEntry, error) {
	sql := `
		SELECT a.id, a.user_id, a.telegram_id,
		       COALESCE(u.full_name, ''), COALESCE(u.username, ''),
		       a.event_type, COALESCE(a.query, ''), a.partner_id, COALESCE(a.partner_name, ''),
		       a.created_at
		FROM activity_log a
		LEFT JOIN users u ON u.telegram_id = a.telegram_id AND u.bot_type = 'pbm'
		WHERE a.telegram_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2
	`
	return r.scanEntries(ctx, sql, telegramID, limit)
}

// GetAll returns the most recent events across all users, within the last N days.
func (r *ActivityRepo) GetAll(ctx context.Context, days int, limit int) ([]domain.ActivityEntry, error) {
	since := time.Now().AddDate(0, 0, -days)
	sql := `
		SELECT a.id, a.user_id, a.telegram_id,
		       COALESCE(u.full_name, ''), COALESCE(u.username, ''),
		       a.event_type, COALESCE(a.query, ''), a.partner_id, COALESCE(a.partner_name, ''),
		       a.created_at
		FROM activity_log a
		LEFT JOIN users u ON u.telegram_id = a.telegram_id AND u.bot_type = 'pbm'
		WHERE a.created_at >= $1
		ORDER BY a.created_at DESC
		LIMIT $2
	`
	return r.scanEntries(ctx, sql, since, limit)
}

// GetUserSummary returns per-user event counts for the last N days.
func (r *ActivityRepo) GetUserSummary(ctx context.Context, days int) ([]domain.UserActivitySummary, error) {
	since := time.Now().AddDate(0, 0, -days)
	sql := `
		SELECT a.telegram_id,
		       COALESCE(u.full_name, ''),
		       COALESCE(u.username, ''),
		       COUNT(*) FILTER (WHERE a.event_type = 'search')       AS search_count,
		       COUNT(*) FILTER (WHERE a.event_type = 'partner_view') AS view_count,
		       MAX(a.created_at)                                      AS last_active
		FROM activity_log a
		LEFT JOIN users u ON u.telegram_id = a.telegram_id AND u.bot_type = 'pbm'
		WHERE a.created_at >= $1
		GROUP BY a.telegram_id, u.full_name, u.username
		ORDER BY last_active DESC
	`
	rows, err := r.db.Pool.Query(ctx, sql, since)
	if err != nil {
		return nil, fmt.Errorf("activity: get user summary: %w", err)
	}
	defer rows.Close()

	var results []domain.UserActivitySummary
	for rows.Next() {
		var s domain.UserActivitySummary
		if err := rows.Scan(&s.TelegramID, &s.FullName, &s.Username, &s.SearchCount, &s.ViewCount, &s.LastActive); err != nil {
			return nil, fmt.Errorf("activity: scan summary: %w", err)
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

func (r *ActivityRepo) scanEntries(ctx context.Context, sql string, args ...any) ([]domain.ActivityEntry, error) {
	rows, err := r.db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("activity: query: %w", err)
	}
	defer rows.Close()

	var entries []domain.ActivityEntry
	for rows.Next() {
		var e domain.ActivityEntry
		var eventType string
		if err := rows.Scan(
			&e.ID, &e.UserID, &e.TelegramID,
			&e.FullName, &e.Username,
			&eventType, &e.Query, &e.PartnerID, &e.PartnerName,
			&e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("activity: scan: %w", err)
		}
		e.EventType = domain.EventType(eventType)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
