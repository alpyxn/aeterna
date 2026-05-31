package services

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"gorm.io/gorm"
)

type FarewellService struct{}

var farewellCrypto = CryptoService{}
var farewellValidation = ValidationService{}
var farewellFileService = FileService{}

const cancelRequiresTriggeredMessage = "Pending farewell letters can only be canceled after the switch has triggered"

func (s FarewellService) Create(userID, messageID, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error) {
	if err := requireFarewellMessageNotTriggered(userID, messageID, "Cannot add farewell letters after the switch has triggered"); err != nil {
		return models.FarewellLetter{}, err
	}

	if err := farewellValidation.ValidateEmail(recipientEmail); err != nil {
		return models.FarewellLetter{}, err
	}

	if subject == "" {
		return models.FarewellLetter{}, BadRequest("Subject is required", nil)
	}

	if err := farewellValidation.ValidateContent(content); err != nil {
		return models.FarewellLetter{}, err
	}

	if delayMinutes < 0 {
		return models.FarewellLetter{}, BadRequest("Delay must be zero or positive", nil)
	}

	safeMarkdown := sanitizeFarewellMarkdown(content)

	encryptedSafe, err := farewellCrypto.Encrypt(safeMarkdown)
	if err != nil {
		return models.FarewellLetter{}, err
	}

	encryptedRaw, err := farewellCrypto.Encrypt(content)
	if err != nil {
		return models.FarewellLetter{}, err
	}

	letter := models.FarewellLetter{
		UserID:             userID,
		MessageID:          messageID,
		RecipientEmail:     recipientEmail,
		Subject:            subject,
		Content:            encryptedSafe,
		RawContent:         encryptedRaw,
		RenderedHTML:       "",
		WordCount:          0,
		DerivativesPending: true,
		DelayMinutes:       delayMinutes,
		Status:             models.FarewellStatusPending,
	}

	if err := database.ForTenant(userID).Create(&letter).Error; err != nil {
		return models.FarewellLetter{}, Internal("Failed to create farewell letter", err)
	}

	letter.Content = safeMarkdown
	return letter, nil
}

func (s FarewellService) List(userID, messageID string) ([]models.FarewellLetter, error) {
	if _, err := loadFarewellMessage(userID, messageID); err != nil {
		return nil, err
	}

	letters := make([]models.FarewellLetter, 0)
	if err := database.ForTenant(userID).Where("message_id = ?", messageID).Order("created_at ASC").Find(&letters).Error; err != nil {
		return nil, Internal("Failed to fetch farewell letters", err)
	}

	for i := range letters {
		decrypted, err := farewellCrypto.Decrypt(letters[i].Content)
		if err != nil {
			return nil, err
		}
		letters[i].Content = decrypted

		count, _ := farewellFileService.CountFarewellAttachmentsByLetterID(userID, letters[i].ID)
		letters[i].AttachmentCount = count
	}

	return letters, nil
}

func (s FarewellService) Update(userID, messageID, id, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error) {
	if err := requireFarewellMessageNotTriggered(userID, messageID, "Cannot edit farewell letters after the switch has triggered; cancel pending deliveries instead"); err != nil {
		return models.FarewellLetter{}, err
	}

	var letter models.FarewellLetter
	if err := database.ForTenant(userID).Where("message_id = ? AND id = ?", messageID, id).First(&letter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.FarewellLetter{}, NotFound("Farewell letter not found", err)
		}
		return models.FarewellLetter{}, Internal("Failed to fetch farewell letter", err)
	}

	if letter.Status == models.FarewellStatusSent {
		return models.FarewellLetter{}, BadRequest("Cannot edit an already-sent farewell letter", nil)
	}

	if err := farewellValidation.ValidateEmail(recipientEmail); err != nil {
		return models.FarewellLetter{}, err
	}

	if subject == "" {
		return models.FarewellLetter{}, BadRequest("Subject is required", nil)
	}

	if err := farewellValidation.ValidateContent(content); err != nil {
		return models.FarewellLetter{}, err
	}

	if delayMinutes < 0 {
		return models.FarewellLetter{}, BadRequest("Delay must be zero or positive", nil)
	}

	safeMarkdown := sanitizeFarewellMarkdown(content)

	encryptedSafe, err := farewellCrypto.Encrypt(safeMarkdown)
	if err != nil {
		return models.FarewellLetter{}, err
	}

	encryptedRaw, err := farewellCrypto.Encrypt(content)
	if err != nil {
		return models.FarewellLetter{}, err
	}

	letter.RecipientEmail = recipientEmail
	letter.Subject = subject
	letter.Content = encryptedSafe
	letter.RawContent = encryptedRaw
	letter.RenderedHTML = ""
	letter.WordCount = 0
	letter.DerivativesPending = true
	letter.DelayMinutes = delayMinutes

	if err := database.ForTenant(userID).Save(&letter).Error; err != nil {
		return models.FarewellLetter{}, Internal("Failed to update farewell letter", err)
	}

	letter.Content = safeMarkdown
	return letter, nil
}

