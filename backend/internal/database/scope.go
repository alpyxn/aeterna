package database

import (
	"gorm.io/gorm"
)

// ScopeForUser returns a GORM scope that restricts queries to the given tenant user ID.
func ScopeForUser(userID string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID)
	}
}
