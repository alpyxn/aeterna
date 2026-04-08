package handlers

import (
	"strings"

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

type HeartbeatRequest struct {
	ID string `json:"id"`
}

var messageService = services.MessageService{}

func CreateMessage(c *fiber.Ctx) error {
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

	msg, err := messageService.Create(userID, req.Content, recipients, req.TriggerDuration, req.Reminders)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"id":      msg.ID,
		"message": "Dead man's switch activated!",
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

// GetMessage is public: reveal content only when message is triggered (unchanged contract).
func GetMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	msg, err := messageService.GetPublicByID(id)
	if err != nil {
		return writeError(c, err)
	}

	content := ""
	if string(msg.Status) == "triggered" {
		content = msg.Content
	}

	return c.JSON(fiber.Map{
		"content":    content,
		"status":     msg.Status,
		"created_at": msg.CreatedAt,
	})
}

func Heartbeat(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	req := new(HeartbeatRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	msg, err := messageService.Heartbeat(userID, req.ID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{"status": "alive", "last_seen": msg.LastSeen})
}

func ListMessages(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	messages, err := messageService.List(userID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(messages)
}

func DeleteMessage(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	id := c.Params("id")
	if err := messageService.Delete(userID, id); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Message deleted successfully"})
}

type UpdateMessageRequest struct {
	Content         string   `json:"content"`
	RecipientEmail  string   `json:"recipient_email"`
	RecipientEmails []string `json:"recipient_emails"`
	TriggerDuration int      `json:"trigger_duration"`
	Reminders       []int    `json:"reminders"`
}

func UpdateMessage(c *fiber.Ctx) error {
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

	msg, err := messageService.Update(userID, id, req.Content, recipients, req.TriggerDuration, req.Reminders)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": msg,
	})
}
