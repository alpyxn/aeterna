package services

import (
	"errors"
	"strings"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

type MessageService struct{}

var cryptoService = CryptoService{}
var msgValidationService = ValidationService{}
var msgFileService = FileService{}
var msgSettingsService = SettingsService{}

func (s MessageService) Create(userID string, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	settings, err := msgSettingsService.Get(userID)
	if err != nil {
		return models.Message{}, err
	}
	if settings.SMTPUser == "" || settings.SMTPHost == "" {
		return models.Message{}, BadRequest("SMTP_NOT_CONFIGURED: SMTP is not configured. Please go to Settings to configure your email server.", nil)
	}

	if err := msgSettingsService.TestSMTP(settings); err != nil {
		return models.Message{}, BadRequest("SMTP_CONNECTION_FAILED: SMTP connection test failed. Please check your email settings.", err)
	}

	if err := msgValidationService.ValidateTriggerDuration(triggerDuration); err != nil {
		return models.Message{}, err
	}

	if err := msgValidationService.ValidateContent(content); err != nil {
		return models.Message{}, err
	}

	if len(recipientEmails) == 0 {
		return models.Message{}, BadRequest("At least one recipient email is required", nil)
	}
	for _, recipientEmail := range recipientEmails {
		if err := msgValidationService.ValidateEmail(recipientEmail); err != nil {
			return models.Message{}, err
		}
	}

	normalizedRecipients := strings.Join(recipientEmails, ",")
	if len(normalizedRecipients) > 2000 {
		return models.Message{}, BadRequest("Too many recipient emails", nil)
	}

	if err := msgValidationService.ValidateEmailListLength(len(recipientEmails)); err != nil {
		return models.Message{}, err
	}

	encrypted, err := cryptoService.Encrypt(content)
	if err != nil {
		return models.Message{}, err
	}

	msg := models.Message{
		UserID:          userID,
		Content:         encrypted,
		KeyFragment:     "v1",
		RecipientEmail:  normalizedRecipients,
		TriggerDuration: triggerDuration,
		LastSeen:        time.Now(),
		Status:          models.StatusActive,
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&msg).Error; err != nil {
			return Internal("Failed to create message", err)
		}

		for _, minutesBefore := range reminders {
			reminder := models.MessageReminder{
				MessageID:     msg.ID,
				MinutesBefore: minutesBefore,
				Sent:          false,
			}
			if err := tx.Create(&reminder).Error; err != nil {
				return Internal("Failed to create reminder", err)
			}
			msg.Reminders = append(msg.Reminders, reminder)
		}
		return nil
	})

	if err != nil {
		return models.Message{}, err
	}

	msg.Content = content
	return msg, nil
}

// GetPublicByID loads a message by ID for the unauthenticated reveal endpoint (no tenant check).
func (s MessageService) GetPublicByID(id string) (models.Message, error) {
	var msg models.Message
	if err := database.DB.Preload("Reminders").First(&msg, "id = ?", id).Error; err != nil {
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

	count, _ := msgFileService.CountByMessageID(msg.UserID, id)
	msg.AttachmentCount = count

	return msg, nil
}

func (s MessageService) GetByID(userID, id string) (models.Message, error) {
	var msg models.Message
	if err := database.ForTenant(userID).Preload("Reminders").First(&msg, "id = ?", id).Error; err != nil {
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

	count, _ := msgFileService.CountByMessageID(userID, id)
	msg.AttachmentCount = count

	return msg, nil
}

func (s MessageService) List(userID string) ([]models.Message, error) {
	var messages []models.Message
	if err := database.ForTenant(userID).Preload("Reminders").Order("created_at DESC").Find(&messages).Error; err != nil {
		return nil, Internal("Failed to fetch messages", err)
	}
	for i := range messages {
		decrypted, err := cryptoService.Decrypt(messages[i].Content)
		if err != nil {
			return nil, err
		}
		messages[i].Content = decrypted

		count, _ := msgFileService.CountByMessageID(userID, messages[i].ID)
		messages[i].AttachmentCount = count
	}
	return messages, nil
}

func (s MessageService) Heartbeat(userID, id string) (models.Message, error) {
	var msg models.Message
	if err := database.ForTenant(userID).First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, NotFound("Message not found", err)
		}
		return models.Message{}, Internal("Failed to fetch message", err)
	}

	if msg.Status == models.StatusTriggered {
		return models.Message{}, BadRequest("Cannot send heartbeat to a triggered message. The message has already been delivered.", nil)
	}

	msg.LastSeen = time.Now()
	if err := database.ForTenant(userID).Save(&msg).Error; err != nil {
		return models.Message{}, Internal("Failed to update heartbeat", err)
	}

	return msg, nil
}

func (s MessageService) Delete(userID, id string) error {
	var msg models.Message
	if err := database.ForTenant(userID).First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Message not found", err)
		}
		return Internal("Failed to fetch message", err)
	}

	if err := msgFileService.DeleteByMessageID(userID, id); err != nil {
		return Internal("Failed to delete attachments", err)
	}

	if err := database.DB.Unscoped().Delete(&msg).Error; err != nil {
		return Internal("Failed to delete message", err)
	}

	return nil
}

func (s MessageService) Update(userID, id, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	var msg models.Message
	if err := database.ForTenant(userID).First(&msg, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, NotFound("Message not found", err)
		}
		return models.Message{}, Internal("Failed to fetch message", err)
	}

	if msg.Status == models.StatusTriggered {
		return models.Message{}, BadRequest("Cannot edit a triggered message. The message has already been delivered.", nil)
	}

	if err := msgValidationService.ValidateContent(content); err != nil {
		return models.Message{}, err
	}

	if err := msgValidationService.ValidateTriggerDuration(triggerDuration); err != nil {
		return models.Message{}, err
	}

	if len(recipientEmails) > 0 {
		if err := msgValidationService.ValidateEmailListLength(len(recipientEmails)); err != nil {
			return models.Message{}, err
		}
		for _, recipientEmail := range recipientEmails {
			if err := msgValidationService.ValidateEmail(recipientEmail); err != nil {
				return models.Message{}, err
			}
		}
		msg.RecipientEmail = strings.Join(recipientEmails, ",")
	}

	encrypted, err := cryptoService.Encrypt(content)
	if err != nil {
		return models.Message{}, err
	}

	msg.Content = encrypted
	msg.TriggerDuration = triggerDuration
	msg.LastSeen = time.Now()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&msg).Error; err != nil {
			return Internal("Failed to update message", err)
		}

		if err := tx.Where("message_id = ?", msg.ID).Delete(&models.MessageReminder{}).Error; err != nil {
			return Internal("Failed to delete old reminders", err)
		}

		msg.Reminders = []models.MessageReminder{}
		for _, minutesBefore := range reminders {
			reminder := models.MessageReminder{
				MessageID:     msg.ID,
				MinutesBefore: minutesBefore,
				Sent:          false,
			}
			if err := tx.Create(&reminder).Error; err != nil {
				return Internal("Failed to create new reminder", err)
			}
			msg.Reminders = append(msg.Reminders, reminder)
		}

		return nil
	})

	if err != nil {
		return models.Message{}, err
	}

	msg.Content = content
	return msg, nil
}
