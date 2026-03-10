package domain

import "time"

// Role defines user access level.
type Role string

const (
	RolePending  Role = "pending"  // Awaiting admin approval
	RoleRejected Role = "rejected" // Rejected by admin — blocked
	RoleUser     Role = "user"     // Approved user (future: partner self-service)
	RoleDistri   Role = "distri"   // Distributor — sees sub-partner cards
	RolePBM      Role = "pbm"      // PBM colleague (e.g. Daulet — RMC only)
	RoleAdmin    Role = "admin"    // Full access (all CCA)
)

// User represents a Telegram user registered with the bot.
type User struct {
	ID            int       `json:"id"`
	TelegramID    int64     `json:"telegram_id"`
	Username      string    `json:"username"`
	FullName      string    `json:"full_name"`
	Role          Role      `json:"role"`
	RegionFilter  string    `json:"region_filter"` // "" = all CCA, "RMC" = RMC only
	PartnerID     *int      `json:"partner_id"`    // NULL for PBM/admin, set for partner users
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Lang          string    `json:"lang"`           // "ru" | "en"
	BotType       string    `json:"bot_type"`       // "pbm" | "partner"
	OnboardStep   string    `json:"onboard_step"`   // "" | "name" | "company" | "email"
	CompanyName   string    `json:"company_name"`   // Partner-entered company name
	OnboardMsgID  *int      `json:"onboard_msg_id"` // Bot message ID for inline editing
	CreatedAt     time.Time `json:"created_at"`
}

// IsAuthorized returns true if the user has at least "user" access.
func (u *User) IsAuthorized() bool {
	return u.Role == RoleUser || u.Role == RoleDistri || u.Role == RolePBM || u.Role == RoleAdmin
}

// IsAdmin returns true if the user has admin access.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPBM returns true if the user is a PBM or admin.
func (u *User) IsPBM() bool {
	return u.Role == RolePBM || u.Role == RoleAdmin
}

// IsDistri returns true if the user is a distributor.
func (u *User) IsDistri() bool {
	return u.Role == RoleDistri
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
