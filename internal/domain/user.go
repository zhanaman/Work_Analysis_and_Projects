package domain

import "time"

// Role defines user access level.
type Role string

const (
	RolePending Role = "pending" // Awaiting admin approval
	RoleUser    Role = "user"    // Approved user (future: partner self-service)
	RolePBM     Role = "pbm"    // PBM colleague (e.g. Daulet — RMC only)
	RoleAdmin   Role = "admin"  // Full access (all CCA)
)

// User represents a Telegram user registered with the bot.
type User struct {
	ID           int       `json:"id"`
	TelegramID   int64     `json:"telegram_id"`
	Username     string    `json:"username"`
	FullName     string    `json:"full_name"`
	Role         Role      `json:"role"`
	RegionFilter string    `json:"region_filter"` // "" = all CCA, "RMC" = RMC only
	CreatedAt    time.Time `json:"created_at"`
}

// IsAuthorized returns true if the user has at least "user" access.
func (u *User) IsAuthorized() bool {
	return u.Role == RoleUser || u.Role == RolePBM || u.Role == RoleAdmin
}

// IsAdmin returns true if the user has admin access.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPBM returns true if the user is a PBM or admin.
func (u *User) IsPBM() bool {
	return u.Role == RolePBM || u.Role == RoleAdmin
}

// CCACountries returns the list of CCA countries for filtering.
var CCACountries = []string{
	"Kazakhstan",
	"Azerbaijan",
	"Uzbekistan",
	"Kyrgyzstan",
	"Turkmenistan",
	"Georgia",
	"Armenia",
	"Tajikistan",
}

// RMCCountries returns non-KZ CCA countries (Daulet's region).
var RMCCountries = []string{
	"Azerbaijan",
	"Uzbekistan",
	"Kyrgyzstan",
	"Turkmenistan",
	"Georgia",
	"Armenia",
	"Tajikistan",
}
