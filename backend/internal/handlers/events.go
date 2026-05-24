package handlers

import (
	"bufio"
	"encoding/json"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	defaultSSEHeartbeatInterval = 20 * time.Second
)

// EventsHandlers exposes user-scoped SSE streams.
type EventsHandlers struct {
	stream ports.EventStreamPort
}

func NewEventsHandlers(stream ports.EventStreamPort) *EventsHandlers {
	return &EventsHandlers{stream: stream}
}

func (h *EventsHandlers) Stream(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	sessionKey := currentSessionKey(c)
	clientID := strings.TrimSpace(c.Query("client_id"))
	if clientID == "" {
		clientID = uuid.NewString()
	}

	ch, done, cancel, err := h.stream.Subscribe(userID, clientID, sessionKey)
	if err != nil {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error": err.Error(),
			"code":  "sse_limit_exceeded",
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer cancel()

		_ = writeSSEEvent(w, ports.RealtimeEvent{
			Type:   ports.EventTypeReady,
			Code:   ports.EventCodeStreamReady,
			At:     time.Now().UTC(),
			Data:   map[string]string{"reason": "connected"},
			Reason: "connected",
		})

		heartbeat := time.NewTicker(defaultSSEHeartbeatInterval)
		defer heartbeat.Stop()

		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				if err := writeSSEEvent(w, event); err != nil {
					return
				}
			case <-done:
				return
			case <-heartbeat.C:
				if err := writeSSEEvent(w, ports.RealtimeEvent{
					Type: ports.EventTypePing,
					Code: ports.EventCodeStreamPing,
					At:   time.Now().UTC(),
				}); err != nil {
					return
				}
			}
		}
	})

	return nil
}

func writeSSEEvent(w *bufio.Writer, event ports.RealtimeEvent) error {
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := w.WriteString("event: " + event.Type + "\n"); err != nil {
		return err
	}
	if _, err := w.WriteString("data: " + string(payload) + "\n\n"); err != nil {
		return err
	}
	return w.Flush()
}
