package models

import "testing"

func TestMessageBeforeCreateGeneratesValues(t *testing.T) {
	msg := &Message{}
	if err := msg.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if msg.ID == "" {
		t.Fatalf("expected generated ID")
	}
	if msg.ManagementToken == "" {
		t.Fatalf("expected generated management token")
	}
}

func TestMessageBeforeCreatePreservesExistingValues(t *testing.T) {
	msg := &Message{ID: "existing-id", ManagementToken: "existing-token"}
	if err := msg.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if msg.ID != "existing-id" {
		t.Fatalf("expected existing ID to be preserved")
	}
	if msg.ManagementToken != "existing-token" {
		t.Fatalf("expected existing management token to be preserved")
	}
}
