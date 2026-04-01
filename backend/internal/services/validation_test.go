package services

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	svc := ValidationService{}

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{name: "valid email", email: "user@example.com", wantErr: false},
		{name: "trimmed valid email", email: "  user@example.com  ", wantErr: false},
		{name: "empty email", email: "", wantErr: true},
		{name: "invalid format", email: "not-an-email", wantErr: true},
		{name: "dangerous javascript pattern", email: "javascript:alert@x.com", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.ValidateEmail(tc.email)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	svc := ValidationService{}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid password", password: "StrongP@ss1", wantErr: false},
		{name: "too short", password: "Aa1!a", wantErr: true},
		{name: "no uppercase", password: "strongp@ss1", wantErr: true},
		{name: "no lowercase", password: "STRONGP@SS1", wantErr: true},
		{name: "no number", password: "StrongP@ss", wantErr: true},
		{name: "no special", password: "StrongPass1", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.ValidatePassword(tc.password)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	svc := ValidationService{}

	tests := []struct {
		name     string
		input    string
		contains string
		equals   string
	}{
		{name: "strips path traversal", input: "../../secret.txt", equals: "secret.txt"},
		{name: "removes leading dots", input: "...hidden.txt", equals: "hidden.txt"},
		{name: "fallback unnamed", input: "\x00\x01", equals: "unnamed_file"},
		{name: "preserves extension", input: "safe.pdf", equals: "safe.pdf"},
		{name: "separator replacement", input: "a/b\\c.txt", contains: "_"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := svc.SanitizeFilename(tc.input)
			if tc.equals != "" && got != tc.equals {
				t.Fatalf("expected %q, got %q", tc.equals, got)
			}
			if tc.contains != "" && !strings.Contains(got, tc.contains) {
				t.Fatalf("expected sanitized filename to contain %q, got %q", tc.contains, got)
			}
		})
	}
}

func TestValidateTriggerDuration(t *testing.T) {
	svc := ValidationService{}

	tests := []struct {
		name     string
		duration int
		wantErr  bool
	}{
		{name: "minimum valid", duration: 1, wantErr: false},
		{name: "maximum valid", duration: 525600, wantErr: false},
		{name: "too low", duration: 0, wantErr: true},
		{name: "too high", duration: 525601, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.ValidateTriggerDuration(tc.duration)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateFile(t *testing.T) {
	svc := ValidationService{}

	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	textBody := []byte("hello world")

	tests := []struct {
		name     string
		filename string
		size     int64
		data     []byte
		wantErr  bool
	}{
		{name: "valid png", filename: "image.png", size: int64(len(pngHeader)), data: pngHeader, wantErr: false},
		{name: "valid text", filename: "note.txt", size: int64(len(textBody)), data: textBody, wantErr: false},
		{name: "empty file", filename: "note.txt", size: 0, data: []byte{}, wantErr: true},
		{name: "missing extension", filename: "note", size: int64(len(textBody)), data: textBody, wantErr: true},
		{name: "disallowed extension", filename: "malware.exe", size: int64(len(textBody)), data: textBody, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := svc.ValidateFile(tc.filename, tc.size, tc.data)
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}

func TestValidateContent(t *testing.T) {
	svc := ValidationService{}

	if err := svc.ValidateContent(""); err == nil {
		t.Fatalf("expected empty content error")
	}

	if err := svc.ValidateContent("ok"); err != nil {
		t.Fatalf("expected valid content, got %v", err)
	}

	tooLong := strings.Repeat("a", MaxContentLength+1)
	if err := svc.ValidateContent(tooLong); err == nil {
		t.Fatalf("expected max length error")
	}
}

func TestSanitizeContent(t *testing.T) {
	svc := ValidationService{}
	in := "<script>alert('x')</script>"
	out := svc.SanitizeContent(in)
	if out == in {
		t.Fatalf("expected html-escaped output, got unchanged content")
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped tag, got %q", out)
	}
}

func TestValidateEmailReturnsAPIError(t *testing.T) {
	svc := ValidationService{}
	err := svc.ValidateEmail("invalid")
	if err == nil {
		t.Fatalf("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != 400 {
		t.Fatalf("expected status 400, got %d", apiErr.Status)
	}
}
