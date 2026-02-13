package handlers

import (
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type passwordRequest struct {
	Password   string `json:"password"`
	OwnerEmail string `json:"owner_email"`
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
	if err := authService.SetMasterPassword(req.Password, req.OwnerEmail); err != nil {
		return writeError(c, err)
	}
	if err := issueSessionCookie(c); err != nil {
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
		// Record failed login attempt for rate limiting
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	// Record successful login to reset rate limit counter
	middleware.RecordSuccessfulLogin(c.IP())
	if err := issueSessionCookie(c); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func SessionStatus(c *fiber.Ctx) error {
	token := c.Cookies("aeterna_session")
	if err := authService.VerifySessionToken(token); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"authorized": true})
}

func Logout(c *fiber.Ctx) error {
	clearSessionCookie(c)
	return c.JSON(fiber.Map{"success": true})
}

func issueSessionCookie(c *fiber.Ctx) error {
	token, exp, err := authService.IssueSessionToken()
	if err != nil {
		return err
	}
	// Only set Secure flag if actually using HTTPS
	// This allows Simple mode (HTTP) to work while keeping HTTPS secure
	isHTTPS := c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https"
	secure := os.Getenv("ENV") == "production" && isHTTPS
	c.Cookie(&fiber.Cookie{
		Name:     "aeterna_session",
		Value:    token,
		Expires:  exp,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode, // Lax allows cross-origin for redirects
	})
	return nil
}

func clearSessionCookie(c *fiber.Ctx) {
	isHTTPS := c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https"
	secure := os.Getenv("ENV") == "production" && isHTTPS
	c.Cookie(&fiber.Cookie{
		Name:     "aeterna_session",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
