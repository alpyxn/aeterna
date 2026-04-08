package middleware

import (
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var authService = services.AuthService{}

func MasterAuth(c *fiber.Ctx) error {
	if token := c.Cookies("aeterna_session"); token != "" {
		userID, err := authService.VerifySessionToken(token)
		if err == nil {
			if err := enforceOriginAllowlist(c); err != nil {
				return err
			}
			c.Locals("user_id", userID)
			return c.Next()
		}
		c.ClearCookie("aeterna_session")
	}

	return c.Status(401).JSON(fiber.Map{
		"error": "Unauthorized access. Session required.",
		"code":  "unauthorized",
	})
}

func enforceOriginAllowlist(c *fiber.Ctx) error {
	origin := strings.TrimSpace(c.Get("Origin"))
	allowedOrigins := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))

	if os.Getenv("ENV") != "production" {
		slog.Info("Origin check", "origin", origin, "allowed", allowedOrigins, "referer", c.Get("Referer"))
	}

	if allowedOrigins == "*" {
		return nil
	}

	if origin == "" {
		referer := strings.TrimSpace(c.Get("Referer"))
		if referer != "" {
			parsed, err := url.Parse(referer)
			if err == nil && parsed.Host != "" {
				origin = parsed.Scheme + "://" + parsed.Host
			}
		}
	}

	if origin == "" {
		env := os.Getenv("ENV")
		if env != "production" {
			return nil
		}
		return c.Status(403).JSON(fiber.Map{
			"error": "Origin required",
			"code":  "origin_required",
		})
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Invalid origin",
			"code":  "invalid_origin",
		})
	}

	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}

	for _, entry := range strings.Split(allowedOrigins, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if origin == entry {
			return nil
		}
	}

	return c.Status(403).JSON(fiber.Map{
		"error": "Origin not allowed",
		"code":  "origin_not_allowed",
	})
}
