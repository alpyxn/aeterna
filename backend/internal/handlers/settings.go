package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var applicationSettingsService = services.ApplicationSettingsService{}

// settingsResponse embeds tenant settings and adds global registration flags.
type settingsResponse struct {
	models.Settings
	AllowRegistration     bool `json:"allow_registration"`
	CanManageRegistration bool `json:"can_manage_registration"`
}

func GetSettings(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	settings, err := settingsService.Get(userID)
	if err != nil {
		return writeError(c, err)
	}
	app, err := applicationSettingsService.Get()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(settingsResponse{
		Settings:              settings,
		AllowRegistration:     app.AllowRegistration,
		CanManageRegistration: services.IsFirstUser(userID),
	})
}

func SaveSettings(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	var req models.SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if req.AllowRegistration != nil && services.IsFirstUser(userID) {
		if err := applicationSettingsService.SetAllowRegistration(userID, *req.AllowRegistration); err != nil {
			return writeError(c, err)
		}
	}
	if err := settingsService.Save(userID, req.ToSettings()); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func TestSMTP(c *fiber.Ctx) error {
	if _, err := currentUserID(c); err != nil {
		return writeError(c, err)
	}
	var req models.SettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := settingsService.TestSMTP(req.ToSettings()); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Connection successful"})
}
