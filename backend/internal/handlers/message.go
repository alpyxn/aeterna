package handlers

import (
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type CreateMessageRequest struct {
	Content         string   `json:"content"`
	RecipientEmail  string   `json:"recipient_email"`
	RecipientEmails []string `json:"recipient_emails"`
	TriggerDuration int      `json:"trigger_duration"`
	Reminders       []int    `json:"reminders"`
}

type UpdateMessageRequest struct {
	Content         string   `json:"content"`
	RecipientEmail  string   `json:"recipient_email"`
	RecipientEmails []string `json:"recipient_emails"`
	TriggerDuration int      `json:"trigger_duration"`
	Reminders       []int    `json:"reminders"`
}

// MessageHandlers groups all switch message route handlers.
type MessageHandlers struct {
	messages ports.MessageServicePort
}

func NewMessageHandlers(messages ports.MessageServicePort) *MessageHandlers {
	return &MessageHandlers{messages: messages}
}

func (h *MessageHandlers) Create(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	req := new(CreateMessageRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	recipients := normalizeRecipients(req.RecipientEmails)
	if len(recipients) == 0 && strings.TrimSpace(req.RecipientEmail) != "" {
		recipients = []string{strings.TrimSpace(req.RecipientEmail)}
	}

	msg, err := h.messages.Create(userID, req.Content, recipients, req.TriggerDuration, req.Reminders)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"id":      msg.ID,
		"message": "Dead man's switch activated!",
	})
}

// GetPublic reveals content only when the message is triggered (unauthenticated endpoint).
func (h *MessageHandlers) GetPublic(c *fiber.Ctx) error {
	id := c.Params("id")
	msg, err := h.messages.GetPublicByID(id)
	if err != nil {
		return writeError(c, err)
	}

	content := ""
	if msg.Status == models.StatusTriggered {
		content = msg.Content
	}

	return c.JSON(fiber.Map{
		"content":    content,
		"status":     msg.Status,
		"created_at": msg.CreatedAt,
	})
}

func (h *MessageHandlers) Heartbeat(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	req := new(struct {
		ID string `json:"id"`
	})
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	msg, err := h.messages.Heartbeat(userID, req.ID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"status":           "alive",
		"last_seen":        msg.LastSeen,
		"next_trigger_at":  msg.NextTriggerAt,
		"next_reminder_at": msg.NextReminderAt,
	})
}

func (h *MessageHandlers) List(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	messages, err := h.messages.List(userID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(messages)
}

func (h *MessageHandlers) Delete(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	id := c.Params("id")
	if err := h.messages.Delete(userID, id); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Message deleted successfully"})
}

func (h *MessageHandlers) Update(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	id := c.Params("id")
	req := new(UpdateMessageRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	recipients := normalizeRecipients(req.RecipientEmails)
	if len(recipients) == 0 && strings.TrimSpace(req.RecipientEmail) != "" {
		recipients = []string{strings.TrimSpace(req.RecipientEmail)}
	}

	msg, err := h.messages.Update(userID, id, req.Content, recipients, req.TriggerDuration, req.Reminders)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": msg,
	})
}

func normalizeRecipients(recipients []string) []string {
	if len(recipients) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(recipients))
	normalized := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		email := strings.TrimSpace(recipient)
		if email == "" {
			continue
		}
		key := strings.ToLower(email)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, email)
	}

	if len(normalized) == 0 {
		return nil
	}

	return normalized
}
