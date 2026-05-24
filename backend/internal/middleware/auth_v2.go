package middleware

import (
	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/gofiber/fiber/v2"
)

// MasterAuthV2 accepts Bearer tokens for mobile clients and falls back to cookie auth.
// Origin allowlist is enforced only for cookie-based browser sessions.
func MasterAuthV2(auth ports.AuthServicePort, cfg config.Config) fiber.Handler {
	allowedOrigins := cfg.AllowedOriginsOrDefault()
	isProd := cfg.IsProduction()
	cookieSecureMode := cfg.Auth.CookieSecureMode

	return func(c *fiber.Ctx) error {
		if token, ok := ExtractBearerToken(c.Get("Authorization")); ok {
			userID, err := auth.VerifySessionToken(token)
			if err != nil {
				return unauthorizedResponse(c)
			}
			c.Locals(LocalUserIDKey, userID)
			c.Locals(LocalSessionKey, SessionKeyFromToken(token))
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
