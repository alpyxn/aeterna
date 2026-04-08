package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct{}

type sessionClaims struct {
	Exp    int64  `json:"exp"`
	Iat    int64  `json:"iat"`
	UserID string `json:"uid"`
	Hash   string `json:"hash,omitempty"`
}

func (s AuthService) passwordHashPrefixForUser(userID string) (string, error) {
	var u models.User
	if err := database.DB.First(&u, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", NewAPIError(401, "unauthorized", "Unauthorized access.", nil)
		}
		return "", Internal("Failed to load user", err)
	}
	h := u.PasswordHash
	if len(h) > 10 {
		return h[:10], nil
	}
	return h, nil
}

// IssueSessionToken creates a session for the given user (tenant).
func (s AuthService) IssueSessionToken(userID string) (string, time.Time, error) {
	hashPrefix, err := s.passwordHashPrefixForUser(userID)
	if err != nil {
		return "", time.Time{}, err
	}
	ttl := sessionTTL()
	now := time.Now().UTC()
	exp := now.Add(ttl)

	claims := sessionClaims{
		Exp:    exp.Unix(),
		Iat:    now.Unix(),
		UserID: userID,
		Hash:   hashPrefix,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, Internal("Failed to encode session", err)
	}

	token, err := cryptoService.Encrypt(string(payload))
	if err != nil {
		return "", time.Time{}, err
	}

	return token, exp, nil
}

// VerifySessionToken validates the cookie token and returns the authenticated user ID.
func (s AuthService) VerifySessionToken(token string) (userID string, err error) {
	if token == "" {
		return "", NewAPIError(401, "unauthorized", "Unauthorized access. Session required.", nil)
	}

	decrypted, err := cryptoService.Decrypt(token)
	if err != nil {
		return "", NewAPIError(401, "unauthorized", "Unauthorized access. Session required.", err)
	}

	var claims sessionClaims
	if err := json.Unmarshal([]byte(decrypted), &claims); err != nil {
		return "", NewAPIError(401, "unauthorized", "Unauthorized access. Session required.", err)
	}

	if claims.UserID == "" {
		return "", NewAPIError(401, "unauthorized", "Invalid session", nil)
	}

	if claims.Exp == 0 || time.Now().UTC().After(time.Unix(claims.Exp, 0)) {
		return "", NewAPIError(401, "unauthorized", "Session expired", nil)
	}

	if claims.Hash != "" {
		prefix, err := s.passwordHashPrefixForUser(claims.UserID)
		if err != nil {
			return "", err
		}
		if claims.Hash != prefix {
			return "", NewAPIError(401, "unauthorized", "Session expired due to password change", nil)
		}
	}

	return claims.UserID, nil
}

func sessionTTL() time.Duration {
	raw := os.Getenv("AUTH_SESSION_TTL_HOURS")
	if raw == "" {
		return 12 * time.Hour
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return 12 * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

// IsConfigured returns true when at least one user account exists.
func (s AuthService) IsConfigured() (bool, error) {
	var n int64
	if err := database.DB.Model(&models.User{}).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s AuthService) normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// RegisterFirstUser creates the first user and settings (initial setup).
func (s AuthService) RegisterFirstUser(email, password, ownerEmail string) (recoveryKey string, user models.User, err error) {
	var n int64
	if err := database.DB.Model(&models.User{}).Count(&n).Error; err != nil {
		return "", models.User{}, err
	}
	if n > 0 {
		return "", models.User{}, NewAPIError(400, "already_configured", "An account already exists. Sign in instead.", nil)
	}

	email = s.normalizeEmail(email)
	if email == "" {
		return "", models.User{}, BadRequest("Email is required", nil)
	}
	if err := validationService.ValidateEmail(email); err != nil {
		return "", models.User{}, err
	}
	if err := validationService.ValidatePassword(password); err != nil {
		return "", models.User{}, err
	}

	ownerEmail = strings.TrimSpace(ownerEmail)
	if ownerEmail != "" {
		if err := validationService.ValidateEmail(ownerEmail); err != nil {
			return "", models.User{}, err
		}
	} else {
		ownerEmail = email
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", models.User{}, Internal("Failed to hash password", err)
	}

	recoveryKey, err = generateRecoveryKey()
	if err != nil {
		return "", models.User{}, Internal("Failed to generate recovery key", err)
	}

	recoveryHash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), bcrypt.DefaultCost)
	if err != nil {
		return "", models.User{}, Internal("Failed to hash recovery key", err)
	}

	heartbeatToken, err := cryptoService.GenerateToken(32)
	if err != nil {
		return "", models.User{}, Internal("Failed to generate heartbeat token", err)
	}

	user = models.User{
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := database.DB.Create(&user).Error; err != nil {
		return "", models.User{}, Internal("Failed to create user", err)
	}

	settings := models.Settings{
		UserID:          user.ID,
		OwnerEmail:      ownerEmail,
		RecoveryKeyHash: string(recoveryHash),
		HeartbeatToken:  heartbeatToken,
	}
	if err := database.DB.Create(&settings).Error; err != nil {
		return "", models.User{}, Internal("Failed to create settings", err)
	}

	return recoveryKey, user, nil
}

var applicationSettingsService = ApplicationSettingsService{}

// AdditionalRegistrationOpen reports whether self-service registration is allowed when an account already exists (env or DB flag).
func (s AuthService) AdditionalRegistrationOpen() (bool, error) {
	if os.Getenv("ALLOW_REGISTRATION") == "true" {
		return true, nil
	}
	app, err := applicationSettingsService.Get()
	if err != nil {
		return false, err
	}
	return app.AllowRegistration, nil
}

// RegisterAdditionalUser creates another user when ALLOW_REGISTRATION=true (env) or application allow_registration (DB, set by primary admin).
func (s AuthService) RegisterAdditionalUser(email, password, ownerEmail string) (recoveryKey string, user models.User, err error) {
	open, err := s.AdditionalRegistrationOpen()
	if err != nil {
		return "", models.User{}, err
	}
	if !open {
		return "", models.User{}, NewAPIError(403, "registration_disabled", "Additional registration is disabled.", nil)
	}
	var n int64
	if err := database.DB.Model(&models.User{}).Count(&n).Error; err != nil {
		return "", models.User{}, err
	}
	if n == 0 {
		return "", models.User{}, BadRequest("Use initial setup first", nil)
	}
	return s.registerUser(email, password, ownerEmail)
}

func (s AuthService) registerUser(email, password, ownerEmail string) (recoveryKey string, user models.User, err error) {
	email = s.normalizeEmail(email)
	if email == "" {
		return "", models.User{}, BadRequest("Email is required", nil)
	}
	if err := validationService.ValidateEmail(email); err != nil {
		return "", models.User{}, err
	}
	if err := validationService.ValidatePassword(password); err != nil {
		return "", models.User{}, err
	}

	var existing int64
	if err := database.DB.Model(&models.User{}).Where("email = ?", email).Count(&existing).Error; err != nil {
		return "", models.User{}, err
	}
	if existing > 0 {
		return "", models.User{}, NewAPIError(400, "email_taken", "That email is already registered.", nil)
	}

	ownerEmail = strings.TrimSpace(ownerEmail)
	if ownerEmail != "" {
		if err := validationService.ValidateEmail(ownerEmail); err != nil {
			return "", models.User{}, err
		}
	} else {
		ownerEmail = email
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", models.User{}, Internal("Failed to hash password", err)
	}

	recoveryKey, err = generateRecoveryKey()
	if err != nil {
		return "", models.User{}, Internal("Failed to generate recovery key", err)
	}

	recoveryHash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), bcrypt.DefaultCost)
	if err != nil {
		return "", models.User{}, Internal("Failed to hash recovery key", err)
	}

	heartbeatToken, err := cryptoService.GenerateToken(32)
	if err != nil {
		return "", models.User{}, Internal("Failed to generate heartbeat token", err)
	}

	user = models.User{
		Email:        email,
		PasswordHash: string(hash),
	}
	if err := database.DB.Create(&user).Error; err != nil {
		return "", models.User{}, Internal("Failed to create user", err)
	}

	settings := models.Settings{
		UserID:          user.ID,
		OwnerEmail:      ownerEmail,
		RecoveryKeyHash: string(recoveryHash),
		HeartbeatToken:  heartbeatToken,
	}
	if err := database.DB.Create(&settings).Error; err != nil {
		return "", models.User{}, Internal("Failed to create settings", err)
	}

	return recoveryKey, user, nil
}

