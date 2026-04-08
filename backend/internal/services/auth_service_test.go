package services

import (
	"os"
	"testing"
	"time"
)

func TestSessionTTLDefault(t *testing.T) {
	old := os.Getenv("AUTH_SESSION_TTL_HOURS")
	defer func() {
		if old == "" {
			_ = os.Unsetenv("AUTH_SESSION_TTL_HOURS")
			return
		}
		_ = os.Setenv("AUTH_SESSION_TTL_HOURS", old)
	}()

	_ = os.Unsetenv("AUTH_SESSION_TTL_HOURS")
	if got := sessionTTL(); got != 12*time.Hour {
		t.Fatalf("expected default 12h, got %v", got)
	}
}

func TestSessionTTLFromEnv(t *testing.T) {
	old := os.Getenv("AUTH_SESSION_TTL_HOURS")
	defer func() {
		if old == "" {
			_ = os.Unsetenv("AUTH_SESSION_TTL_HOURS")
			return
		}
		_ = os.Setenv("AUTH_SESSION_TTL_HOURS", old)
	}()

	_ = os.Setenv("AUTH_SESSION_TTL_HOURS", "6")
	if got := sessionTTL(); got != 6*time.Hour {
		t.Fatalf("expected 6h, got %v", got)
	}

	_ = os.Setenv("AUTH_SESSION_TTL_HOURS", "not-a-number")
	if got := sessionTTL(); got != 12*time.Hour {
		t.Fatalf("expected fallback 12h for invalid value, got %v", got)
	}

	_ = os.Setenv("AUTH_SESSION_TTL_HOURS", "0")
	if got := sessionTTL(); got != 12*time.Hour {
		t.Fatalf("expected fallback 12h for non-positive value, got %v", got)
	}
}
