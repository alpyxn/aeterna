package services

import (
	"strings"
	"testing"
)

func TestMarkdownToHTML_StripsRawHTML(t *testing.T) {
	input := "Hello <img src=x onerror=alert(1)> <script>alert('x')</script> world"
	out := markdownToHTML(input)

	if strings.Contains(out, "<img") {
		t.Fatalf("expected raw img tag to be stripped, got: %s", out)
	}
	if strings.Contains(out, "<script") {
		t.Fatalf("expected raw script tag to be stripped, got: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "onerror") {
		t.Fatalf("expected inline handlers to be stripped, got: %s", out)
	}
}

func TestMarkdownToHTML_BlocksUnsafeLinks(t *testing.T) {
	input := "[click](javascript:alert(1))"
	out := markdownToHTML(input)

	if strings.Contains(strings.ToLower(out), "javascript:") {
		t.Fatalf("expected javascript scheme to be blocked, got: %s", out)
	}
}

func TestMarkdownToHTML_AllowsSafeLinkWithRelProtection(t *testing.T) {
	input := "[site](https://example.com)"
	out := markdownToHTML(input)

	if !strings.Contains(out, "https://example.com") {
		t.Fatalf("expected https link to be preserved, got: %s", out)
	}
	if !strings.Contains(out, "noopener") || !strings.Contains(out, "noreferrer") || !strings.Contains(out, "nofollow") {
		t.Fatalf("expected rel protection attributes, got: %s", out)
	}
}
