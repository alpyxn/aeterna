package handlers

import (
	"errors"

	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

const productionModeLocalKey = "is_production"

// AttachRuntimeFlags stores runtime-only flags in request locals for handler helpers.
func AttachRuntimeFlags(isProduction bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals(productionModeLocalKey, isProduction)
		return c.Next()
	}
}

func currentUserID(c *fiber.Ctx) (string, error) {
	uid, ok := c.Locals(middleware.LocalUserIDKey).(string)
	if !ok || uid == "" {
		return "", services.NewAPIError(401, "unauthorized", "Unauthorized", nil)
	}
	return uid, nil
}

func currentSessionKey(c *fiber.Ctx) string {
	sessionKey, _ := c.Locals(middleware.LocalSessionKey).(string)
	return sessionKey
}

type originScopedService[T any] interface {
	WithOriginSession(sessionKey string) T
}

func withOriginSession[T any](c *fiber.Ctx, svc T) T {
	if aware, ok := any(svc).(originScopedService[T]); ok {
		return aware.WithOriginSession(currentSessionKey(c))
	}
	return svc
}

func writeError(c *fiber.Ctx, err error) error {
	isProd, _ := c.Locals(productionModeLocalKey).(bool)
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
		if !isProd && apiErr.Err != nil {
			payload["detail"] = apiErr.Err.Error()
		}
		return c.Status(apiErr.Status).JSON(payload)
	}
	payload := fiber.Map{
		"error": "Internal server error",
		"code":  "internal_error",
	}
	if !isProd && err != nil {
		payload["detail"] = err.Error()
	}
	return c.Status(500).JSON(payload)
}
