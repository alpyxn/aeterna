package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

func initTestKeyManager(t *testing.T) {
	t.Helper()
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}

	dir := t.TempDir()
	keyPath := filepath.Join(dir, "enc.key")
	if err := os.WriteFile(keyPath, []byte(key), 0600); err != nil {
		t.Fatalf("failed to write test key: %v", err)
	}
	if err := os.Chmod(keyPath, 0600); err != nil {
		t.Fatalf("failed to chmod test key: %v", err)
	}

	InitKeyManager(keyPath)
}

func TestFarewellCreate_PersistsZeroDelay(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)

	msg := models.Message{
		ID:              "m-delay-zero",
		UserID:          "u-delay-zero",
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

	letter, err := (FarewellService{}).Create(
		msg.UserID,
		msg.ID,
		"recipient@example.com",
		"Subject",
		"hello from aeterna",
		0,
	)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if letter.DelayMinutes != 0 {
		t.Fatalf("expected returned delay 0, got %d", letter.DelayMinutes)
	}
	if letter.WordCount != 0 {
		t.Fatalf("expected returned word_count 0 before background derivation, got %d", letter.WordCount)
	}
	if !letter.DerivativesPending {
		t.Fatalf("expected derivatives_pending=true after create")
	}

	var stored models.FarewellLetter
	if err := db.First(&stored, "id = ?", letter.ID).Error; err != nil {
		t.Fatalf("failed to load stored farewell letter: %v", err)
	}
	if stored.DelayMinutes != 0 {
		t.Fatalf("expected stored delay 0, got %d", stored.DelayMinutes)
	}
	if stored.WordCount != 0 {
		t.Fatalf("expected stored word_count 0 before background derivation, got %d", stored.WordCount)
	}
	if !stored.DerivativesPending {
		t.Fatalf("expected stored derivatives_pending=true after create")
	}
}

func TestFarewellCreate_RejectsTriggeredMessage(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-triggered-create",
		UserID:          "u-triggered-create",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusTriggered,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}

	_, err := (FarewellService{}).Create(
		msg.UserID,
		msg.ID,
		"recipient@example.com",
		"Subject",
		"content",
		10,
	)
	if err == nil || !strings.Contains(err.Error(), "Cannot add farewell letters after the switch has triggered") {
		t.Fatalf("expected triggered create rejection, got %v", err)
	}
}

func TestFarewellUpdate_MarksDerivativesPending(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)

	msg := models.Message{
		ID:              "m-word-count",
		UserID:          "u-word-count",
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

	created, err := (FarewellService{}).Create(
		msg.UserID,
		msg.ID,
		"recipient@example.com",
		"Subject",
		"one two",
		0,
	)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := (FarewellService{}).Update(
		msg.UserID,
		msg.ID,
		created.ID,
		"recipient@example.com",
		"Updated subject",
		"one two three four five",
		10,
	)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.WordCount != 0 {
		t.Fatalf("expected returned word_count 0 before background derivation, got %d", updated.WordCount)
	}
	if !updated.DerivativesPending {
		t.Fatalf("expected derivatives_pending=true after update")
	}

	var stored models.FarewellLetter
	if err := db.First(&stored, "id = ?", created.ID).Error; err != nil {
		t.Fatalf("failed to load stored farewell letter: %v", err)
	}
	if stored.WordCount != 0 {
		t.Fatalf("expected stored word_count 0 before background derivation, got %d", stored.WordCount)
	}
	if !stored.DerivativesPending {
		t.Fatalf("expected stored derivatives_pending=true after update")
	}
}

func TestFarewellUpdate_RejectsTriggeredMessage(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-triggered-update",
		UserID:          "u-triggered-update",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusTriggered,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.FarewellLetter{
		ID:             "l-triggered-update",
		UserID:         msg.UserID,
		MessageID:      msg.ID,
		RecipientEmail: "recipient@example.com",
		Subject:        "Subject",
		Content:        "encrypted",
		DelayMinutes:   60,
		Status:         models.FarewellStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}

	_, err := (FarewellService{}).Update(
		msg.UserID,
		msg.ID,
		"l-triggered-update",
		"recipient@example.com",
		"Updated subject",
		"updated content",
		10,
	)
	if err == nil || !strings.Contains(err.Error(), "Cannot edit farewell letters after the switch has triggered") {
		t.Fatalf("expected triggered update rejection, got %v", err)
	}
}