func (s FarewellService) Delete(userID, messageID, id string) error {
	if err := requireFarewellMessageNotTriggered(userID, messageID, "Cannot delete farewell letters after the switch has triggered; cancel pending deliveries instead"); err != nil {
		return err
	}

	var letter models.FarewellLetter
	if err := database.ForTenant(userID).Where("message_id = ? AND id = ?", messageID, id).First(&letter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Farewell letter not found", err)
		}
		return Internal("Failed to fetch farewell letter", err)
	}
	if letter.Status == models.FarewellStatusSent {
		return BadRequest("Cannot delete an already-sent farewell letter", nil)
	}

	if err := farewellFileService.DeleteFarewellAttachmentsByLetterID(userID, id); err != nil {
		return Internal("Failed to delete farewell attachments", err)
	}

	if err := database.ForTenant(userID).Unscoped().Delete(&letter).Error; err != nil {
		return Internal("Failed to delete farewell letter", err)
	}

	return nil
}

func (s FarewellService) CancelPending(userID, messageID, id string) error {
	if err := requireFarewellMessageTriggered(userID, messageID); err != nil {
		return err
	}

	var letter models.FarewellLetter
	if err := database.ForTenant(userID).Where("message_id = ? AND id = ?", messageID, id).First(&letter).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Farewell letter not found", err)
		}
		return Internal("Failed to fetch farewell letter", err)
	}

	if letter.Status != models.FarewellStatusPending {
		return BadRequest("Only pending farewell letters can be canceled", nil)
	}

	var attachments []models.FarewellAttachment
	if err := database.ForTenant(userID).Where("letter_id = ?", id).Find(&attachments).Error; err != nil {
		return Internal("Failed to fetch farewell attachments", err)
	}

	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := database.TenantTx(tx, userID).Unscoped().
			Where("letter_id = ?", id).
			Delete(&models.FarewellAttachment{}).Error; err != nil {
			return Internal("Failed to delete farewell attachment records", err)
		}
		if err := database.TenantTx(tx, userID).Unscoped().Delete(&letter).Error; err != nil {
			return Internal("Failed to cancel farewell letter", err)
		}
		return nil
	}); err != nil {
		return err
	}

	for _, att := range attachments {
		if err := os.Remove(att.StoragePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove farewell attachment file", "path", att.StoragePath, "error", err)
		}
	}
	os.Remove(filepath.Join(farewellFileService.uploadsDir(), userID, "farewell", id))

	return nil
}

func (s FarewellService) CancelPendingByMessageID(userID, messageID string) (int64, error) {
	if err := requireFarewellMessageTriggered(userID, messageID); err != nil {
		return 0, err
	}

	var letters []models.FarewellLetter
	if err := database.ForTenant(userID).
		Where("message_id = ? AND status = ?", messageID, models.FarewellStatusPending).
		Find(&letters).Error; err != nil {
		return 0, Internal("Failed to fetch pending farewell letters", err)
	}

	if len(letters) == 0 {
		return 0, nil
	}

	letterIDs := make([]string, len(letters))
	for i, l := range letters {
		letterIDs[i] = l.ID
	}

	// Pre-fetch attachment records to collect file paths before any DB deletion.
	var attachments []models.FarewellAttachment
	if err := database.ForTenant(userID).
		Where("letter_id IN ?", letterIDs).
		Find(&attachments).Error; err != nil {
		return 0, Internal("Failed to fetch farewell attachments", err)
	}

	// Delete all DB records atomically so a partial failure cannot leave orphaned rows.
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := database.TenantTx(tx, userID).Unscoped().
			Where("letter_id IN ?", letterIDs).
			Delete(&models.FarewellAttachment{}).Error; err != nil {
			return Internal("Failed to delete farewell attachment records", err)
		}
		if err := database.TenantTx(tx, userID).Unscoped().
			Where("id IN ?", letterIDs).
			Delete(&models.FarewellLetter{}).Error; err != nil {
			return Internal("Failed to cancel farewell letters", err)
		}
		return nil
	}); err != nil {
		return 0, err
	}

	// Filesystem cleanup runs after a successful commit; errors are logged but not fatal.
	for _, att := range attachments {
		if err := os.Remove(att.StoragePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove farewell attachment file", "path", att.StoragePath, "error", err)
		}
	}
	uploadsDir := farewellFileService.uploadsDir()
	for _, id := range letterIDs {
		letterDir := filepath.Join(uploadsDir, userID, "farewell", id)
		os.Remove(letterDir)
	}

	return int64(len(letters)), nil
}

func loadFarewellMessage(userID, messageID string) (models.Message, error) {
	var msg models.Message
	if err := database.ForTenant(userID).First(&msg, "id = ?", messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Message{}, NotFound("Message not found", err)
		}
		return models.Message{}, Internal("Failed to fetch message", err)
	}
	return msg, nil
}

func requireFarewellMessageNotTriggered(userID, messageID, message string) error {
	msg, err := loadFarewellMessage(userID, messageID)
	if err != nil {
		return err
	}
	if msg.Status == models.StatusTriggered {
		return BadRequest(message, nil)
	}
	return nil
}

func requireFarewellMessageTriggered(userID, messageID string) error {
	msg, err := loadFarewellMessage(userID, messageID)
	if err != nil {
		return err
	}
	if msg.Status != models.StatusTriggered {
		return BadRequest(cancelRequiresTriggeredMessage, nil)
	}
	return nil
}
