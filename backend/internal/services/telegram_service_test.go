package services

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alpyxn/aeterna/backend/internal/models"
)

func TestTelegramServiceSendTestHeartbeat(t *testing.T) {
	var gotPath string
	var gotBody telegramSendMessageRequest
	var decodeErr error

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		decodeErr = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	t.Setenv("TELEGRAM_API_BASE", server.URL)
	t.Setenv("BASE_URL", "https://dead.example")

	settings := models.Settings{
		TelegramBotToken: "bot-token",
		TelegramChatID:   "123456",
		HeartbeatToken:   "heartbeat-token",
	}

	if err := (TelegramService{}).SendTestHeartbeat(settings); err != nil {
		t.Fatalf("SendTestHeartbeat returned error: %v", err)
	}
	if decodeErr != nil {
		t.Fatalf("decode request: %v", decodeErr)
	}

	if gotPath != "/botbot-token/sendMessage" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotBody.ChatID != "123456" {
		t.Fatalf("expected chat_id 123456, got %q", gotBody.ChatID)
	}
	if gotBody.ReplyMarkup == nil || len(gotBody.ReplyMarkup.InlineKeyboard) != 1 || len(gotBody.ReplyMarkup.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected a single inline button, got %#v", gotBody.ReplyMarkup)
	}
	if gotBody.ReplyMarkup.InlineKeyboard[0][0].URL != "https://dead.example/api/quick-heartbeat/heartbeat-token?auto=1" {
		t.Fatalf("unexpected button URL: %s", gotBody.ReplyMarkup.InlineKeyboard[0][0].URL)
	}
}

func TestTelegramServiceRejectsAPILevelFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":false,"description":"chat not found"}`))
	}))
	defer server.Close()

	t.Setenv("TELEGRAM_API_BASE", server.URL)
	t.Setenv("BASE_URL", "https://dead.example")

	settings := models.Settings{
		TelegramBotToken: "bot-token",
		TelegramChatID:   "123456",
		HeartbeatToken:   "heartbeat-token",
	}

	err := (TelegramService{}).SendTestHeartbeat(settings)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if err.Error() != "Telegram API rejected the request: chat not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}
