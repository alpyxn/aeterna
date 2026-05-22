package services

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alpyxn/aeterna/backend/internal/config"
	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileService struct {
	cfg config.Config
}

func NewFileService(cfg config.Config) FileService {
	return FileService{cfg: cfg}
}

var fileCryptoService = CryptoService{}
var fileValidationService = ValidationService{}

func (s FileService) uploadsDir() string {
	return filepath.Join(filepath.Dir(s.cfg.Database.Path), "uploads")
}

// GetUploadsDir returns the base directory for file uploads given a database path.
func GetUploadsDir(dbPath string) string {
	return filepath.Join(filepath.Dir(dbPath), "uploads")
}

// EnsureUploadsDir creates the uploads directory if it does not exist.
func EnsureUploadsDir(dbPath string) error {
	return os.MkdirAll(GetUploadsDir(dbPath), 0700)
}

// Upload validates, encrypts, and stores a file on disk, then creates a DB record
func (s FileService) Upload(userID, messageID, filename, mimeType string, data []byte) (models.Attachment, error) {
	var msg models.Message
	if err := database.ForTenant(userID).First(&msg, "id = ?", messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Attachment{}, NotFound("Message not found", err)
		}
		return models.Attachment{}, Internal("Failed to fetch message", err)
	}

	if msg.Status == models.StatusTriggered {
		return models.Attachment{}, BadRequest("Cannot attach files to a triggered message", nil)
	}

	cleanFilename := fileValidationService.SanitizeFilename(filename)

	if err := fileValidationService.ValidateFile(cleanFilename, int64(len(data)), data); err != nil {
		return models.Attachment{}, err
	}

	var existingCount int64
	database.ForTenant(userID).Model(&models.Attachment{}).Where("message_id = ?", messageID).Count(&existingCount)
	if existingCount >= int64(MaxAttachmentsPerMsg) {
		return models.Attachment{}, BadRequest(fmt.Sprintf("Maximum %d attachments per message", MaxAttachmentsPerMsg), nil)
	}

	var totalSize int64
	database.ForTenant(userID).Model(&models.Attachment{}).Where("message_id = ?", messageID).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	if totalSize+int64(len(data)) > MaxTotalAttachSize {
		return models.Attachment{}, BadRequest("Total attachment size exceeds 25 MB limit", nil)
	}

	encrypted, err := fileCryptoService.EncryptBytes(data)
	if err != nil {
		return models.Attachment{}, Internal("Failed to encrypt file", err)
	}

	userDir := filepath.Join(s.uploadsDir(), userID)
	msgDir := filepath.Join(userDir, messageID)
	if err := os.MkdirAll(msgDir, 0700); err != nil {
		return models.Attachment{}, Internal("Failed to create upload directory", err)
	}

	storageFilename := uuid.NewString() + ".enc"
	storagePath := filepath.Join(msgDir, storageFilename)

	if err := os.WriteFile(storagePath, encrypted, 0600); err != nil {
		return models.Attachment{}, Internal("Failed to write file", err)
	}

	attachment := models.Attachment{
		UserID:      userID,
		MessageID:   messageID,
		Filename:    cleanFilename,
		StoragePath: storagePath,
		Size:        int64(len(data)),
		MimeType:    mimeType,
	}

	if err := database.ForTenant(userID).Create(&attachment).Error; err != nil {
		os.Remove(storagePath)
		return models.Attachment{}, Internal("Failed to save attachment record", err)
	}

	slog.Info("File uploaded", "attachment_id", attachment.ID, "message_id", messageID, "filename", cleanFilename, "size", len(data))
	return attachment, nil
}

// Delete removes a single attachment (file + DB record) scoped to the user
func (s FileService) Delete(userID, attachmentID string) error {
	var attachment models.Attachment
	if err := database.ForTenant(userID).First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Attachment not found", err)
		}
		return Internal("Failed to fetch attachment", err)
	}

	if err := os.Remove(attachment.StoragePath); err != nil && !os.IsNotExist(err) {
		slog.Error("Failed to remove attachment file", "path", attachment.StoragePath, "error", err)
	}

	if err := database.ForTenant(userID).Unscoped().Delete(&attachment).Error; err != nil {
		return Internal("Failed to delete attachment record", err)
	}

	slog.Info("File deleted", "attachment_id", attachmentID)
	return nil
}

