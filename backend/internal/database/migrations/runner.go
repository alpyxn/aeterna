package migrations

import (
	"fmt"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"gorm.io/gorm"
)

// Step defines one ordered startup migration.
type Step struct {
	Date        string
	Name        string
	Description string
	Run         func(db *gorm.DB, cfg config.Config) error
}

var orderedSteps = []Step{
	{
		Date:        "20250508",
		Name:        "legacy_multitenant_backfill",
		Description: "Assign legacy single-tenant rows to a user and normalize orphan user_id values.",
		Run:         MigrateLegacyToMultitenant,
	},
	{
		Date:        "20260531",
		Name:        "refresh_session_id_integrity",
		Description: "Backfill refresh session IDs and enforce NOT NULL on refresh_sessions.session_id.",
		Run: func(db *gorm.DB, _ config.Config) error {
			return EnsureRefreshSessionIDIntegrity(db)
		},
	},
}

// RunAll executes all startup migrations in deterministic date/name order.
func RunAll(db *gorm.DB, cfg config.Config) error {
	for _, step := range orderedSteps {
		if err := step.Run(db, cfg); err != nil {
			return fmt.Errorf("migration %s_%s failed: %w", step.Date, step.Name, err)
		}
	}
	return nil
}
