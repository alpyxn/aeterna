package services

import (
	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
)

// IsFirstUser returns true if userID is the earliest-created account (primary administrator).
func IsFirstUser(userID string) bool {
	if userID == "" {
		return false
	}
	var first models.User
	if err := database.DB.Order("created_at ASC, id ASC").First(&first).Error; err != nil {
		return false
	}
	return first.ID == userID
}
