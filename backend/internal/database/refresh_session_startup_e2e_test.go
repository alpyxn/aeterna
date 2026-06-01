package database

import (
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyRefreshSession struct {
	ID                  string     `gorm:"type:text;primaryKey"`
	UserID              string     `gorm:"type:text;index;not null"`
	TokenHash           string     `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt           time.Time  `gorm:"index;not null"`
	RevokedAt           *time.Time `gorm:"index"`
	ReplacedByTokenHash string     `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (legacyRefreshSession) TableName() string {
	return "refresh_sessions"
}

func TestStartupMigrationPipeline_E2E_FromLegacyRefreshSessions(t *testing.T) {
	db := mustOpenTestDB(t)

	// Create baseline tables and a legacy refresh_sessions schema (without session_id).
	if err := db.AutoMigrate(
		&models.User{},
		&legacyRefreshSession{},
		&models.Message{},
		&models.MessageReminder{},
		&models.Settings{},
		&models.Webhook{},
		&models.Attachment{},
		&models.ApplicationSettings{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		t.Fatalf("bootstrap automigrate failed: %v", err)
	}

	user := models.User{
		ID:           "u-startup-e2e",
		Email:        "startup-e2e@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}

	now := time.Now().UTC()
	if err := db.Create(&legacyRefreshSession{
		ID:        "legacy-rs-1",
		UserID:    user.ID,
		TokenHash: "legacy-token-hash-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("seed legacy refresh session failed: %v", err)
	}

	// Real startup order in main.go:
	// 1) RunPreAutoMigrate (sanitizes malformed/legacy refresh_sessions),
	// 2) AutoMigrate (includes RefreshSession model),
	// 3) RunMigrations (idempotent backfill + hardening).
	if err := RunPreAutoMigrate(db, config.Config{}); err != nil {
		t.Fatalf("startup pre-auto migrations failed: %v", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.RefreshSession{},
		&models.Message{},
		&models.MessageReminder{},
		&models.Settings{},
		&models.Webhook{},
		&models.Attachment{},
		&models.ApplicationSettings{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		t.Fatalf("startup automigrate failed: %v", err)
	}

	if err := RunMigrations(db, config.Config{}); err != nil {
		t.Fatalf("startup run migrations failed: %v", err)
	}

	type pragmaColumn struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	var columns []pragmaColumn
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		t.Fatalf("pragma table_info failed: %v", err)
	}

	sessionIDNotNull := 0
	for _, column := range columns {
		if column.Name == "session_id" {
			sessionIDNotNull = column.NotNull
			break
		}
	}
	if sessionIDNotNull != 1 {
		t.Fatalf("expected refresh_sessions.session_id NOT NULL after startup pipeline, got %d", sessionIDNotNull)
	}

	var sessionID string
	if err := db.Raw("SELECT session_id FROM refresh_sessions WHERE id = ?", "legacy-rs-1").Scan(&sessionID).Error; err != nil {
		t.Fatalf("select migrated session_id failed: %v", err)
	}
	if sessionID != "legacy-rs-1" {
		t.Fatalf("expected backfilled session_id to equal row id, got %q", sessionID)
	}
}

func TestStartupMigrationPipeline_E2E_FromForeignKeyRefreshSessions(t *testing.T) {
	db := mustOpenTestDB(t)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Message{},
		&models.MessageReminder{},
		&models.Settings{},
		&models.Webhook{},
		&models.Attachment{},
		&models.ApplicationSettings{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		t.Fatalf("bootstrap automigrate failed: %v", err)
	}

	user := models.User{
		ID:           "u-startup-fk",
		Email:        "startup-fk@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("seed user failed: %v", err)
	}

	now := time.Now().UTC()
	if err := db.Exec(`
		CREATE TABLE refresh_sessions (
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
		t.Fatalf("create fk schema failed: %v", err)
	}
	if err := db.Exec(`
		INSERT INTO refresh_sessions (
			id, user_id, session_id, token_hash, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?);
	`, "rs-startup-fk", user.ID, "sid-startup-fk", "hash-startup-fk", now.Add(time.Hour), now, now).Error; err != nil {
		t.Fatalf("seed refresh session failed: %v", err)
	}

	if err := runStartupPipeline(db); err != nil {
		t.Fatalf("startup pipeline failed: %v", err)
	}
	// Idempotency across restarts.
	if err := runStartupPipeline(db); err != nil {
		t.Fatalf("startup pipeline second run failed: %v", err)
	}
}

func runStartupPipeline(db *gorm.DB) error {
	if err := RunPreAutoMigrate(db, config.Config{}); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.RefreshSession{},
		&models.Message{},
		&models.MessageReminder{},
		&models.Settings{},
		&models.Webhook{},
		&models.Attachment{},
		&models.ApplicationSettings{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		return err
	}
	return RunMigrations(db, config.Config{})
}

func mustOpenTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
