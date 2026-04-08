package database

import (
	"strings"

	"gorm.io/gorm"
)

// ForTenant returns a *gorm.DB scoped with ScopeForUser(userID). Use it for every
// query that touches tenant-owned rows (messages, attachments, webhooks tied to a user, etc.)
// instead of using DB directly without a scope.
//
// Empty or whitespace userID returns a chain that matches no rows (WHERE 1=0), so accidental
// unscoped access with a missing id fails closed instead of leaking data.
func ForTenant(userID string) *gorm.DB {
	if strings.TrimSpace(userID) == "" {
		return DB.Where("1 = 0")
	}
	return DB.Scopes(ScopeForUser(userID))
}

// TenantTx applies the same tenant scope to an existing handle (e.g. inside database.DB.Transaction).
func TenantTx(tx *gorm.DB, userID string) *gorm.DB {
	if strings.TrimSpace(userID) == "" {
		return tx.Where("1 = 0")
	}
	return tx.Scopes(ScopeForUser(userID))
}
