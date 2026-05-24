package config

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alpyxn/aeterna/backend/internal/config/common"
	"github.com/alpyxn/aeterna/backend/internal/config/services"
)

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"development", false},
		{"", false},
		{"staging", false},
		{"PRODUCTION", false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("env=%q", tc.env), func(t *testing.T) {
			cfg := Config{App: services.AppSection{Env: tc.env}}
			if got := cfg.IsProduction(); got != tc.want {
				t.Fatalf("IsProduction() = %v, want %v (env=%q)", got, tc.want, tc.env)
			}
		})
	}
}

func TestConfig_AllowedOriginsOrDefault(t *testing.T) {
	t.Run("empty AllowedOrigins uses default", func(t *testing.T) {
		cfg := Config{HTTP: services.HTTPSection{AllowedOrigins: ""}}
		got := cfg.AllowedOriginsOrDefault()
		if got != common.DefaultAllowedOrigins {
			t.Fatalf("got %q, want default %q", got, common.DefaultAllowedOrigins)
		}
	})

	t.Run("non-empty AllowedOrigins returns configured value", func(t *testing.T) {
		cfg := Config{HTTP: services.HTTPSection{AllowedOrigins: "https://example.com"}}
		got := cfg.AllowedOriginsOrDefault()
		if got != "https://example.com" {
			t.Fatalf("got %q, want %q", got, "https://example.com")
		}
	})

	t.Run("multiple origins returned as-is", func(t *testing.T) {
		origins := "https://app.example.com,https://admin.example.com"
		cfg := Config{HTTP: services.HTTPSection{AllowedOrigins: origins}}
		got := cfg.AllowedOriginsOrDefault()
		if got != origins {
			t.Fatalf("got %q, want %q", got, origins)
		}
	})
}

func mustPanic(t *testing.T, fn func()) string {
	t.Helper()
	var msg string
	func() {
		defer func() {
			if r := recover(); r != nil {
				msg = fmt.Sprintf("%v", r)
			}
		}()
		fn()
	}()
	return msg
}

func TestLoad_DevelopmentMode(t *testing.T) {
	t.Setenv("ENV", "")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("PROXY_MODE", "")
	t.Setenv("AUTH_SESSION_TTL_HOURS", "")
	t.Setenv("AUTH_REFRESH_TTL_HOURS", "")
	t.Setenv("BASE_URL", "")

	cfg := Load()

	if cfg.IsProduction() {
		t.Fatal("expected non-production config")
	}
	if cfg.Database.Path != common.DefaultDatabasePath {
		t.Fatalf("Database.Path = %q, want %q", cfg.Database.Path, common.DefaultDatabasePath)
	}
	if cfg.Auth.SessionTTLHours != common.DefaultSessionTTLHours {
		t.Fatalf("Auth.SessionTTLHours = %d, want %d", cfg.Auth.SessionTTLHours, common.DefaultSessionTTLHours)
	}
	if cfg.Auth.RefreshTTLHours != common.DefaultRefreshTTLHours {
		t.Fatalf("Auth.RefreshTTLHours = %d, want %d", cfg.Auth.RefreshTTLHours, common.DefaultRefreshTTLHours)
	}
	if cfg.Worker.BaseURL != common.DefaultWorkerBaseURL {
		t.Fatalf("Worker.BaseURL = %q, want %q", cfg.Worker.BaseURL, common.DefaultWorkerBaseURL)
	}
}

func TestLoad_ProductionModePanicsWithoutRequiredVars(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("DATABASE_PATH", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	msg := mustPanic(t, func() { Load() })
	if msg == "" {
		t.Fatal("expected a panic but Load returned normally")
	}
	if !strings.Contains(msg, "config validation failed") {
		t.Fatalf("unexpected panic message: %q", msg)
	}
}

func TestLoad_ProductionModeWithAllRequiredVars(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("DATABASE_PATH", "/prod/aeterna.db")
	t.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("PROXY_MODE", "")

	cfg := Load()

	if !cfg.IsProduction() {
		t.Fatal("expected production config")
	}
	if cfg.Database.Path != "/prod/aeterna.db" {
		t.Fatalf("Database.Path = %q, want %q", cfg.Database.Path, "/prod/aeterna.db")
	}
	if cfg.HTTP.AllowedOrigins != "https://app.example.com" {
		t.Fatalf("HTTP.AllowedOrigins = %q, want %q", cfg.HTTP.AllowedOrigins, "https://app.example.com")
	}
}
