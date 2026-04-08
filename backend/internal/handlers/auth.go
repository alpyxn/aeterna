package handlers

import (
	"os"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	OwnerEmail  string `json:"owner_email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var authService = services.AuthService{}

func SetupStatus(c *fiber.Ctx) error {
	configured, err := authService.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	out := fiber.Map{"configured": configured}
	if configured {
		allow, err := authService.AdditionalRegistrationOpen()
		if err != nil {
			return writeError(c, err)
		}
		out["allow_registration"] = allow
	} else {
		out["allow_registration"] = false
	}
	return c.JSON(out)
}

// SetupMasterPassword is the initial install: creates the first user (same as Register when no users exist).
func SetupMasterPassword(c *fiber.Ctx) error {
	configured, err := authService.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	if configured {
		return writeError(c, services.NewAPIError(400, "already_configured", "An account already exists. Sign in instead.", nil))
	}

	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if req.Email == "" && req.Password != "" {
		// Backward compatibility: old clients sent only password + owner_email
		req.Email = req.OwnerEmail
	}
	recoveryKey, user, err := authService.RegisterFirstUser(req.Email, req.Password, req.OwnerEmail)
	if err != nil {
		return writeError(c, err)
	}
	if err := issueSessionCookie(c, user.ID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "recovery_key": recoveryKey})
}

// Register creates the first account or an additional account when ALLOW_REGISTRATION=true.
func Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	configured, err := authService.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	var recoveryKey string
	var user models.User
	if !configured {
		recoveryKey, user, err = authService.RegisterFirstUser(req.Email, req.Password, req.OwnerEmail)
	} else {
		recoveryKey, user, err = authService.RegisterAdditionalUser(req.Email, req.Password, req.OwnerEmail)
	}
	if err != nil {
		return writeError(c, err)
	}
	if err := issueSessionCookie(c, user.ID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "recovery_key": recoveryKey})
}

// Login authenticates with email and password.
func Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	user, err := authService.Login(req.Email, req.Password)
	if err != nil {
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	middleware.RecordSuccessfulLogin(c.IP())
	if err := issueSessionCookie(c, user.ID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func ResetMasterPassword(c *fiber.Ctx) error {
	var req struct {
		Email       string `json:"email"`
		RecoveryKey string `json:"recovery_key"`
		NewPassword string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	newRecoveryKey, err := authService.ResetPasswordWithRecovery(req.Email, req.RecoveryKey, req.NewPassword)
	if err != nil {
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	middleware.RecordSuccessfulLogin(c.IP())

	user, err := authService.Login(req.Email, req.NewPassword)
	if err != nil {
		return writeError(c, err)
	}
	if err := issueSessionCookie(c, user.ID); err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{"success": true, "recovery_key": newRecoveryKey})
}

// VerifyMasterPassword is kept for backward compatibility: same as Login with email in body.
func VerifyMasterPassword(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if req.Email == "" {
		return writeError(c, services.BadRequest("Email is required", nil))
	}
	user, err := authService.Login(req.Email, req.Password)
	if err != nil {
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	middleware.RecordSuccessfulLogin(c.IP())
	if err := issueSessionCookie(c, user.ID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func SessionStatus(c *fiber.Ctx) error {
	token := c.Cookies("aeterna_session")
	userID, err := authService.VerifySessionToken(token)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"authorized": true, "user_id": userID})
}

func Logout(c *fiber.Ctx) error {
	clearSessionCookie(c)
	return c.JSON(fiber.Map{"success": true})
}

func issueSessionCookie(c *fiber.Ctx, userID string) error {
	token, exp, err := authService.IssueSessionToken(userID)
	if err != nil {
		return err
	}
	isHTTPS := c.Protocol() == "https" || c.Get("X-Forwarded-Proto") == "https"
	secure := os.Getenv("ENV") == "production" && isHTTPS
	c.Cookie(&fiber.Cookie{
		Name:     "aeterna_session",
		Value:    token,
		Expires:  exp,
		Path:     "/",
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteStrictMode,
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
		SameSite: fiber.CookieSameSiteStrictMode,
	})
}
