package services

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/config/common"
	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSessionTTL(t *testing.T) {
	tests := []struct {
		name  string
		hours int
		want  time.Duration
	}{
		{"default mobile app window", common.DefaultSessionTTLHours, 168 * time.Hour},
		{"twelve hours", 12, 12 * time.Hour},
		{"six hours", 6, 6 * time.Hour},
		{"one hour", 1, time.Hour},
		{"large value", 720, 720 * time.Hour},
		{"zero falls back to default", 0, time.Duration(common.DefaultSessionTTLHours) * time.Hour},
		{"negative falls back to default", -4, time.Duration(common.DefaultSessionTTLHours) * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewAuthService(config.Config{Auth: config.AuthConfig{SessionTTLHours: tc.hours}})
			if got := svc.sessionTTL(); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	replacer := strings.NewReplacer("/", "_", " ", "_")
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared&_foreign_keys=1", replacer.Replace(t.Name()), time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.RefreshSession{}); err != nil {
		t.Fatal(err)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func createAuthTestUser(t *testing.T, db *gorm.DB, id, email, password string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := models.User{
		ID:           id,
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

func TestRefreshTTL(t *testing.T) {
	tests := []struct {
		name  string
		hours int
		want  time.Duration
	}{
		{"default refresh window", common.DefaultRefreshTTLHours, 720 * time.Hour},
		{"custom refresh window", 336, 336 * time.Hour},
		{"zero falls back to default", 0, time.Duration(common.DefaultRefreshTTLHours) * time.Hour},
		{"negative falls back to default", -4, time.Duration(common.DefaultRefreshTTLHours) * time.Hour},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewAuthService(config.Config{Auth: config.AuthConfig{RefreshTTLHours: tc.hours}})
			if got := svc.refreshTTL(); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestIssueAndRefreshSessionPair_RotatesRefreshToken(t *testing.T) {
	db := setupAuthTestDB(t)
	initTestKeyManager(t)
	user := createAuthTestUser(t, db, "u1", "user@example.com", "StrongPass1!")

	svc := NewAuthService(config.Config{
		Auth: config.AuthConfig{
			SessionTTLHours: 1,
			RefreshTTLHours: 24,
		},
	})

	accessToken, accessExp, refreshToken, refreshExp, err := svc.IssueSessionPair(user.ID)
	if err != nil {
		t.Fatalf("IssueSessionPair failed: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Fatal("expected non-empty access and refresh tokens")
	}
	if !accessExp.After(time.Now().UTC()) || !refreshExp.After(time.Now().UTC()) {
		t.Fatal("expected token expirations in the future")
	}
	if gotUserID, verifyErr := svc.VerifySessionToken(accessToken); verifyErr != nil || gotUserID != user.ID {
		t.Fatalf("VerifySessionToken failed: user=%q err=%v", gotUserID, verifyErr)
	}

	userID, nextAccess, nextAccessExp, nextRefresh, nextRefreshExp, err := svc.RefreshSessionPair(refreshToken)
	if err != nil {
		t.Fatalf("RefreshSessionPair failed: %v", err)
	}
	if userID != user.ID {
		t.Fatalf("expected user %q, got %q", user.ID, userID)
	}
	if nextAccess == "" || nextRefresh == "" {
		t.Fatal("expected rotated non-empty tokens")
	}
	if nextRefresh == refreshToken {
		t.Fatal("expected rotated refresh token to differ from previous one")
	}
	if !nextAccessExp.After(time.Now().UTC()) || !nextRefreshExp.After(time.Now().UTC()) {
		t.Fatal("expected rotated expirations in the future")
	}

	if _, _, _, _, _, err := svc.RefreshSessionPair(refreshToken); err == nil {
		t.Fatal("expected old refresh token to be rejected after rotation")
	}
	if err := svc.RevokeRefreshToken(nextRefresh); err != nil {
		t.Fatalf("RevokeRefreshToken failed: %v", err)
	}
	if _, _, _, _, _, err := svc.RefreshSessionPair(nextRefresh); err == nil {
		t.Fatal("expected revoked refresh token to be rejected")
	}
}

func TestRefreshSessionPair_AllowsSingleConcurrentUse(t *testing.T) {
	db := setupAuthTestDB(t)
	initTestKeyManager(t)
	user := createAuthTestUser(t, db, "u2", "user2@example.com", "StrongPass1!")

	svc := NewAuthService(config.Config{
		Auth: config.AuthConfig{
			SessionTTLHours: 1,
			RefreshTTLHours: 24,
		},
	})

	_, _, refreshToken, _, err := svc.IssueSessionPair(user.ID)
	if err != nil {
		t.Fatalf("IssueSessionPair failed: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _, _, _, _, refreshErr := svc.RefreshSessionPair(refreshToken)
			results <- refreshErr
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0
	failures := 0
	for refreshErr := range results {
		if refreshErr == nil {
			successes++
			continue
		}
		failures++
	}

	if successes != 1 || failures != 1 {
		t.Fatalf("expected exactly one success and one failure, got successes=%d failures=%d", successes, failures)
	}
}
