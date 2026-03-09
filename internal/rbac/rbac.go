// Package rbac provides a simple role-based access control registry.
// Permissions are assigned to roles in one place. Handlers check access
// using Can(user, permission) without hardcoding role checks.
//
// To give a role new capabilities, add the permission to RolePermissions.
// To create a new role, add a new entry to the map.
package rbac

import "github.com/anonimouskz/pbm-partner-bot/internal/domain"

// Permission represents a single capability.
type Permission string

const (
	SearchPartners Permission = "search:partners"  // /search + text search
	ViewStats      Permission = "view:stats"        // /stats + stats:* callbacks
	ViewCharts     Permission = "view:charts"       // chart:* callbacks
	ManageUsers    Permission = "manage:users"      // /users + approve/reject
	ViewOwnCard    Permission = "view:own_card"     // Partner: /status
)

// RolePermissions maps each role to its allowed permissions.
// Edit this map to grant/revoke capabilities for any role.
var RolePermissions = map[domain.Role][]Permission{
	domain.RoleAdmin: {SearchPartners, ViewStats, ViewCharts, ManageUsers},
	domain.RolePBM:   {SearchPartners},
	domain.RoleUser:  {SearchPartners},
	// Partner bot users use ViewOwnCard but are checked in partner-bot middleware
}

// Can returns true if the user's role has the given permission.
func Can(user *domain.User, perm Permission) bool {
	if user == nil {
		return false
	}
	perms, ok := RolePermissions[user.Role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}
