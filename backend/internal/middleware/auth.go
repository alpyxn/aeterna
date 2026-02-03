package middleware

import (
	"net/url"
	"os"
	"strings"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var authService = services.AuthService{}

func MasterAuth(c *fiber.Ctx) error {
	if token := c.Cookies("aeterna_session"); token != "" {
		if err := authService.VerifySessionToken(token); err == nil {
			if err := enforceOriginAllowlist(c); err != nil {
				return err
			}
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
	
	// Same-origin requests may not send Origin header
	// In that case, check Referer or allow the request
	if origin == "" {
		referer := strings.TrimSpace(c.Get("Referer"))
		if referer != "" {
			parsed, err := url.Parse(referer)
			if err == nil && parsed.Host != "" {
				origin = parsed.Scheme + "://" + parsed.Host
			}
		}
	}
	
	// If still no origin (same-origin fetch, curl, etc.), allow in development
	if origin == "" {
		if os.Getenv("ENV") != "production" {
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

	allowedOrigins := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173"
	}
	for _, entry := range strings.Split(allowedOrigins, ",") {
		if strings.TrimSpace(entry) == origin {
			return nil
		}
	}

	return c.Status(403).JSON(fiber.Map{
		"error": "Origin not allowed",
		"code":  "origin_not_allowed",
	})
}

