package services

import (
	"crypto/subtle"
	"errors"
	"os"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct{}

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

func (s AuthService) SetMasterPassword(password string) error {
	if len(password) < 8 {
		return BadRequest("Master password must be at least 8 characters", nil)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Internal("Failed to hash master password", err)
	}

	var settings models.Settings
	result := database.DB.First(&settings)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			settings = models.Settings{
				ID:                 1,
				MasterPasswordHash: string(hash),
			}
			if err := database.DB.Create(&settings).Error; err != nil {
				return Internal("Failed to save master password", err)
			}
			return nil
		}
		return Internal("Failed to fetch settings", result.Error)
	}

	settings.MasterPasswordHash = string(hash)
	if err := database.DB.Save(&settings).Error; err != nil {
		return Internal("Failed to save master password", err)
	}
	return nil
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
