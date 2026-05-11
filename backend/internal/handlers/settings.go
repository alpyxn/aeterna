package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// settingsResponse embeds tenant settings and adds global registration flags.
type settingsResponse struct {
	models.Settings
	AllowRegistration     bool `json:"allow_registration"`
	CanManageRegistration bool `json:"can_manage_registration"`
	TelegramBotConfigured bool `json:"telegram_bot_configured"`
}

// SettingsHandlers groups SMTP settings and application configuration handlers.
type SettingsHandlers struct {
	settings    ports.SettingsServicePort
	appSettings ports.ApplicationSettingsServicePort
}

func NewSettingsHandlers(settings ports.SettingsServicePort, appSettings ports.ApplicationSettingsServicePort) *SettingsHandlers {
	return &SettingsHandlers{settings: settings, appSettings: appSettings}
}

func (h *SettingsHandlers) Get(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	settings, err := h.settings.Get(userID)
	if err != nil {
		return writeError(c, err)
	}
	app, err := h.appSettings.Get()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(settingsResponse{
		Settings:              settings,
		AllowRegistration:     app.AllowRegistration,
		CanManageRegistration: h.appSettings.CanManageRegistration(userID),
		TelegramBotConfigured: settings.TelegramBotToken != "",
	})
}

func (h *SettingsHandlers) Save(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req models.SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if req.AllowRegistration != nil {
		if err := h.appSettings.SetAllowRegistration(userID, *req.AllowRegistration); err != nil {
			return writeError(c, err)
		}
	}
	if err := h.settings.Save(userID, req.ToSettings()); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *SettingsHandlers) TestSMTP(c *fiber.Ctx) error {
	if _, err := currentUserID(c); err != nil {
		return writeError(c, err)
	}
	var req models.SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := h.settings.TestSMTP(req.ToSettings()); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Connection successful"})
}

func (h *SettingsHandlers) TestTelegram(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req models.SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := h.settings.TestTelegram(userID, req.ToSettings()); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Telegram message sent"})
}
