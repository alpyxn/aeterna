package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// settingsService is defined in heartbeat.go


func GetSettings(c *fiber.Ctx) error {
	settings, err := settingsService.Get()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(settings)
}

func SaveSettings(c *fiber.Ctx) error {
	var req models.Settings
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := settingsService.Save(req); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func TestSMTP(c *fiber.Ctx) error {
	var req models.Settings
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := settingsService.TestSMTP(req); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Connection successful"})
}
