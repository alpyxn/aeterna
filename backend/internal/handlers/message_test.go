package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type fakeMessageService struct {
	heartbeatResult models.Message
	heartbeatErr    error
}

func (f fakeMessageService) Create(userID, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	return models.Message{}, nil
}

func (f fakeMessageService) GetPublicByID(id string) (models.Message, error) {
	return models.Message{}, nil
}

func (f fakeMessageService) GetByID(userID, id string) (models.Message, error) {
	return models.Message{}, nil
}

func (f fakeMessageService) List(userID string) ([]models.Message, error) {
	return nil, nil
}

func (f fakeMessageService) Heartbeat(userID, id string) (models.Message, error) {
	if f.heartbeatErr != nil {
		return models.Message{}, f.heartbeatErr
	}
	return f.heartbeatResult, nil
}

func (f fakeMessageService) BulkHeartbeat(userID string) error {
	return nil
}

func (f fakeMessageService) Delete(userID, id string) error {
	return nil
}

func (f fakeMessageService) Update(userID, id, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	return models.Message{}, nil
}

func TestHeartbeatReturnsComputedScheduleFields(t *testing.T) {
	lastSeen := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	nextTrigger := lastSeen.Add(90 * time.Minute)
	nextReminder := nextTrigger.Add(-30 * time.Minute)

	handler := NewMessageHandlers(fakeMessageService{
		heartbeatResult: models.Message{
			LastSeen:       lastSeen,
			NextTriggerAt:  &nextTrigger,
			NextReminderAt: &nextReminder,
		},
	})

	app := fiber.New()
	app.Post("/api/heartbeat", func(c *fiber.Ctx) error {
		c.Locals("user_id", "u-test")
		return handler.Heartbeat(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/heartbeat", strings.NewReader(`{"id":"m1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	got := string(body)
	for _, want := range []string{
		`"status":"alive"`,
		`"last_seen":"2026-05-29T12:00:00Z"`,
		`"next_trigger_at":"2026-05-29T13:30:00Z"`,
		`"next_reminder_at":"2026-05-29T13:00:00Z"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("response %s does not contain %s", got, want)
		}
	}
}

func TestHeartbeatReturnsNullReminderWhenNoPendingReminder(t *testing.T) {
	lastSeen := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	nextTrigger := lastSeen.Add(15 * time.Minute)

	handler := NewMessageHandlers(fakeMessageService{
		heartbeatResult: models.Message{
			LastSeen:       lastSeen,
			NextTriggerAt:  &nextTrigger,
			NextReminderAt: nil,
		},
	})

	app := fiber.New()
	app.Post("/api/heartbeat", func(c *fiber.Ctx) error {
		c.Locals("user_id", "u-test")
		return handler.Heartbeat(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/heartbeat", strings.NewReader(`{"id":"m1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	got := string(body)
	if !strings.Contains(got, `"next_reminder_at":null`) {
		t.Fatalf("expected null next_reminder_at in response, got %s", got)
	}
}

func TestHeartbeatReturnsUnauthorizedWithoutUserContext(t *testing.T) {
	handler := NewMessageHandlers(fakeMessageService{
		heartbeatErr: services.NewAPIError(401, "unauthorized", "Unauthorized", nil),
	})
	app := fiber.New()
	app.Post("/api/heartbeat", handler.Heartbeat)

	req := httptest.NewRequest(http.MethodPost, "/api/heartbeat", strings.NewReader(`{"id":"m1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
