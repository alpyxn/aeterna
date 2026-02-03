package services

import (
	"errors"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageService struct{}

var cryptoService = CryptoService{}
var msgValidationService = ValidationService{}

func (s MessageService) Create(content, recipientEmail string, triggerDuration int) (models.Message, error) {
	// Validate trigger duration
	if err := msgValidationService.ValidateTriggerDuration(triggerDuration); err != nil {
		return models.Message{}, err
	}

	// Validate and sanitize content
	if err := msgValidationService.ValidateContent(content); err != nil {
		return models.Message{}, err
	}

	// Validate email format
	if err := msgValidationService.ValidateEmail(recipientEmail); err != nil {
		return models.Message{}, err
	}

	encrypted, err := cryptoService.Encrypt(content)
	if err != nil {
		return models.Message{}, err
	}

	msg := models.Message{
		Content:         encrypted,
		KeyFragment:     "v1",
		ManagementToken: uuid.NewString(),
		RecipientEmail:  recipientEmail,
		TriggerDuration: triggerDuration,
		LastSeen:        time.Now(),
		Status:          models.StatusActive,
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		return models.Message{}, Internal("Failed to create message", err)
	}

	// Return decrypted content for API consumers
	msg.Content = content
	return msg, nil
}

func (s MessageService) GetByID(id string) (models.Message, error) {
	var msg models.Message
	if err := database.DB.First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, NotFound("Message not found", err)
		}
		return models.Message{}, Internal("Failed to fetch message", err)
	}
	decrypted, err := cryptoService.Decrypt(msg.Content)
	if err != nil {
		return models.Message{}, err
	}
	msg.Content = decrypted
	return msg, nil
}

func (s MessageService) List() ([]models.Message, error) {
	var messages []models.Message
	if err := database.DB.Order("created_at DESC").Find(&messages).Error; err != nil {
		return nil, Internal("Failed to fetch messages", err)
	}
	for i := range messages {
		decrypted, err := cryptoService.Decrypt(messages[i].Content)
		if err != nil {
			return nil, err
		}
		messages[i].Content = decrypted
	}
	return messages, nil
}

func (s MessageService) Heartbeat(id string) (models.Message, error) {
	var msg models.Message
	if err := database.DB.First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, NotFound("Message not found", err)
		}
		return models.Message{}, Internal("Failed to fetch message", err)
	}

	if msg.Status == models.StatusTriggered {
		return models.Message{}, BadRequest("Cannot send heartbeat to a triggered message. The message has already been delivered.", nil)
	}

	msg.LastSeen = time.Now()
	if err := database.DB.Save(&msg).Error; err != nil {
		return models.Message{}, Internal("Failed to update heartbeat", err)
	}

	return msg, nil
}

func (s MessageService) Delete(id string) error {
	var msg models.Message
	if err := database.DB.First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Message not found", err)
		}
		return Internal("Failed to fetch message", err)
	}

	if err := database.DB.Unscoped().Delete(&msg).Error; err != nil {
		return Internal("Failed to delete message", err)
	}

	return nil
}
