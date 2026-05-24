package services

import (
	"testing"

	"github.com/alpyxn/aeterna/backend/internal/config/common"
)

func TestAuthModule_Metadata(t *testing.T) {
	m := AuthModule{}
	if got := m.Name(); got != "AuthModule" {
		t.Fatalf("Name() = %q, want %q", got, "AuthModule")
	}
	if got := m.Section(); got != "auth" {
		t.Fatalf("Section() = %q, want %q", got, "auth")
	}
}

func TestAuthModule_LoadAndValidate(t *testing.T) {
	t.Run("defaults when no env vars set", func(t *testing.T) {
		t.Setenv("AUTH_SESSION_TTL_HOURS", "")
		t.Setenv("AUTH_REFRESH_TTL_HOURS", "")
		t.Setenv("ALLOW_REGISTRATION", "")
		t.Setenv("MASTER_PASSWORD", "")
		t.Setenv("AUTH_COOKIE_SECURE_MODE", "")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.SessionTTLHours != common.DefaultSessionTTLHours {
			t.Fatalf("SessionTTLHours = %d, want %d", section.SessionTTLHours, common.DefaultSessionTTLHours)
		}
		if section.RefreshTTLHours != common.DefaultRefreshTTLHours {
			t.Fatalf("RefreshTTLHours = %d, want %d", section.RefreshTTLHours, common.DefaultRefreshTTLHours)
		}
		if section.AllowRegistration {
			t.Fatal("AllowRegistration should default to false")
		}
		if section.MasterPassword != "" {
			t.Fatalf("MasterPassword = %q, want empty", section.MasterPassword)
		}
		if section.CookieSecureMode != "" {
			t.Fatalf("CookieSecureMode = %q, want empty", section.CookieSecureMode)
		}
	})

	t.Run("custom session TTL", func(t *testing.T) {
		t.Setenv("AUTH_SESSION_TTL_HOURS", "48")
		t.Setenv("AUTH_REFRESH_TTL_HOURS", "1440")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.SessionTTLHours != 48 {
			t.Fatalf("SessionTTLHours = %d, want 48", section.SessionTTLHours)
		}
		if section.RefreshTTLHours != 1440 {
			t.Fatalf("RefreshTTLHours = %d, want 1440", section.RefreshTTLHours)
		}
	})

	t.Run("zero TTL falls back to default", func(t *testing.T) {
		t.Setenv("AUTH_SESSION_TTL_HOURS", "0")
		t.Setenv("AUTH_REFRESH_TTL_HOURS", "0")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.SessionTTLHours != common.DefaultSessionTTLHours {
			t.Fatalf("SessionTTLHours = %d, want default %d", section.SessionTTLHours, common.DefaultSessionTTLHours)
		}
		if section.RefreshTTLHours != common.DefaultRefreshTTLHours {
			t.Fatalf("RefreshTTLHours = %d, want default %d", section.RefreshTTLHours, common.DefaultRefreshTTLHours)
		}
	})

	t.Run("negative TTL falls back to default", func(t *testing.T) {
		t.Setenv("AUTH_SESSION_TTL_HOURS", "-1")
		t.Setenv("AUTH_REFRESH_TTL_HOURS", "-1")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.SessionTTLHours != common.DefaultSessionTTLHours {
			t.Fatalf("SessionTTLHours = %d, want default %d", section.SessionTTLHours, common.DefaultSessionTTLHours)
		}
		if section.RefreshTTLHours != common.DefaultRefreshTTLHours {
			t.Fatalf("RefreshTTLHours = %d, want default %d", section.RefreshTTLHours, common.DefaultRefreshTTLHours)
		}
	})

	t.Run("invalid TTL string falls back to default", func(t *testing.T) {
		t.Setenv("AUTH_SESSION_TTL_HOURS", "not-a-number")
		t.Setenv("AUTH_REFRESH_TTL_HOURS", "not-a-number")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.SessionTTLHours != common.DefaultSessionTTLHours {
			t.Fatalf("SessionTTLHours = %d, want default %d", section.SessionTTLHours, common.DefaultSessionTTLHours)
		}
		if section.RefreshTTLHours != common.DefaultRefreshTTLHours {
			t.Fatalf("RefreshTTLHours = %d, want default %d", section.RefreshTTLHours, common.DefaultRefreshTTLHours)
		}
	})

	t.Run("ALLOW_REGISTRATION true enables registration", func(t *testing.T) {
		t.Setenv("ALLOW_REGISTRATION", "true")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !section.AllowRegistration {
			t.Fatal("AllowRegistration should be true")
		}
	})

	t.Run("ALLOW_REGISTRATION non-true value disables registration", func(t *testing.T) {
		t.Setenv("ALLOW_REGISTRATION", "1")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.AllowRegistration {
			t.Fatal("AllowRegistration should be false for value '1' (only 'true' is accepted)")
		}
	})

	t.Run("MASTER_PASSWORD is captured as-is", func(t *testing.T) {
		t.Setenv("MASTER_PASSWORD", "s3cr3tP@ss!")
		section, err := AuthModule{}.LoadAndValidate()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.MasterPassword != "s3cr3tP@ss!" {
			t.Fatalf("MasterPassword = %q, want %q", section.MasterPassword, "s3cr3tP@ss!")
		}
	})

	cookieModeTests := []struct {
		name     string
		input    string
		wantMode string
	}{
		{"always mode", "always", "always"},
		{"never mode", "never", "never"},
		{"always uppercase normalized", "ALWAYS", "always"},
		{"never uppercase normalized", "NEVER", "never"},
		{"invalid mode cleared", "sometimes", ""},
		{"empty mode cleared", "", ""},
	}
	for _, tc := range cookieModeTests {
		tc := tc
		t.Run("cookie_mode_"+tc.name, func(t *testing.T) {
			t.Setenv("AUTH_COOKIE_SECURE_MODE", tc.input)
			section, err := AuthModule{}.LoadAndValidate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if section.CookieSecureMode != tc.wantMode {
				t.Fatalf("CookieSecureMode = %q, want %q", section.CookieSecureMode, tc.wantMode)
			}
		})
	}
}
