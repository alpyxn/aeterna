package middleware

import (
	"errors"
	"os"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var authService = services.AuthService{}

func MasterAuth(c *fiber.Ctx) error {
	masterKey := os.Getenv("MASTER_PASSWORD")
	clientKey := c.Get("X-Master-Key")
	
	// If no env password set, fall back to DB-stored master password.
	if masterKey == "" {
		if err := authService.VerifyMasterPassword(clientKey); err != nil {
			var apiErr *services.APIError
			if errors.As(err, &apiErr) {
				return c.Status(apiErr.Status).JSON(fiber.Map{
					"error": apiErr.Message,
					"code":  apiErr.Code,
				})
			}
			return c.Status(401).JSON(fiber.Map{
				"error": "Unauthorized access. Master key required.",
				"code":  "unauthorized",
			})
		}
		return c.Next()
	}

	if clientKey != masterKey {
		return c.Status(401).JSON(fiber.Map{
			"error": "Unauthorized access. Master key required.",
			"code":  "unauthorized",
		})
	}

	return c.Next()
}

