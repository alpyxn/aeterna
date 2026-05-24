package services

import (
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

func TestFarewellDerivationService_ProcessPending_DerivesAndClearsPending(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)

	msg := models.Message{
		ID:              "m-derive",
		UserID:          "u-derive",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusActive,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}

	svc := FarewellService{}
	created, err := svc.Create(
		msg.UserID,
		msg.ID,
		"recipient@example.com",
		"Subject",
		"# Title\n\nRead <https://example.com> and [site](javascript:alert(1))",
		10,
	)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	derivation := NewFarewellDerivationService()
	processed, err := derivation.ProcessPending(10)
	if err != nil {
		t.Fatalf("process pending failed: %v", err)
	}
	if processed < 1 {
		t.Fatalf("expected at least one processed letter, got %d", processed)
	}

	var stored models.FarewellLetter
	if err := db.First(&stored, "id = ?", created.ID).Error; err != nil {
		t.Fatalf("failed to load stored letter: %v", err)
	}

	if stored.DerivativesPending {
		t.Fatal("expected derivatives_pending=false after derivation")
	}
	if stored.WordCount == 0 {
		t.Fatal("expected non-zero word_count after derivation")
	}
	if strings.TrimSpace(stored.RenderedHTML) == "" {
		t.Fatal("expected encrypted rendered HTML to be persisted")
	}
}