func TestFarewellDelete_RejectsTriggeredMessage(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-triggered-delete",
		UserID:          "u-triggered-delete",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusTriggered,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.FarewellLetter{
		ID:             "l-triggered-delete",
		UserID:         msg.UserID,
		MessageID:      msg.ID,
		RecipientEmail: "recipient@example.com",
		Subject:        "Subject",
		Content:        "encrypted",
		DelayMinutes:   60,
		Status:         models.FarewellStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}

	err := (FarewellService{}).Delete(msg.UserID, msg.ID, "l-triggered-delete")
	if err == nil || !strings.Contains(err.Error(), "Cannot delete farewell letters after the switch has triggered") {
		t.Fatalf("expected triggered delete rejection, got %v", err)
	}
}

func TestFarewellCancelPending_RemovesOnlyPendingLetters(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-cancel-pending",
		UserID:          "u-cancel-pending",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusTriggered,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}
	for _, letter := range []models.FarewellLetter{
		{
			ID:             "l-pending-1",
			UserID:         msg.UserID,
			MessageID:      msg.ID,
			RecipientEmail: "one@example.com",
			Subject:        "One",
			Content:        "encrypted",
			DelayMinutes:   60,
			Status:         models.FarewellStatusPending,
		},
		{
			ID:             "l-pending-2",
			UserID:         msg.UserID,
			MessageID:      msg.ID,
			RecipientEmail: "two@example.com",
			Subject:        "Two",
			Content:        "encrypted",
			DelayMinutes:   120,
			Status:         models.FarewellStatusPending,
		},
		{
			ID:             "l-sent",
			UserID:         msg.UserID,
			MessageID:      msg.ID,
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

	count, err := (FarewellService{}).CancelPendingByMessageID(msg.UserID, msg.ID)
	if err != nil {
		t.Fatalf("CancelPendingByMessageID failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 canceled letters, got %d", count)
	}

	var pendingCount int64
	db.Model(&models.FarewellLetter{}).Where("message_id = ? AND status = ?", msg.ID, models.FarewellStatusPending).Count(&pendingCount)
	if pendingCount != 0 {
		t.Fatalf("expected no pending letters after cancel, got %d", pendingCount)
	}
	var sentCount int64
	db.Model(&models.FarewellLetter{}).Where("message_id = ? AND status = ?", msg.ID, models.FarewellStatusSent).Count(&sentCount)
	if sentCount != 1 {
		t.Fatalf("expected sent letter to remain, got %d", sentCount)
	}
}

func TestFarewellCreate_SanitizesStoredContentAndPreservesRaw(t *testing.T) {
	db := setupTestDB(t)
	initTestKeyManager(t)

	msg := models.Message{
		ID:              "m-markdown-word-count",
		UserID:          "u-markdown-word-count",
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

	content := "# Title\n\n<script>alert('x')</script>\n\nVisit [Aeterna](javascript:alert(1))"
	letter, err := (FarewellService{}).Create(
		msg.UserID,
		msg.ID,
		"recipient@example.com",
		"Subject",
		content,
		0,
	)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if letter.WordCount != 0 {
		t.Fatalf("expected returned word_count 0 before background derivation, got %d", letter.WordCount)
	}
	if strings.Contains(strings.ToLower(letter.Content), "<script") {
		t.Fatalf("expected sanitized content without raw html, got: %s", letter.Content)
	}
	if strings.Contains(strings.ToLower(letter.Content), "javascript:") {
		t.Fatalf("expected sanitized content without javascript links, got: %s", letter.Content)
	}

	var stored models.FarewellLetter
	if err := db.First(&stored, "id = ?", letter.ID).Error; err != nil {
		t.Fatalf("failed to load stored farewell letter: %v", err)
	}
	if strings.TrimSpace(stored.RawContent) == "" {
		t.Fatal("expected raw encrypted content to be persisted")
	}
	rawDecrypted, err := (CryptoService{}).Decrypt(stored.RawContent)
	if err != nil {
		t.Fatalf("failed to decrypt raw content: %v", err)
	}
	if rawDecrypted != content {
		t.Fatalf("expected raw content to match input, got %q", rawDecrypted)
	}
}

func TestFarewellCancelPending_RejectsActiveMessage(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-cancel-active",
		UserID:          "u-cancel-active",
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
	if err := db.Create(&models.FarewellLetter{
		ID:             "l-cancel-active",
		UserID:         msg.UserID,
		MessageID:      msg.ID,
		RecipientEmail: "one@example.com",
		Subject:        "One",
		Content:        "encrypted",
		DelayMinutes:   60,
		Status:         models.FarewellStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}

	err := (FarewellService{}).CancelPending(msg.UserID, msg.ID, "l-cancel-active")
	if err == nil || !strings.Contains(err.Error(), cancelRequiresTriggeredMessage) {
		t.Fatalf("expected active message cancellation rejection, got %v", err)
	}

	var count int64
	db.Model(&models.FarewellLetter{}).Where("id = ?", "l-cancel-active").Count(&count)
	if count != 1 {
		t.Fatalf("expected pending letter to remain, got %d", count)
	}
}

func TestFarewellCancelPendingByMessageID_RejectsActiveMessage(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-cancel-all-active",
		UserID:          "u-cancel-all-active",
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
	if err := db.Create(&models.FarewellLetter{
		ID:             "l-cancel-all-active",
		UserID:         msg.UserID,
		MessageID:      msg.ID,
		RecipientEmail: "one@example.com",
		Subject:        "One",
		Content:        "encrypted",
		DelayMinutes:   60,
		Status:         models.FarewellStatusPending,
	}).Error; err != nil {
		t.Fatal(err)
	}

	canceled, err := (FarewellService{}).CancelPendingByMessageID(msg.UserID, msg.ID)
	if err == nil || !strings.Contains(err.Error(), cancelRequiresTriggeredMessage) {
		t.Fatalf("expected active message bulk cancellation rejection, got %v", err)
	}
	if canceled != 0 {
		t.Fatalf("expected no canceled letters, got %d", canceled)
	}

	var count int64
	db.Model(&models.FarewellLetter{}).Where("id = ?", "l-cancel-all-active").Count(&count)
	if count != 1 {
		t.Fatalf("expected pending letter to remain, got %d", count)
	}
}

func TestFarewellCancelPending_RemovesSinglePendingLetter(t *testing.T) {
	db := setupTestDB(t)
	msg := models.Message{
		ID:              "m-cancel-one",
		UserID:          "u-cancel-one",
		Content:         "encrypted",
		KeyFragment:     "v1",
		ManagementToken: "tok",
		RecipientEmail:  "owner@example.com",
		TriggerDuration: 60,
		LastSeen:        time.Now(),
		Status:          models.StatusTriggered,
	}
	if err := db.Create(&msg).Error; err != nil {
		t.Fatal(err)
	}
	for _, letter := range []models.FarewellLetter{
		{
			ID:             "l-cancel-one",
			UserID:         msg.UserID,
			MessageID:      msg.ID,
			RecipientEmail: "one@example.com",
			Subject:        "One",
			Content:        "encrypted",
			DelayMinutes:   60,
			Status:         models.FarewellStatusPending,
		},
		{
			ID:             "l-keep-pending",
			UserID:         msg.UserID,
			MessageID:      msg.ID,
			RecipientEmail: "two@example.com",
			Subject:        "Two",
			Content:        "encrypted",
			DelayMinutes:   120,
			Status:         models.FarewellStatusPending,
		},
	} {
		if err := db.Create(&letter).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := (FarewellService{}).CancelPending(msg.UserID, msg.ID, "l-cancel-one"); err != nil {
		t.Fatalf("CancelPending failed: %v", err)
	}

	var canceledCount int64
	db.Model(&models.FarewellLetter{}).Where("id = ?", "l-cancel-one").Count(&canceledCount)
	if canceledCount != 0 {
		t.Fatalf("expected selected pending letter to be removed, got %d", canceledCount)
	}
	var remainingCount int64
	db.Model(&models.FarewellLetter{}).Where("id = ?", "l-keep-pending").Count(&remainingCount)
	if remainingCount != 1 {
		t.Fatalf("expected other pending letter to remain, got %d", remainingCount)
	}
}
