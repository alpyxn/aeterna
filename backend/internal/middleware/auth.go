package middleware

import (
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/gofiber/fiber/v2"
)

func unauthorizedResponse(c *fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": "Unauthorized access. Session required.",
		"code":  "unauthorized",
	})
}

// MasterAuth returns a middleware that validates the session cookie and enforces the origin allowlist.
func MasterAuth(auth ports.AuthServicePort, cfg config.Config) fiber.Handler {
	allowedOrigins := cfg.AllowedOriginsOrDefault()
	isProd := cfg.IsProduction()
	cookieSecureMode := cfg.Auth.CookieSecureMode
	return func(c *fiber.Ctx) error {
		if path := c.Path(); path == "/api/v2" || strings.HasPrefix(path, "/api/v2/") {
			return c.Next()
		}

		if token := c.Cookies("aeterna_session"); token != "" {
			userID, err := auth.VerifySessionToken(token)
			if err == nil {
				if !enforceOriginAllowlist(c, allowedOrigins, isProd) {
					return nil
				}
				c.Locals(LocalUserIDKey, userID)
				c.Locals(LocalSessionKey, SessionKeyFromToken(token))
				return c.Next()
			}
			clearSessionCookieWith(c, cookieSecureMode)
		}

		return unauthorizedResponse(c)
	}
}

func enforceOriginAllowlist(c *fiber.Ctx, allowedOrigins string, isProd bool) bool {
	origin := strings.TrimSpace(c.Get("Origin"))

	if !isProd {
		slog.Info("Origin check", "origin", origin, "allowed", allowedOrigins, "referer", c.Get("Referer"))
	}

	if allowedOrigins == "*" {
		return true
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
		if !isProd {
			return true
		}
		_ = c.Status(403).JSON(fiber.Map{
			"error": "Origin required",
			"code":  "origin_required",
		})
		return false
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		_ = c.Status(403).JSON(fiber.Map{
			"error": "Invalid origin",
			"code":  "invalid_origin",
		})
		return false
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
			return true
		}
	}

	_ = c.Status(403).JSON(fiber.Map{
		"error": "Origin not allowed",
		"code":  "origin_not_allowed",
	})
	return false
}

func clearSessionCookieWith(c *fiber.Ctx, cookieSecureMode string) {
	secure := ShouldUseSecureCookie(c, cookieSecureMode)
	c.Cookie(&fiber.Cookie{
		Name:     "aeterna_session",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}

// ShouldUseSecureCookie returns true when the session cookie should be flagged Secure.
func ShouldUseSecureCookie(c *fiber.Ctx, cookieSecureMode string) bool {
	switch cookieSecureMode {
	case "always":
		return true
	case "never":
		return false
	}
	return requestIsHTTPS(c)
}

func requestIsHTTPS(c *fiber.Ctx) bool {
	if c.Protocol() == "https" {
		return true
	}
	forwardedProto := strings.ToLower(strings.TrimSpace(c.Get("X-Forwarded-Proto")))
	if forwardedProto == "" {
		return false
	}
	first := strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
	return first == "https"
}

func ExtractBearerToken(header string) (string, bool) {
	value := strings.TrimSpace(header)
	if value == "" {
		return "", false
	}
	if len(value) < 7 || !strings.EqualFold(value[:7], "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(value[7:])
	if token == "" {
		return "", false
	}
	return token, true
}
