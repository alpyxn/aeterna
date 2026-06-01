package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/gofiber/fiber/v2"
)

type fakeEventStream struct {
	subscribeCalls int
	lastUserID     string
	lastClientID   string
	lastSessionKey string
}

func (f *fakeEventStream) Subscribe(userID, clientID, sessionKey string) (<-chan ports.RealtimeEvent, <-chan struct{}, func(), error) {
	f.subscribeCalls++
	f.lastUserID = userID
	f.lastClientID = clientID
	f.lastSessionKey = sessionKey

	ch := make(chan ports.RealtimeEvent)
	done := make(chan struct{})
	close(done)
	cancel := func() {}
	return ch, done, cancel, nil
}

func (f *fakeEventStream) Publish(userID string, event ports.RealtimeEvent) {}

func TestEventsStreamRejectsInvalidClientID(t *testing.T) {
	stream := &fakeEventStream{}
	handler := NewEventsHandlers(stream)

	app := fiber.New()
	app.Get("/api/events", func(c *fiber.Ctx) error {
		c.Locals("user_id", "u1")
		return handler.Stream(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/events?client_id=bad%20id", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	if stream.subscribeCalls != 0 {
		t.Fatalf("expected subscribe not to be called for invalid client_id, got %d calls", stream.subscribeCalls)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	got := string(body)
	if !strings.Contains(got, `"code":"bad_request"`) {
		t.Fatalf("expected bad_request code, got %s", got)
	}
}

func TestEventsStreamAcceptsValidatedClientID(t *testing.T) {
	stream := &fakeEventStream{}
	handler := NewEventsHandlers(stream)

	app := fiber.New()
	app.Get("/api/events", func(c *fiber.Ctx) error {
		c.Locals("user_id", "u1")
		c.Locals("session_key", "sess-a")
		return handler.Stream(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/events?client_id=web-client_1:tab-2", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if stream.subscribeCalls != 1 {
		t.Fatalf("expected exactly one subscribe call, got %d", stream.subscribeCalls)
	}
	if stream.lastUserID != "u1" {
		t.Fatalf("expected userID u1, got %q", stream.lastUserID)
	}
	if stream.lastClientID != "web-client_1:tab-2" {
		t.Fatalf("expected clientID web-client_1:tab-2, got %q", stream.lastClientID)
	}
}
