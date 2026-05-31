package services

import (
	"strings"
	"testing"
)

func TestSanitizeFarewellMarkdown_RemovesRawHTMLAndUnsafeLinks(t *testing.T) {
	input := "Hi <script>alert('x')</script> [link](javascript:alert(1))"
	out := sanitizeFarewellMarkdown(input)

	if strings.Contains(strings.ToLower(out), "<script") {
		t.Fatalf("expected raw html to be removed, got: %s", out)
	}
	if strings.Contains(strings.ToLower(out), "javascript:") {
		t.Fatalf("expected javascript scheme to be removed, got: %s", out)
	}
}

func TestSanitizeFarewellMarkdown_PreservesAutolinks(t *testing.T) {
	input := "Read <https://example.com> and <mailto:test@example.com>"
	out := sanitizeFarewellMarkdown(input)

	if !strings.Contains(out, "<https://example.com>") {
		t.Fatalf("expected https autolink to be preserved, got: %s", out)
	}
	if !strings.Contains(out, "<mailto:test@example.com>") {
		t.Fatalf("expected mailto autolink to be preserved, got: %s", out)
	}
}

func TestCountWordsFromMarkdown_UsesVisibleText(t *testing.T) {
	input := "# Title\n\n- first item\n- second item\n\nVisit [Aeterna](https://aeterna.example)"
	got := countWordsFromMarkdown(input)

	if got != 7 {
		t.Fatalf("expected 7 words, got %d", got)
	}
}
