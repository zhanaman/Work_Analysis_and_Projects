package domain

import "time"

// EventType defines the type of activity logged.
type EventType string

const (
	EventSearch      EventType = "search"
	EventPartnerView EventType = "partner_view"
)

// ActivityEntry represents a single logged user action.
type ActivityEntry struct {
	ID          int
	UserID      *int
	TelegramID  int64
	FullName    string    // joined from users table
	Username    string    // joined from users table
	EventType   EventType
	Query       string    // for search events
	PartnerID   *int      // for partner_view events
	PartnerName string    // for partner_view events
	CreatedAt   time.Time
}

// UserActivitySummary aggregates a user's event counts over a time period.
type UserActivitySummary struct {
	TelegramID  int64
	FullName    string
	Username    string
	SearchCount int
	ViewCount   int
	LastActive  time.Time
}

