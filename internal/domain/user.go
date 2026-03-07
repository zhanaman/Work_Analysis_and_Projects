package domain

import "time"

// Role defines user access level.
type Role string

const (
	RolePending Role = "pending" // Awaiting admin approval
	RoleUser    Role = "user"    // Approved user
	RoleAdmin   Role = "admin"   // Full access
)

// User represents a Telegram user registered with the bot.
type User struct {
	ID         int       `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	Username   string    `json:"username"`
	FullName   string    `json:"full_name"`
	Role       Role      `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}

// IsAuthorized returns true if the user has at least "user" access.
func (u *User) IsAuthorized() bool {
	return u.Role == RoleUser || u.Role == RoleAdmin
}

// IsAdmin returns true if the user has admin access.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
