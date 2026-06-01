package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type webhookRequest struct {
	URL     string `json:"url"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// WebhookHandlers groups webhook CRUD route handlers.
type WebhookHandlers struct {
	webhooks ports.WebhookStorePort
}

func NewWebhookHandlers(webhooks ports.WebhookStorePort) *WebhookHandlers {
	return &WebhookHandlers{webhooks: webhooks}
}

func (h *WebhookHandlers) List(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.webhooks.List(userID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(items)
}

func (h *WebhookHandlers) Create(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	webhookStore := withOriginSession(c, h.webhooks)
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

func (h *WebhookHandlers) Update(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	webhookStore := withOriginSession(c, h.webhooks)
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

func (h *WebhookHandlers) Delete(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	webhookStore := withOriginSession(c, h.webhooks)
	id := c.Params("id")
	if err := webhookStore.Delete(userID, id); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}
