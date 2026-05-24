package handlers

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type fakeAuthService struct {
	verifyToken      string
	verifyUser       string
	verifyErr        error
	issuedToken      string
	issuedExp        time.Time
	issuedRefresh    string
	issuedRefreshExp time.Time
	refreshToken     string
	refreshUser      string
	refreshErr       error
	refreshNextToken string
	refreshNextExp   time.Time
	revokeErr        error
}

func (f fakeAuthService) IsConfigured() (bool, error) {
	return false, nil
}

func (f fakeAuthService) RegisterFirstUser(email, password, ownerEmail string) (string, models.User, error) {
	return "", models.User{}, nil
}

func (f fakeAuthService) RegisterAdditionalUser(email, password, ownerEmail string) (string, models.User, error) {
	return "", models.User{}, nil
}

func (f fakeAuthService) Login(email, password string) (models.User, error) {
	return models.User{}, nil
}

func (f fakeAuthService) IssueSessionToken(userID string) (string, time.Time, error) {
	return f.issuedToken, f.issuedExp, nil
}

func (f fakeAuthService) IssueSessionPair(userID string) (string, time.Time, string, time.Time, error) {
	return f.issuedToken, f.issuedExp, f.issuedRefresh, f.issuedRefreshExp, nil
}

func (f fakeAuthService) RefreshSessionPair(refreshToken string) (string, string, time.Time, string, time.Time, error) {
	if f.refreshErr != nil {
		return "", "", time.Time{}, "", time.Time{}, f.refreshErr
	}
	if refreshToken != f.refreshToken {
		return "", "", time.Time{}, "", time.Time{}, services.NewAPIError(401, "unauthorized", "Invalid refresh token.", nil)
	}
	return f.refreshUser, f.issuedToken, f.issuedExp, f.refreshNextToken, f.refreshNextExp, nil
}

func (f fakeAuthService) RevokeRefreshToken(refreshToken string) error {
	return f.revokeErr
}

func (f fakeAuthService) VerifySessionToken(token string) (string, error) {
	if f.verifyErr != nil {
		return "", f.verifyErr
	}
	if token != f.verifyToken {
		return "", services.NewAPIError(401, "unauthorized", "Unauthorized access. Session required.", nil)
	}
	return f.verifyUser, nil
}

func (f fakeAuthService) ResetPasswordWithRecovery(email, recoveryKey, newPassword string) (string, error) {
	return "", nil
}

func (f fakeAuthService) AdditionalRegistrationOpen() (bool, error) {
	return false, nil
}

func TestRefreshV2IssuesNewBearerToken(t *testing.T) {
	exp := time.Date(2026, 5, 22, 10, 30, 0, 0, time.UTC)
	refreshExp := time.Date(2026, 6, 22, 10, 30, 0, 0, time.UTC)
	auth := fakeAuthService{
		refreshToken:     "old-refresh",
		refreshUser:      "user-1",
		issuedToken:      "new-access-token",
		issuedExp:        exp,
		refreshNextToken: "new-refresh-token",
		refreshNextExp:   refreshExp,
	}
	app := fiber.New()
	app.Post("/api/v2/auth/refresh", NewAuthHandlers(auth, config.Config{}).RefreshV2)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/auth/refresh", strings.NewReader(`{"refresh_token":"old-refresh"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	got := string(body)
	for _, want := range []string{
		`"success":true`,
		`"user_id":"user-1"`,
		`"token_type":"Bearer"`,
		`"access_token":"new-access-token"`,
		`"expires_at":"2026-05-22T10:30:00Z"`,
		`"refresh_token":"new-refresh-token"`,
		`"refresh_expires_at":"2026-06-22T10:30:00Z"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response %s does not contain %s", got, want)
		}
	}
}

func TestRefreshV2RequiresRefreshToken(t *testing.T) {
	app := fiber.New()
	app.Post("/api/v2/auth/refresh", NewAuthHandlers(fakeAuthService{}, config.Config{}).RefreshV2)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/auth/refresh", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestRefreshV2RejectsInvalidToken(t *testing.T) {
	auth := fakeAuthService{
		refreshErr: services.NewAPIError(401, "unauthorized", "Refresh token has expired.", errors.New("expired")),
	}
	app := fiber.New()
	app.Post("/api/v2/auth/refresh", NewAuthHandlers(auth, config.Config{}).RefreshV2)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/auth/refresh", strings.NewReader(`{"refresh_token":"expired-token"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
