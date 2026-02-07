package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type CreateMessageRequest struct {
	Content         string `json:"content"`
	RecipientEmail  string `json:"recipient_email"`
	TriggerDuration int    `json:"trigger_duration"` // in minutes
}

type HeartbeatRequest struct {
	ID string `json:"id"`
}

var messageService = services.MessageService{}

func CreateMessage(c *fiber.Ctx) error {
	req := new(CreateMessageRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	msg, err := messageService.Create(req.Content, req.RecipientEmail, req.TriggerDuration)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"id":      msg.ID,
		"message": "Dead man's switch activated!",
	})
}

func GetMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	msg, err := messageService.GetByID(id)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"content":    msg.Content,
		"status":     msg.Status,
		"created_at": msg.CreatedAt,
	})
}

func Heartbeat(c *fiber.Ctx) error {
	req := new(HeartbeatRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	msg, err := messageService.Heartbeat(req.ID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{"status": "alive", "last_seen": msg.LastSeen})
}

func ListMessages(c *fiber.Ctx) error {
	messages, err := messageService.List()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(messages)
}

func DeleteMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := messageService.Delete(id); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Message deleted successfully"})
}

type UpdateMessageRequest struct {
	Content         string `json:"content"`
	TriggerDuration int    `json:"trigger_duration"`
}

func UpdateMessage(c *fiber.Ctx) error {
	id := c.Params("id")
	req := new(UpdateMessageRequest)
	if err := c.BodyParser(req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	msg, err := messageService.Update(id, req.Content, req.TriggerDuration)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": msg,
	})
}
