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
	Exp int64 `json:"exp"`
	Iat int64 `json:"iat"`
}

func (s AuthService) IssueSessionToken() (string, time.Time, error) {
	ttl := sessionTTL()
	now := time.Now().UTC()
	exp := now.Add(ttl)

	claims := sessionClaims{
		Exp: exp.Unix(),
		Iat: now.Unix(),
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

func (s AuthService) VerifySessionToken(token string) error {
	if token == "" {
		return NewAPIError(401, "unauthorized", "Unauthorized access. Master key required.", nil)
	}

	decrypted, err := cryptoService.Decrypt(token)
	if err != nil {
		return NewAPIError(401, "unauthorized", "Unauthorized access. Master key required.", err)
	}

	var claims sessionClaims
	if err := json.Unmarshal([]byte(decrypted), &claims); err != nil {
		return NewAPIError(401, "unauthorized", "Unauthorized access. Master key required.", err)
	}

	if claims.Exp == 0 || time.Now().UTC().After(time.Unix(claims.Exp, 0)) {
		return NewAPIError(401, "unauthorized", "Session expired", nil)
	}

	return nil
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

func (s AuthService) IsConfigured() (bool, error) {
	if os.Getenv("MASTER_PASSWORD") != "" {
		return true, nil
	}
	hash, err := s.GetMasterHash()
	if err != nil {
		return false, err
	}
	return hash != "", nil
}

func (s AuthService) GetMasterHash() (string, error) {
	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", Internal("Failed to fetch settings", result.Error)
	}
	return settings.MasterPasswordHash, nil
}

var validationService = ValidationService{}

func generateRecoveryKey() (string, error) {
	bytes := make([]byte, 10) // 20 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hexStr := strings.ToUpper(hex.EncodeToString(bytes))
	return fmt.Sprintf("RK-%s-%s-%s-%s", hexStr[0:5], hexStr[5:10], hexStr[10:15], hexStr[15:20]), nil
}

func (s AuthService) SetMasterPassword(password string, ownerEmail string) (string, error) {
	if err := validationService.ValidatePassword(password); err != nil {
		return "", err
	}

	if ownerEmail != "" {
		if err := validationService.ValidateEmail(ownerEmail); err != nil {
			return "", err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash master password", err)
	}

	recoveryKey, err := generateRecoveryKey()
	if err != nil {
		return "", Internal("Failed to generate recovery key", err)
	}

	recoveryHash, err := bcrypt.GenerateFromPassword([]byte(recoveryKey), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash recovery key", err)
	}

	// Generate heartbeat token
	heartbeatToken, err := cryptoService.GenerateToken(32)
	if err != nil {
		return "", Internal("Failed to generate heartbeat token", err)
	}

	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			settings = models.Settings{
				ID:                 1,
				MasterPasswordHash: string(hash),
				RecoveryKeyHash:    string(recoveryHash),
				OwnerEmail:         ownerEmail,
				HeartbeatToken:     heartbeatToken,
			}
			if err := database.DB.Create(&settings).Error; err != nil {
				return "", Internal("Failed to save master password", err)
			}
			return recoveryKey, nil
		}
		return "", Internal("Failed to fetch settings", result.Error)
	}

	settings.MasterPasswordHash = string(hash)
	settings.RecoveryKeyHash = string(recoveryHash)
	settings.OwnerEmail = ownerEmail
	settings.HeartbeatToken = heartbeatToken
	if err := database.DB.Save(&settings).Error; err != nil {
		return "", Internal("Failed to save master password", err)
	}
	return recoveryKey, nil
}

func (s AuthService) ResetMasterPassword(recoveryKey string, newPassword string) (string, error) {
	if err := validationService.ValidatePassword(newPassword); err != nil {
		return "", err
	}

	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		return "", Internal("Failed to fetch settings", result.Error)
	}

	if settings.RecoveryKeyHash == "" {
		return "", BadRequest("Recovery key not configured for this account", nil)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(settings.RecoveryKeyHash), []byte(recoveryKey)); err != nil {
		return "", NewAPIError(401, "unauthorized", "Invalid recovery key.", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash new master password", err)
	}

	newRecoveryKey, err := generateRecoveryKey()
	if err != nil {
		return "", Internal("Failed to generate new recovery key", err)
	}

	newRecoveryHash, err := bcrypt.GenerateFromPassword([]byte(newRecoveryKey), bcrypt.DefaultCost)
	if err != nil {
		return "", Internal("Failed to hash new recovery key", err)
	}

	settings.MasterPasswordHash = string(hash)
	settings.RecoveryKeyHash = string(newRecoveryHash)

	if err := database.DB.Save(&settings).Error; err != nil {
		return "", Internal("Failed to update password and recovery key", err)
	}

	return newRecoveryKey, nil
}

func (s AuthService) VerifyMasterPassword(password string) error {
	if password == "" {
		return BadRequest("Master password is required", nil)
	}

	if envPassword := os.Getenv("MASTER_PASSWORD"); envPassword != "" {
		if subtle.ConstantTimeCompare([]byte(envPassword), []byte(password)) != 1 {
			return NewAPIError(401, "unauthorized", "Unauthorized access. Master key required.", nil)
		}
		return nil
	}

	hash, err := s.GetMasterHash()
	if err != nil {
		return err
	}
	if hash == "" {
		return NewAPIError(401, "setup_required", "Master password not configured", nil)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return NewAPIError(401, "unauthorized", "Unauthorized access. Master key required.", err)
	}
	return nil
}
