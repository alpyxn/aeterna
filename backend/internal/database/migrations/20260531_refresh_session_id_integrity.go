package migrations

import (
	"fmt"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

// EnsureRefreshSessionIDIntegrity backfills refresh_sessions.session_id and enforces NOT NULL.
// Safe to call on every startup (idempotent).
func EnsureRefreshSessionIDIntegrity(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.RefreshSession{}) {
		return createRefreshSessionsTable(db)
	}

	info, err := refreshSessionTableInfo(db)
	if err != nil {
		return err
	}

	if !info.hasColumn("session_id") {
		if err := db.Exec("ALTER TABLE refresh_sessions ADD COLUMN session_id TEXT;").Error; err != nil {
			return err
		}
		info, err = refreshSessionTableInfo(db)
		if err != nil {
			return err
		}
	}

	if err := backfillRefreshSessionIDs(db, info); err != nil {
		return err
	}

	needsMigration := info.hasColumn("FOREIGN") || info.notNull("session_id") == 0 || info.hasForeignKeyConstraint()
	if !needsMigration {
		return nil
	}

	if err := migrateRefreshSessionsSessionIDNotNull(db); err != nil {
		return err
	}

	info, err = refreshSessionTableInfo(db)
	if err != nil {
		return err
	}
	return backfillRefreshSessionIDs(db, info)
}

type refreshSessionColumnInfo struct {
	Name    string `gorm:"column:name"`
	NotNull int    `gorm:"column:notnull"`
}

type refreshSessionTableMeta struct {
	columns   []refreshSessionColumnInfo
	createSQL string
}

func (m refreshSessionTableMeta) hasColumn(name string) bool {
	for _, column := range m.columns {
		if strings.EqualFold(column.Name, name) {
			return true
		}
	}
	return false
}

func (m refreshSessionTableMeta) notNull(name string) int {
	for _, column := range m.columns {
		if column.Name == name {
			return column.NotNull
		}
	}
	return 0
}

func (m refreshSessionTableMeta) hasForeignKeyConstraint() bool {
	return strings.Contains(strings.ToUpper(m.createSQL), "FOREIGN KEY")
}

func refreshSessionTableInfo(db *gorm.DB) (refreshSessionTableMeta, error) {
	var columns []refreshSessionColumnInfo
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		return refreshSessionTableMeta{}, err
	}
	if len(columns) == 0 {
		return refreshSessionTableMeta{}, fmt.Errorf("refresh_sessions table has no columns")
	}

	type sqliteMasterRow struct {
		SQL string `gorm:"column:sql"`
	}
	var row sqliteMasterRow
	if err := db.Raw(
		"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1;",
		"refresh_sessions",
	).Scan(&row).Error; err != nil {
		return refreshSessionTableMeta{}, err
	}
	if strings.TrimSpace(row.SQL) == "" {
		return refreshSessionTableMeta{}, fmt.Errorf("refresh_sessions table definition not found in sqlite_master")
	}

	return refreshSessionTableMeta{
		columns:   columns,
		createSQL: row.SQL,
	}, nil
}

func backfillRefreshSessionIDs(db *gorm.DB, info refreshSessionTableMeta) error {
	if !info.hasColumn("session_id") {
		return fmt.Errorf("refresh_sessions.session_id column not found")
	}
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
				updated_at DATETIME
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

		if err := tx.Exec("DROP INDEX IF EXISTS idx_refresh_sessions_token_hash;").Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP INDEX IF EXISTS idx_refresh_sessions_user_id;").Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP INDEX IF EXISTS idx_refresh_sessions_session_id;").Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP INDEX IF EXISTS idx_refresh_sessions_expires_at;").Error; err != nil {
			return err
		}
		if err := tx.Exec("DROP INDEX IF EXISTS idx_refresh_sessions_revoked_at;").Error; err != nil {
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

func createRefreshSessionsTable(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS refresh_sessions (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				session_id TEXT NOT NULL,
				token_hash TEXT NOT NULL,
				expires_at DATETIME NOT NULL,
				revoked_at DATETIME,
				replaced_by_token_hash TEXT,
				created_at DATETIME,
				updated_at DATETIME
			);
		`).Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_sessions_token_hash ON refresh_sessions(token_hash);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_refresh_sessions_user_id ON refresh_sessions(user_id);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_refresh_sessions_session_id ON refresh_sessions(session_id);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_refresh_sessions_expires_at ON refresh_sessions(expires_at);").Error; err != nil {
			return err
		}
		if err := tx.Exec("CREATE INDEX IF NOT EXISTS idx_refresh_sessions_revoked_at ON refresh_sessions(revoked_at);").Error; err != nil {
			return err
		}
		return nil
	})
}
