package handlers

import (
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/middleware"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type registerRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	OwnerEmail string `json:"owner_email"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type resetPasswordRequest struct {
	Email       string `json:"email"`
	RecoveryKey string `json:"recovery_key"`
	NewPassword string `json:"new_password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type sessionMode int

const (
	sessionModeCookie sessionMode = iota
	sessionModeBearer
)

// AuthHandlers groups all authentication-related route handlers.
type AuthHandlers struct {
	auth ports.AuthServicePort
	cfg  config.Config
}

func NewAuthHandlers(auth ports.AuthServicePort, cfg config.Config) *AuthHandlers {
	return &AuthHandlers{auth: auth, cfg: cfg}
}

func (h *AuthHandlers) SetupStatus(c *fiber.Ctx) error {
	configured, err := h.auth.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	out := fiber.Map{"configured": configured}
	if configured {
		allow, err := h.auth.AdditionalRegistrationOpen()
		if err != nil {
			return writeError(c, err)
		}
		out["allow_registration"] = allow
	} else {
		out["allow_registration"] = false
	}
	return c.JSON(out)
}

func (h *AuthHandlers) SetupMasterPassword(c *fiber.Ctx) error {
	return h.setupMasterPassword(c, sessionModeCookie)
}

func (h *AuthHandlers) SetupMasterPasswordV2(c *fiber.Ctx) error {
	return h.setupMasterPassword(c, sessionModeBearer)
}

func (h *AuthHandlers) setupMasterPassword(c *fiber.Ctx, mode sessionMode) error {
	configured, err := h.auth.IsConfigured()
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
		req.Email = req.OwnerEmail
	}
	recoveryKey, user, err := h.auth.RegisterFirstUser(req.Email, req.Password, req.OwnerEmail)
	if err != nil {
		return writeError(c, err)
	}
	return h.respondWithSession(c, user.ID, mode, recoveryKey)
}

func (h *AuthHandlers) Register(c *fiber.Ctx) error {
	return h.register(c, sessionModeCookie)
}

func (h *AuthHandlers) RegisterV2(c *fiber.Ctx) error {
	return h.register(c, sessionModeBearer)
}

func (h *AuthHandlers) register(c *fiber.Ctx, mode sessionMode) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	configured, err := h.auth.IsConfigured()
	if err != nil {
		return writeError(c, err)
	}
	var recoveryKey string
	var userID string
	if !configured {
		rk, u, err := h.auth.RegisterFirstUser(req.Email, req.Password, req.OwnerEmail)
		if err != nil {
			return writeError(c, err)
		}
		recoveryKey, userID = rk, u.ID
	} else {
		rk, u, err := h.auth.RegisterAdditionalUser(req.Email, req.Password, req.OwnerEmail)
		if err != nil {
			return writeError(c, err)
		}
		recoveryKey, userID = rk, u.ID
	}
	return h.respondWithSession(c, userID, mode, recoveryKey)
}

func (h *AuthHandlers) Login(c *fiber.Ctx) error {
	return h.login(c, sessionModeCookie, false)
}

func (h *AuthHandlers) LoginV2(c *fiber.Ctx) error {
	return h.login(c, sessionModeBearer, false)
}

func (h *AuthHandlers) login(c *fiber.Ctx, mode sessionMode, requireEmail bool) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}
	if requireEmail && req.Email == "" {
		return writeError(c, services.BadRequest("Email is required", nil))
	}
	user, err := h.auth.Login(req.Email, req.Password)
	if err != nil {
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	middleware.RecordSuccessfulLogin(c.IP())
	return h.respondWithSession(c, user.ID, mode, "")
}

func (h *AuthHandlers) ResetMasterPassword(c *fiber.Ctx) error {
	return h.resetPassword(c, sessionModeCookie)
}

// VerifyMasterPassword is kept for backward compatibility: same as Login.
func (h *AuthHandlers) VerifyMasterPassword(c *fiber.Ctx) error {
	return h.login(c, sessionModeCookie, true)
}

func (h *AuthHandlers) SessionStatus(c *fiber.Ctx) error {
	token := c.Cookies("aeterna_session")
	userID, err := h.auth.VerifySessionToken(token)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"authorized": true, "user_id": userID})
}

func (h *AuthHandlers) SessionStatusV2(c *fiber.Ctx) error {
	token, ok := middleware.ExtractBearerToken(c.Get("Authorization"))
	if !ok {
		token = c.Cookies("aeterna_session")
	}
	userID, err := h.auth.VerifySessionToken(token)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"authorized": true, "user_id": userID})
}

func (h *AuthHandlers) RefreshV2(c *fiber.Ctx) error {
	var req refreshRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return writeError(c, services.BadRequest("Invalid request body", err))
		}
	}
	if strings.TrimSpace(req.RefreshToken) == "" {
		return writeError(c, services.NewAPIError(401, "unauthorized", "Invalid refresh token.", nil))
	}
	userID, accessToken, accessExp, nextRefreshToken, nextRefreshExp, err := h.auth.RefreshSessionPair(req.RefreshToken)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(bearerSessionPayload(userID, accessToken, accessExp, nextRefreshToken, nextRefreshExp))
}

func (h *AuthHandlers) Logout(c *fiber.Ctx) error {
	h.clearSessionCookie(c)
	return c.JSON(fiber.Map{"success": true})
}

func (h *AuthHandlers) LogoutV2(c *fiber.Ctx) error {
	var req refreshRequest
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return writeError(c, services.BadRequest("Invalid request body", err))
		}
	}
	if revokeErr := h.auth.RevokeRefreshToken(req.RefreshToken); revokeErr != nil {
		return writeError(c, revokeErr)
	}
	h.clearSessionCookie(c)
	return c.JSON(fiber.Map{"success": true})
}

func (h *AuthHandlers) ResetMasterPasswordV2(c *fiber.Ctx) error {
	return h.resetPassword(c, sessionModeBearer)
}

func (h *AuthHandlers) resetPassword(c *fiber.Ctx, mode sessionMode) error {
	var req resetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return writeError(c, services.BadRequest("Invalid request body", err))
	}

	newRecoveryKey, err := h.auth.ResetPasswordWithRecovery(req.Email, req.RecoveryKey, req.NewPassword)
	if err != nil {
		middleware.RecordFailedLogin(c.IP())
		return writeError(c, err)
	}
	middleware.RecordSuccessfulLogin(c.IP())

	user, err := h.auth.Login(req.Email, req.NewPassword)
	if err != nil {
		return writeError(c, err)
	}
	return h.respondWithSession(c, user.ID, mode, newRecoveryKey)
}

func (h *AuthHandlers) respondWithSession(c *fiber.Ctx, userID string, mode sessionMode, recoveryKey string) error {
	session, err := h.sessionPayload(c, userID, mode)
	if err != nil {
		return writeError(c, err)
	}
	if recoveryKey != "" {
		session["recovery_key"] = recoveryKey
	}
	return c.JSON(session)
}

func (h *AuthHandlers) sessionPayload(c *fiber.Ctx, userID string, mode sessionMode) (fiber.Map, error) {
	if mode == sessionModeCookie {
		if err := h.issueSessionCookie(c, userID); err != nil {
			return nil, err
		}
		return fiber.Map{"success": true}, nil
	}
	return h.issueSessionPayload(userID)
}

func (h *AuthHandlers) issueSessionPayload(userID string) (fiber.Map, error) {
	accessToken, accessExp, refreshToken, refreshExp, err := h.auth.IssueSessionPair(userID)
	if err != nil {
		return nil, err
	}
	return bearerSessionPayload(userID, accessToken, accessExp, refreshToken, refreshExp), nil
}

func bearerSessionPayload(userID, accessToken string, accessExp time.Time, refreshToken string, refreshExp time.Time) fiber.Map {
	return fiber.Map{
		"success":            true,
		"user_id":            userID,
		"token_type":         "Bearer",
		"access_token":       accessToken,
		"expires_at":         accessExp.UTC().Format(time.RFC3339),
		"refresh_token":      refreshToken,
		"refresh_expires_at": refreshExp.UTC().Format(time.RFC3339),
	}
}

func (h *AuthHandlers) issueSessionCookie(c *fiber.Ctx, userID string) error {
	token, exp, err := h.auth.IssueSessionToken(userID)
	if err != nil {
		return err
	}
	secure := middleware.ShouldUseSecureCookie(c, h.cfg.Auth.CookieSecureMode)
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

func (h *AuthHandlers) clearSessionCookie(c *fiber.Ctx) {
	secure := middleware.ShouldUseSecureCookie(c, h.cfg.Auth.CookieSecureMode)
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
