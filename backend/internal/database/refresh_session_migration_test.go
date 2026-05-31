package database

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func refreshSessionTestDSN(t *testing.T) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "_", " ", "_")
	return fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_foreign_keys=1", replacer.Replace(t.Name()), time.Now().UnixNano())
}

func TestEnsureRefreshSessionIDIntegrity_MigratesNullableSchemaToNotNull(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{
		ID:           "u-migrate",
		Email:        "migrate@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	// Legacy schema: session_id is nullable.
	if err := db.Exec(`
		CREATE TABLE refresh_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			session_id TEXT,
			token_hash TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			revoked_at DATETIME,
			replaced_by_token_hash TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);
	`).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	if err := db.Exec(`
		INSERT INTO refresh_sessions (
			id, user_id, session_id, token_hash, expires_at, created_at, updated_at
		) VALUES
			(?, ?, NULL, ?, ?, ?, ?),
			(?, ?, '', ?, ?, ?, ?),
			(?, ?, 'sess-keep', ?, ?, ?, ?);
	`,
		"rs-null", user.ID, "hash-null", now.Add(time.Hour), now, now,
		"rs-empty", user.ID, "hash-empty", now.Add(time.Hour), now, now,
		"rs-keep", user.ID, "hash-keep", now.Add(time.Hour), now, now,
	).Error; err != nil {
		t.Fatal(err)
	}

	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity failed: %v", err)
	}
	// Idempotency.
	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity should be idempotent: %v", err)
	}

	type refreshRow struct {
		ID        string
		SessionID string
	}
	var rows []refreshRow
	if err := db.Raw("SELECT id, session_id FROM refresh_sessions ORDER BY id ASC").Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 refresh sessions, got %d", len(rows))
	}

	got := map[string]string{}
	for _, row := range rows {
		got[row.ID] = row.SessionID
		if strings.TrimSpace(row.SessionID) == "" {
			t.Fatalf("row %s still has empty session_id", row.ID)
		}
	}
	if got["rs-null"] != "rs-null" {
		t.Fatalf("expected rs-null session_id backfilled to id, got %q", got["rs-null"])
	}
	if got["rs-empty"] != "rs-empty" {
		t.Fatalf("expected rs-empty session_id backfilled to id, got %q", got["rs-empty"])
	}
	if got["rs-keep"] != "sess-keep" {
		t.Fatalf("expected existing session_id preserved, got %q", got["rs-keep"])
	}

	type pragmaColumn struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	var columns []pragmaColumn
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		t.Fatal(err)
	}
	notNull := 0
	for _, column := range columns {
		if column.Name == "session_id" {
			notNull = column.NotNull
			break
		}
	}
	if notNull != 1 {
		t.Fatalf("expected refresh_sessions.session_id to be NOT NULL, got notnull=%d", notNull)
	}

	dupErr := db.Exec(`
		INSERT INTO refresh_sessions (id, user_id, session_id, token_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?);
	`, "rs-dup", user.ID, "sess-dup", "hash-keep", now.Add(time.Hour), now, now).Error
	if dupErr == nil {
		t.Fatal("expected unique token_hash index to reject duplicates after migration")
	}
}