// DeleteByMessageID removes all attachments for a message
func (s FileService) DeleteByMessageID(userID, messageID string) error {
	var attachments []models.Attachment
	if err := database.ForTenant(userID).Where("message_id = ?", messageID).Find(&attachments).Error; err != nil {
		return Internal("Failed to fetch attachments", err)
	}

	for _, att := range attachments {
		if err := os.Remove(att.StoragePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove attachment file", "path", att.StoragePath, "error", err)
		}
	}

	if err := database.ForTenant(userID).Unscoped().Where("message_id = ?", messageID).Delete(&models.Attachment{}).Error; err != nil {
		return Internal("Failed to delete attachment records", err)
	}

	msgDir := filepath.Join(s.uploadsDir(), userID, messageID)
	os.Remove(msgDir)

	slog.Info("All attachments deleted for message", "message_id", messageID)
	return nil
}

// GetDecrypted reads an encrypted file from disk for the given tenant
func (s FileService) GetDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error) {
	var attachment models.Attachment
	if err := database.ForTenant(userID).First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, NotFound("Attachment not found", err)
		}
		return "", "", nil, Internal("Failed to fetch attachment", err)
	}

	encrypted, err := os.ReadFile(attachment.StoragePath)
	if err != nil {
		return "", "", nil, Internal("Failed to read attachment file", err)
	}

	decrypted, err := fileCryptoService.DecryptBytes(encrypted)
	if err != nil {
		return "", "", nil, Internal("Failed to decrypt attachment", err)
	}

	return attachment.Filename, attachment.MimeType, decrypted, nil
}

// ListByMessageID returns all attachments for a message within a tenant
func (s FileService) ListByMessageID(userID, messageID string) ([]models.Attachment, error) {
	var attachments []models.Attachment
	if err := database.ForTenant(userID).Where("message_id = ?", messageID).Order("created_at ASC").Find(&attachments).Error; err != nil {
		return nil, Internal("Failed to fetch attachments", err)
	}
	return attachments, nil
}

// CountByMessageID returns the number of attachments for a message
func (s FileService) CountByMessageID(userID, messageID string) (int64, error) {
	var count int64
	if err := database.ForTenant(userID).Model(&models.Attachment{}).Where("message_id = ?", messageID).Count(&count).Error; err != nil {
		return 0, Internal("Failed to count attachments", err)
	}
	return count, nil
}

// UploadFarewellAttachment validates, encrypts, and stores a file for a farewell letter.
func (s FileService) UploadFarewellAttachment(userID, letterID, filename, mimeType string, data []byte) (models.FarewellAttachment, error) {
	var letter models.FarewellLetter
	if err := database.ForTenant(userID).First(&letter, "id = ?", letterID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.FarewellAttachment{}, NotFound("Farewell letter not found", err)
		}
		return models.FarewellAttachment{}, Internal("Failed to fetch farewell letter", err)
	}

	if letter.Status == models.FarewellStatusSent {
		return models.FarewellAttachment{}, BadRequest("Cannot attach files to an already-sent farewell letter", nil)
	}
	if err := requireFarewellMessageNotTriggered(userID, letter.MessageID, "Cannot modify farewell attachments after the switch has triggered"); err != nil {
		return models.FarewellAttachment{}, err
	}

	cleanFilename := fileValidationService.SanitizeFilename(filename)
	if err := fileValidationService.ValidateFarewellFile(cleanFilename, int64(len(data)), data); err != nil {
		return models.FarewellAttachment{}, err
	}

	var existingCount int64
	database.ForTenant(userID).Model(&models.FarewellAttachment{}).Where("letter_id = ?", letterID).Count(&existingCount)
	if existingCount >= int64(MaxFarewellAttachments) {
		return models.FarewellAttachment{}, BadRequest(fmt.Sprintf("Maximum %d attachments per farewell letter", MaxFarewellAttachments), nil)
	}

	var totalSize int64
	database.ForTenant(userID).Model(&models.FarewellAttachment{}).Where("letter_id = ?", letterID).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	if totalSize+int64(len(data)) > MaxFarewellTotalSize {
		return models.FarewellAttachment{}, BadRequest("Total attachment size exceeds 50 MB limit", nil)
	}

	encrypted, err := fileCryptoService.EncryptBytes(data)
	if err != nil {
		return models.FarewellAttachment{}, Internal("Failed to encrypt file", err)
	}

	letterDir := filepath.Join(s.uploadsDir(), userID, "farewell", letterID)
	if err := os.MkdirAll(letterDir, 0700); err != nil {
		return models.FarewellAttachment{}, Internal("Failed to create upload directory", err)
	}

	storageFilename := uuid.NewString() + ".enc"
	storagePath := filepath.Join(letterDir, storageFilename)

	if err := os.WriteFile(storagePath, encrypted, 0600); err != nil {
		return models.FarewellAttachment{}, Internal("Failed to write file", err)
	}

	attachment := models.FarewellAttachment{
		UserID:      userID,
		LetterID:    letterID,
		Filename:    cleanFilename,
		StoragePath: storagePath,
		Size:        int64(len(data)),
		MimeType:    mimeType,
	}

	if err := database.ForTenant(userID).Create(&attachment).Error; err != nil {
		os.Remove(storagePath)
		return models.FarewellAttachment{}, Internal("Failed to save attachment record", err)
	}

	slog.Info("Farewell attachment uploaded", "attachment_id", attachment.ID, "letter_id", letterID, "filename", cleanFilename)
	return attachment, nil
}

