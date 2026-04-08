package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type webhookRequest struct {
	URL     string `json:"url"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

var webhookStore = services.WebhookStore{}

func ListWebhooks(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	items, err := webhookStore.List(userID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(items)
}

func CreateWebhook(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req webhookRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	item := models.Webhook{
		URL:     req.URL,
		Secret:  req.Secret,
		Enabled: req.Enabled,
	}
	created, err := webhookStore.Create(userID, item)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(created)
}

func UpdateWebhook(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	id := c.Params("id")
	var req webhookRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	item := models.Webhook{
		URL:     req.URL,
		Secret:  req.Secret,
		Enabled: req.Enabled,
	}
	updated, err := webhookStore.Update(userID, id, item)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(updated)
}

func DeleteWebhook(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	id := c.Params("id")
	if err := webhookStore.Delete(userID, id); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}
