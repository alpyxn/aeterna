package middleware

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders adds security-related HTTP headers to all responses
func SecurityHeaders(c *fiber.Ctx) error {
	// Prevent MIME type sniffing
	c.Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking
	c.Set("X-Frame-Options", "DENY")

	// XSS Protection (legacy but still useful for older browsers)
	c.Set("X-XSS-Protection", "1; mode=block")

	// Control referrer information
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Permissions Policy (formerly Feature-Policy)
	c.Set("Permissions-Policy", "geolocation=(), camera=(), microphone=(), payment=()")

	// Strict Transport Security (only in production with HTTPS)
	if os.Getenv("ENV") == "production" {
		c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	// Prevent caching of sensitive API responses
	if strings.HasPrefix(c.Path(), "/api") {
		c.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		c.Set("Pragma", "no-cache")
		c.Set("Expires", "0")
	}

	return c.Next()
}