// ListFarewellAttachmentsByLetterID returns all attachments for a farewell letter.
func (s FileService) ListFarewellAttachmentsByLetterID(userID, letterID string) ([]models.FarewellAttachment, error) {
	var attachments []models.FarewellAttachment
	if err := database.ForTenant(userID).Where("letter_id = ?", letterID).Order("created_at ASC").Find(&attachments).Error; err != nil {
		return nil, Internal("Failed to fetch farewell attachments", err)
	}
	return attachments, nil
}

// CountFarewellAttachmentsByLetterID returns the number of attachments for a farewell letter.
func (s FileService) CountFarewellAttachmentsByLetterID(userID, letterID string) (int64, error) {
	var count int64
	if err := database.ForTenant(userID).Model(&models.FarewellAttachment{}).Where("letter_id = ?", letterID).Count(&count).Error; err != nil {
		return 0, Internal("Failed to count farewell attachments", err)
	}
	return count, nil
}

// DeleteFarewellAttachment removes a single farewell attachment (file + DB record).
func (s FileService) DeleteFarewellAttachment(userID, attachmentID string) error {
	var attachment models.FarewellAttachment
	if err := database.ForTenant(userID).First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Farewell attachment not found", err)
		}
		return Internal("Failed to fetch farewell attachment", err)
	}
	var letter models.FarewellLetter
	if err := database.ForTenant(userID).First(&letter, "id = ?", attachment.LetterID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Farewell letter not found", err)
		}
		return Internal("Failed to fetch farewell letter", err)
	}
	if err := requireFarewellMessageNotTriggered(userID, letter.MessageID, "Cannot modify farewell attachments after the switch has triggered"); err != nil {
		return err
	}

	if err := os.Remove(attachment.StoragePath); err != nil && !os.IsNotExist(err) {
		slog.Error("Failed to remove farewell attachment file", "path", attachment.StoragePath, "error", err)
	}

	if err := database.ForTenant(userID).Unscoped().Delete(&attachment).Error; err != nil {
		return Internal("Failed to delete farewell attachment record", err)
	}

	slog.Info("Farewell attachment deleted", "attachment_id", attachmentID)
	return nil
}

// DeleteFarewellAttachmentsByLetterID removes all attachments for a farewell letter.
func (s FileService) DeleteFarewellAttachmentsByLetterID(userID, letterID string) error {
	var attachments []models.FarewellAttachment
	if err := database.ForTenant(userID).Where("letter_id = ?", letterID).Find(&attachments).Error; err != nil {
		return Internal("Failed to fetch farewell attachments", err)
	}

	for _, att := range attachments {
		if err := os.Remove(att.StoragePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove farewell attachment file", "path", att.StoragePath, "error", err)
		}
	}

	if err := database.ForTenant(userID).Unscoped().Where("letter_id = ?", letterID).Delete(&models.FarewellAttachment{}).Error; err != nil {
		return Internal("Failed to delete farewell attachment records", err)
	}

	letterDir := filepath.Join(s.uploadsDir(), userID, "farewell", letterID)
	os.Remove(letterDir)

	slog.Info("All farewell attachments deleted", "letter_id", letterID)
	return nil
}

// GetFarewellAttachmentDecrypted reads and decrypts a farewell attachment.
func (s FileService) GetFarewellAttachmentDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error) {
	var attachment models.FarewellAttachment
	if err := database.ForTenant(userID).First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, NotFound("Farewell attachment not found", err)
		}
		return "", "", nil, Internal("Failed to fetch farewell attachment", err)
	}

	encrypted, err := os.ReadFile(attachment.StoragePath)
	if err != nil {
		return "", "", nil, Internal("Failed to read farewell attachment file", err)
	}

	decrypted, err := fileCryptoService.DecryptBytes(encrypted)
	if err != nil {
		return "", "", nil, Internal("Failed to decrypt farewell attachment", err)
	}

	return attachment.Filename, attachment.MimeType, decrypted, nil
}
