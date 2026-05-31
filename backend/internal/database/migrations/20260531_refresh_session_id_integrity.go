package migrations

import (
	"fmt"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

// EnsureRefreshSessionIDIntegrity backfills refresh_sessions.session_id and enforces NOT NULL.
// Safe to call on every startup (idempotent).
func EnsureRefreshSessionIDIntegrity(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.RefreshSession{}) {
		return nil
	}

	if err := backfillRefreshSessionIDs(db); err != nil {
		return err
	}

	needsMigration, err := refreshSessionIDNeedsNotNullMigration(db)
	if err != nil {
		return err
	}
	if !needsMigration {
		return nil
	}

	if err := migrateRefreshSessionsSessionIDNotNull(db); err != nil {
		return err
	}

	return backfillRefreshSessionIDs(db)
}

func backfillRefreshSessionIDs(db *gorm.DB) error {
	if err := db.Exec(
		"UPDATE refresh_sessions SET session_id = id WHERE session_id IS NULL OR TRIM(session_id) = '';",
	).Error; err != nil {
		return err
	}

	var invalidCount int64
	if err := db.Raw(
		"SELECT COUNT(1) FROM refresh_sessions WHERE session_id IS NULL OR TRIM(session_id) = '';",
	).Scan(&invalidCount).Error; err != nil {
		return err
	}
	if invalidCount > 0 {
		return fmt.Errorf("refresh_sessions still contains %d rows with empty session_id after backfill", invalidCount)
	}
	return nil
}

func refreshSessionIDNeedsNotNullMigration(db *gorm.DB) (bool, error) {
	type pragmaColumn struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}

	var columns []pragmaColumn
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		return false, err
	}

	for _, column := range columns {
		if column.Name == "session_id" {
			return column.NotNull == 0, nil
		}
	}
	return false, fmt.Errorf("refresh_sessions.session_id column not found")
}

func migrateRefreshSessionsSessionIDNotNull(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			CREATE TABLE refresh_sessions_new (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				session_id TEXT NOT NULL,
				token_hash TEXT NOT NULL,
				expires_at DATETIME NOT NULL,
				revoked_at DATETIME,
				replaced_by_token_hash TEXT,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`).Error; err != nil {
			return err
		}

		if err := tx.Exec(`
			INSERT INTO refresh_sessions_new (
				id,
				user_id,
				session_id,
				token_hash,
				expires_at,
				revoked_at,
				replaced_by_token_hash,
				created_at,
				updated_at
			)
			SELECT
				id,
				user_id,
				COALESCE(NULLIF(TRIM(session_id), ''), id),
				token_hash,
				expires_at,
				revoked_at,
				replaced_by_token_hash,
				created_at,
				updated_at
			FROM refresh_sessions;
		`).Error; err != nil {
			return err
		}

		if err := tx.Exec("DROP TABLE refresh_sessions;").Error; err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE refresh_sessions_new RENAME TO refresh_sessions;").Error; err != nil {
			return err
		}

		if err := tx.Exec("CREATE UNIQUE INDEX idx_refresh_sessions_token_hash ON refresh_sessions(token_hash);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX idx_refresh_sessions_user_id ON refresh_sessions(user_id);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX idx_refresh_sessions_session_id ON refresh_sessions(session_id);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX idx_refresh_sessions_expires_at ON refresh_sessions(expires_at);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX idx_refresh_sessions_revoked_at ON refresh_sessions(revoked_at);").Error; err != nil {
			return err
		}

		return nil
	})
}
