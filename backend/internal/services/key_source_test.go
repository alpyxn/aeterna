package services

import (
	"encoding/base64"
	"testing"
)

func TestValidateKeyFormat(t *testing.T) {
	validBytes := make([]byte, 32)
	valid := base64.StdEncoding.EncodeToString(validBytes)

	if _, err := ValidateKeyFormat(valid); err != nil {
		t.Fatalf("expected valid key, got error: %v", err)
	}

	if _, err := ValidateKeyFormat(""); err == nil {
		t.Fatalf("expected empty key error")
	}

	if _, err := ValidateKeyFormat("not-base64"); err == nil {
		t.Fatalf("expected base64 error")
	}

	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := ValidateKeyFormat(short); err == nil {
		t.Fatalf("expected length error")
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey returned error: %v", err)
	}

	if key1 == key2 {
		t.Fatalf("expected distinct random keys")
	}

	decoded, err := base64.StdEncoding.DecodeString(key1)
	if err != nil {
		t.Fatalf("generated key should be valid base64: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected 32-byte key, got %d", len(decoded))
	}
}
