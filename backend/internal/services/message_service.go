package services

import (
	"errors"
	"log/slog"
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

type attachCountRow struct {
	MessageID string
	Count     int64
}

type farewellCountRow struct {
	MessageID        string
	Total            int64
	PendingFarewells int64
}

func enrichMessageSchedule(msg *models.Message) {
	if msg == nil {
		return
	}

	triggerAt := msg.LastSeen.UTC().Add(time.Duration(msg.TriggerDuration) * time.Minute)
	triggerAtUTC := triggerAt.UTC()
	msg.NextTriggerAt = &triggerAtUTC
	msg.NextReminderAt = nil

	if msg.Status != models.StatusActive {
		return
	}

	for _, reminder := range msg.Reminders {
		if reminder.Sent {
			continue
		}
		candidate := triggerAt.Add(-time.Duration(reminder.MinutesBefore) * time.Minute).UTC()
		if msg.NextReminderAt == nil || candidate.Before(*msg.NextReminderAt) {
			candidateUTC := candidate
			msg.NextReminderAt = &candidateUTC
		}
	}
}

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
		LastSeen:        time.Now().UTC(),
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
	enrichMessageSchedule(&msg)
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
	enrichMessageSchedule(&msg)

	return msg, nil
}

func (s MessageService) List(userID string) ([]models.Message, error) {
	var messages []models.Message
	if err := database.ForTenant(userID).Preload("Reminders").Order("created_at DESC").Find(&messages).Error; err != nil {
		return nil, Internal("Failed to fetch messages", err)
	}

	if len(messages) == 0 {
		return messages, nil
	}

	msgIDs := make([]string, len(messages))
	for i := range messages {
		decrypted, err := cryptoService.Decrypt(messages[i].Content)
		if err != nil {
			return nil, err
		}
		messages[i].Content = decrypted
		msgIDs[i] = messages[i].ID
	}

	var attachCounts []attachCountRow
	if err := database.ForTenant(userID).Model(&models.Attachment{}).
		Select("message_id, COUNT(*) as count").
		Where("message_id IN ?", msgIDs).
		Group("message_id").
		Scan(&attachCounts).Error; err != nil {
		slog.Warn("Failed to batch-count attachments", "user_id", userID, "error", err)
	}

	attachMap := make(map[string]int64, len(attachCounts))
	for _, r := range attachCounts {
		attachMap[r.MessageID] = r.Count
	}

	var farewellCounts []farewellCountRow
	if err := database.ForTenant(userID).Model(&models.FarewellLetter{}).
		Select("message_id, COUNT(*) as total, COUNT(CASE WHEN status = ? THEN 1 END) as pending_farewells", models.FarewellStatusPending).
		Where("message_id IN ?", msgIDs).
		Group("message_id").
		Scan(&farewellCounts).Error; err != nil {
		slog.Warn("Failed to batch-count farewell letters", "user_id", userID, "error", err)
	}

	farewellMap := make(map[string]farewellCountRow, len(farewellCounts))
	for _, r := range farewellCounts {
		farewellMap[r.MessageID] = r
	}

	for i := range messages {
		messages[i].AttachmentCount = attachMap[messages[i].ID]
		if fc, ok := farewellMap[messages[i].ID]; ok {
			messages[i].FarewellCount = fc.Total
			messages[i].PendingFarewells = fc.PendingFarewells
		}
		enrichMessageSchedule(&messages[i])
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

	msg.LastSeen = time.Now().UTC()
	if err := database.ForTenant(userID).Save(&msg).Error; err != nil {
		return models.Message{}, Internal("Failed to update heartbeat", err)
	}
	enrichMessageSchedule(&msg)

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

	// Filesystem cleanup for farewell letter attachments; DB records are cascaded by Message.BeforeDelete.
	var letters []models.FarewellLetter
	if err := database.ForTenant(userID).Where("message_id = ?", id).Find(&letters).Error; err != nil {
		return Internal("Failed to fetch farewell letters", err)
	}
	for _, letter := range letters {
		if err := msgFileService.DeleteFarewellAttachmentsByLetterID(userID, letter.ID); err != nil {
			return Internal("Failed to delete farewell letter attachments", err)
		}
	}

	if err := database.ForTenant(userID).Unscoped().Delete(&msg).Error; err != nil {
		return Internal("Failed to delete message", err)
	}

	return nil
}

// BulkHeartbeat resets last_seen for all active messages of a user and clears sent reminders.
func (s MessageService) BulkHeartbeat(userID string) error {
	now := time.Now().UTC()
	return database.DB.Transaction(func(tx *gorm.DB) error {
		if err := database.TenantTx(tx, userID).Model(&models.Message{}).
			Where("status = ?", models.StatusActive).
			Update("last_seen", now).Error; err != nil {
			return Internal("failed to update heartbeats", err)
		}
		if err := tx.Model(&models.MessageReminder{}).
			Where("message_id IN (SELECT id FROM messages WHERE user_id = ? AND status = ?)", userID, models.StatusActive).
			Update("sent", false).Error; err != nil {
			return Internal("failed to reset reminders", err)
		}
		return nil
	})
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
	msg.LastSeen = time.Now().UTC()
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := database.TenantTx(tx, userID).Save(&msg).Error; err != nil {
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
	enrichMessageSchedule(&msg)
	return msg, nil
}