// Login verifies email and password and returns the user.
func (s AuthService) Login(email, password string) (models.User, error) {
	email = s.normalizeEmail(email)
	if email == "" || password == "" {
		return models.User{}, BadRequest("Email and password are required", nil)
	}

	if envPassword := os.Getenv("MASTER_PASSWORD"); envPassword != "" {
		var n int64
		database.DB.Model(&models.User{}).Count(&n)
		if n == 1 {
			var u models.User
			if err := database.DB.First(&u).Error; err == nil {
				if subtle.ConstantTimeCompare([]byte(envPassword), []byte(password)) == 1 {
					return u, nil
				}
			}
		}
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.User{}, NewAPIError(401, "unauthorized", "Invalid email or password.", nil)
		}
		return models.User{}, Internal("Failed to load user", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return models.User{}, NewAPIError(401, "unauthorized", "Invalid email or password.", err)
	}
	return user, nil
}

var validationService = ValidationService{}

func generateRecoveryKey() (string, error) {
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hexStr := strings.ToUpper(hex.EncodeToString(bytes))
	return fmt.Sprintf("RK-%s-%s-%s-%s", hexStr[0:5], hexStr[5:10], hexStr[10:15], hexStr[15:20]), nil
}

// ResetPasswordWithRecovery uses recovery key + email to set a new password for that account.
func (s AuthService) ResetPasswordWithRecovery(email, recoveryKey, newPassword string) (newRecoveryKey string, err error) {
	if err := validationService.ValidatePassword(newPassword); err != nil {
		return "", err
	}
	email = s.normalizeEmail(email)
	if email == "" {
		return "", BadRequest("Email is required", nil)
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", NewAPIError(401, "unauthorized", "Invalid recovery request.", nil)
		}
		return "", Internal("Failed to load user", err)
	}

	var settings models.Settings
	if err := database.DB.Where("user_id = ?", user.ID).First(&settings).Error; err != nil {
		return "", Internal("Failed to load settings", err)
	}
	if settings.RecoveryKeyHash == "" {
		return "", BadRequest("Recovery key not configured for this account", nil)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(settings.RecoveryKeyHash), []byte(recoveryKey)); err != nil {
		return "", NewAPIError(401, "unauthorized", "Invalid recovery key.", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash new password", err)
	}

	newRec, err := generateRecoveryKey()
	if err != nil {
		return "", Internal("Failed to generate new recovery key", err)
	}
	newRecHash, err := bcrypt.GenerateFromPassword([]byte(newRec), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash new recovery key", err)
	}

	user.PasswordHash = string(hash)
	settings.RecoveryKeyHash = string(newRecHash)

	if err := database.DB.Save(&user).Error; err != nil {
		return "", Internal("Failed to update password", err)
	}
	if err := database.DB.Save(&settings).Error; err != nil {
		return "", Internal("Failed to update recovery key", err)
	}

	return newRec, nil
}
