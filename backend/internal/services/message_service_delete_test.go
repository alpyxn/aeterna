package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testSQLiteDSN(t *testing.T) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "_", " ", "_")
	return fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_foreign_keys=1", replacer.Replace(t.Name()), time.Now().UnixNano())
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(testSQLiteDSN(t)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.Message{},
		&models.MessageReminder{},
		&models.Attachment{},
		&models.FarewellLetter{},
		&models.FarewellAttachment{},
	); err != nil {
		t.Fatal(err)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func TestMessageDelete_NoFarewellNoAttachments(t *testing.T) {
	db := setupTestDB(t)
	if err := db.Create(&models.Message{
		ID: "m1", UserID: "u1", Content: "x", KeyFragment: "v1",
		ManagementToken: "tok", RecipientEmail: "a@a.com",
		TriggerDuration: 60, LastSeen: time.Now(), Status: models.StatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := (MessageService{}).Delete("u1", "m1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMessageList_IncludesFarewellCounts(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)
	lastSeen := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	encrypted, err := (CryptoService{}).Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Message{
		ID: "m-counts", UserID: "u-counts", Content: encrypted, KeyFragment: "v1",
		ManagementToken: "tok", RecipientEmail: "a@a.com",
		TriggerDuration: 60, LastSeen: lastSeen, Status: models.StatusTriggered,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, letter := range []models.FarewellLetter{
		{
			ID:             "l-counts-pending",
			UserID:         "u-counts",
			MessageID:      "m-counts",
			RecipientEmail: "pending@example.com",
			Subject:        "Pending",
			Content:        "encrypted",
			DelayMinutes:   60,
			Status:         models.FarewellStatusPending,
		},
		{
			ID:             "l-counts-sent",
			UserID:         "u-counts",
			MessageID:      "m-counts",
			RecipientEmail: "sent@example.com",
			Subject:        "Sent",
			Content:        "encrypted",
			DelayMinutes:   0,
			Status:         models.FarewellStatusSent,
		},
	} {
		if err := db.Create(&letter).Error; err != nil {
			t.Fatal(err)
		}
	}

	messages, err := (MessageService{}).List("u-counts")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].FarewellCount != 2 {
		t.Fatalf("expected farewell_count=2, got %d", messages[0].FarewellCount)
	}
	if messages[0].PendingFarewells != 1 {
		t.Fatalf("expected pending_farewells=1, got %d", messages[0].PendingFarewells)
	}
	if messages[0].NextTriggerAt == nil {
		t.Fatal("expected next_trigger_at to be populated")
	}
	expectedTrigger := lastSeen.Add(60 * time.Minute)
	if !messages[0].NextTriggerAt.Equal(expectedTrigger) {
		t.Fatalf("expected next_trigger_at=%s, got %s", expectedTrigger.Format(time.RFC3339), messages[0].NextTriggerAt.Format(time.RFC3339))
	}
	if messages[0].NextReminderAt != nil {
		t.Fatalf("expected next_reminder_at to be nil for triggered message, got %s", messages[0].NextReminderAt.Format(time.RFC3339))
	}
}

func TestMessageList_ComputesNextReminderAtForPendingReminders(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)
	lastSeen := time.Date(2024, 2, 15, 9, 30, 0, 0, time.UTC)
	encrypted, err := (CryptoService{}).Encrypt("hello")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Message{
		ID: "m-reminder", UserID: "u-reminder", Content: encrypted, KeyFragment: "v1",
		ManagementToken: "tok", RecipientEmail: "a@a.com",
		TriggerDuration: 180, LastSeen: lastSeen, Status: models.StatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, reminder := range []models.MessageReminder{
		{MessageID: "m-reminder", MinutesBefore: 30, Sent: false},
		{MessageID: "m-reminder", MinutesBefore: 120, Sent: false},
		{MessageID: "m-reminder", MinutesBefore: 5, Sent: true},
	} {
		if err := db.Create(&reminder).Error; err != nil {
			t.Fatal(err)
		}
	}

	messages, err := (MessageService{}).List("u-reminder")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	msg := messages[0]
	if msg.NextTriggerAt == nil {
		t.Fatal("expected next_trigger_at to be populated")
	}
	expectedTrigger := lastSeen.Add(180 * time.Minute)
	if !msg.NextTriggerAt.Equal(expectedTrigger) {
		t.Fatalf("expected next_trigger_at=%s, got %s", expectedTrigger.Format(time.RFC3339), msg.NextTriggerAt.Format(time.RFC3339))
	}
	if msg.NextReminderAt == nil {
		t.Fatal("expected next_reminder_at to be populated")
	}
	expectedReminder := expectedTrigger.Add(-120 * time.Minute)
	if !msg.NextReminderAt.Equal(expectedReminder) {
		t.Fatalf("expected next_reminder_at=%s, got %s", expectedReminder.Format(time.RFC3339), msg.NextReminderAt.Format(time.RFC3339))
	}
}

func TestMessageDelete_WithFarewellLetter(t *testing.T) {
	db := setupTestDB(t)
	if err := db.Create(&models.Message{
		ID: "m1", UserID: "u1", Content: "x", KeyFragment: "v1",
		ManagementToken: "tok", RecipientEmail: "a@a.com",
		TriggerDuration: 60, LastSeen: time.Now(), Status: models.StatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.FarewellLetter{
		ID: "l1", UserID: "u1", MessageID: "m1",
		RecipientEmail: "b@b.com", Subject: "bye", Content: "x",
		DelayMinutes: 60, Status: models.FarewellStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := (MessageService{}).Delete("u1", "m1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	var count int64
	db.Unscoped().Model(&models.FarewellLetter{}).Where("message_id = ?", "m1").Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 farewell letters after delete, got %d", count)
	}
}
