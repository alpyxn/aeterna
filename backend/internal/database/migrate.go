package database

import (
	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/database/migrations"
	"gorm.io/gorm"
)

// RunMigrations executes ordered startup migrations from database/migrations.
func RunMigrations(db *gorm.DB, cfg config.Config) error {
	return migrations.RunAll(db, cfg)
}

// MigrateLegacyToMultitenant is kept as a compatibility wrapper.
func MigrateLegacyToMultitenant(db *gorm.DB, cfg config.Config) error {
	return migrations.MigrateLegacyToMultitenant(db, cfg)
}

// EnsureRefreshSessionIDIntegrity is kept as a compatibility wrapper.
func EnsureRefreshSessionIDIntegrity(db *gorm.DB) error {
	return migrations.EnsureRefreshSessionIDIntegrity(db)
}
