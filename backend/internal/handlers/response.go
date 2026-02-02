package handlers

import (
	"errors"
	"os"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

func writeError(c *fiber.Ctx, err error) error {
	var apiErr *services.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.Code
		if code == "" {
			code = "internal_error"
		}
		payload := fiber.Map{
			"error": apiErr.Message,
			"code":  code,
		}
		if os.Getenv("ENV") != "production" && apiErr.Err != nil {
			payload["detail"] = apiErr.Err.Error()
		}
		return c.Status(apiErr.Status).JSON(payload)
	}
	payload := fiber.Map{
		"error": "Internal server error",
		"code":  "internal_error",
	}
	if os.Getenv("ENV") != "production" && err != nil {
		payload["detail"] = err.Error()
	}
	return c.Status(500).JSON(payload)
}
