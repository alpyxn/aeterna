package services

import "testing"

func TestAuthWithFallbackSkipsWhenNoUsername(t *testing.T) {
	// A nil client would panic if auth were attempted.
	if err := authWithFallback(nil, "", "", "smtp.lan"); err != nil {
		t.Fatalf("expected no-op for empty username, got %v", err)
	}
}
