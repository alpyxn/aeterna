package services

import (
	"errors"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

const applicationSettingsSingletonID uint = 1

type ApplicationSettingsService struct{}

func (s ApplicationSettingsService) Get() (models.ApplicationSettings, error) {
	var app models.ApplicationSettings
	err := database.DB.First(&app, applicationSettingsSingletonID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.ApplicationSettings{ID: applicationSettingsSingletonID, AllowRegistration: false}, nil
		}
		return models.ApplicationSettings{}, Internal("Failed to load application settings", err)
	}
	return app, nil
}

// SetAllowRegistration updates the global flag; only the first (primary) user may call this.
func (s ApplicationSettingsService) SetAllowRegistration(actorUserID string, allow bool) error {
	if !IsFirstUser(actorUserID) {
		return NewAPIError(403, "forbidden", "Only the primary administrator can change registration settings.", nil)
	}
	var app models.ApplicationSettings
	err := database.DB.First(&app, applicationSettingsSingletonID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		app = models.ApplicationSettings{ID: applicationSettingsSingletonID, AllowRegistration: allow}
		return database.DB.Create(&app).Error
	}
	if err != nil {
		return Internal("Failed to load application settings", err)
	}
	app.AllowRegistration = allow
	return database.DB.Save(&app).Error
}

// EnsureApplicationSettingsRow creates the singleton row if missing.
func EnsureApplicationSettingsRow() error {
	var n int64
	if err := database.DB.Model(&models.ApplicationSettings{}).Where("id = ?", applicationSettingsSingletonID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	return database.DB.Create(&models.ApplicationSettings{
		ID:                applicationSettingsSingletonID,
		AllowRegistration: false,
	}).Error
}
