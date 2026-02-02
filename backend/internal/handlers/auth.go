package handlers

import (
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type passwordRequest struct {
	Password string `json:"password"`
}

var authService = services.AuthService{}

func SetupStatus(c *fiber.Ctx) error {
	configured, err := authService.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"configured": configured})
}

func SetupMasterPassword(c *fiber.Ctx) error {
	configured, err := authService.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	if configured {
		return writeError(c, services.NewAPIError(400, "already_configured", "Master password already configured", nil))
	}

	var req passwordRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := authService.SetMasterPassword(req.Password); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func VerifyMasterPassword(c *fiber.Ctx) error {
	var req passwordRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if err := authService.VerifyMasterPassword(req.Password); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}
