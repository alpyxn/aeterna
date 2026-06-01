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

func TestEnsureRefreshSessionIDIntegrity_RepairsMalformedForeignColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{
		ID:           "u-malformed",
		Email:        "malformed@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
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
			"FOREIGN" TEXT
		);
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		INSERT INTO refresh_sessions (
			id, user_id, session_id, token_hash, expires_at, created_at, updated_at, "FOREIGN"
		) VALUES (?, ?, '', ?, ?, ?, ?, ?);
	`, "rs-foreign", user.ID, "hash-foreign", now.Add(time.Hour), now, now, "garbage").Error; err != nil {
		t.Fatal(err)
	}

	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity failed: %v", err)
	}

	type pragmaColumn struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	var columns []pragmaColumn
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		t.Fatal(err)
	}
	hasForeign := false
	sessionNotNull := 0
	for _, column := range columns {
		if column.Name == "FOREIGN" {
			hasForeign = true
		}
		if column.Name == "session_id" {
			sessionNotNull = column.NotNull
		}
	}
	if hasForeign {
		t.Fatal("expected malformed FOREIGN column to be removed")
	}
	if sessionNotNull != 1 {
		t.Fatalf("expected session_id NOT NULL after repair, got %d", sessionNotNull)
	}

	var sessionID string
	if err := db.Raw("SELECT session_id FROM refresh_sessions WHERE id = ?", "rs-foreign").Scan(&sessionID).Error; err != nil {
		t.Fatal(err)
	}
	if sessionID != "rs-foreign" {
		t.Fatalf("expected repaired row session_id backfilled from id, got %q", sessionID)
	}
}

func TestEnsureRefreshSessionIDIntegrity_RepairsForeignKeySchemaForAutoMigrate(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}

	user := models.User{
		ID:           "u-fk-only",
		Email:        "fk-only@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
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
		t.Fatal(err)
	}
	if err := db.Exec(`
		INSERT INTO refresh_sessions (
			id, user_id, session_id, token_hash, expires_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?);
	`, "rs-fk", user.ID, "sid-fk", "hash-fk", now.Add(time.Hour), now, now).Error; err != nil {
		t.Fatal(err)
	}

	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity failed: %v", err)
	}

	type masterRow struct {
		SQL string `gorm:"column:sql"`
	}
	var row masterRow
	if err := db.Raw(
		"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1;",
		"refresh_sessions",
	).Scan(&row).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ToUpper(row.SQL), "FOREIGN KEY") {
		t.Fatalf("expected refresh_sessions schema without FOREIGN KEY after repair, got: %s", row.SQL)
	}

	// Reproduce startup step: AutoMigrate should now run without the sqlite migrator FOREIGN failure.
	if err := db.AutoMigrate(&models.RefreshSession{}); err != nil {
		t.Fatalf("expected AutoMigrate to succeed after schema repair, got: %v", err)
	}
}

func TestEnsureRefreshSessionIDIntegrity_CreatesTableWhenMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}

	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity failed: %v", err)
	}

	if !db.Migrator().HasTable("refresh_sessions") {
		t.Fatal("expected refresh_sessions table to be created")
	}

	type pragmaColumn struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	var columns []pragmaColumn
	if err := db.Raw("PRAGMA table_info('refresh_sessions');").Scan(&columns).Error; err != nil {
		t.Fatal(err)
	}

	sessionIDNotNull := 0
	for _, column := range columns {
		if column.Name == "session_id" {
			sessionIDNotNull = column.NotNull
			break
		}
	}
	if sessionIDNotNull != 1 {
		t.Fatalf("expected created schema with session_id NOT NULL, got %d", sessionIDNotNull)
	}
}

func TestEnsureRefreshSessionIDIntegrity_AddsMissingSessionIDColumn(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(refreshSessionTestDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatal(err)
	}
	user := models.User{
		ID:           "u-no-session-id",
		Email:        "no-session-id@example.com",
		PasswordHash: "hash",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	// Legacy schema without session_id column.
	if err := db.Exec(`
		CREATE TABLE refresh_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			revoked_at DATETIME,
			replaced_by_token_hash TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		INSERT INTO refresh_sessions (id, user_id, token_hash, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?);
	`, "rs-no-sid", user.ID, "hash-no-sid", now.Add(time.Hour), now, now).Error; err != nil {
		t.Fatal(err)
	}

	if err := EnsureRefreshSessionIDIntegrity(db); err != nil {
		t.Fatalf("EnsureRefreshSessionIDIntegrity failed: %v", err)
	}

	var sessionID string
	if err := db.Raw("SELECT session_id FROM refresh_sessions WHERE id = ?", "rs-no-sid").Scan(&sessionID).Error; err != nil {
		t.Fatal(err)
	}
	if sessionID != "rs-no-sid" {
		t.Fatalf("expected added session_id backfilled from id, got %q", sessionID)
	}
}
