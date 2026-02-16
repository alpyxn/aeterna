package services

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/alpyxn/aeterna/backend/internal/database"
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FileService struct{}

var fileCryptoService = CryptoService{}
var fileValidationService = ValidationService{}

// GetUploadsDir returns the base directory for file uploads
func GetUploadsDir() string {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath != "" {
		return filepath.Join(filepath.Dir(dbPath), "uploads")
	}
	return filepath.Join(".", "data", "uploads")
}

// EnsureUploadsDir creates the uploads directory if it doesn't exist
func EnsureUploadsDir() error {
	dir := GetUploadsDir()
	return os.MkdirAll(dir, 0700)
}

// Upload validates, encrypts, and stores a file on disk, then creates a DB record
func (s FileService) Upload(messageID, filename, mimeType string, data []byte) (models.Attachment, error) {
	// Verify the message exists
	var msg models.Message
	if err := database.DB.First(&msg, "id = ?", messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return models.Attachment{}, NotFound("Message not found", err)
		}
		return models.Attachment{}, Internal("Failed to fetch message", err)
	}

	// Cannot attach to triggered messages
	if msg.Status == models.StatusTriggered {
		return models.Attachment{}, BadRequest("Cannot attach files to a triggered message", nil)
	}

	// Sanitize filename
	cleanFilename := fileValidationService.SanitizeFilename(filename)

	// Validate file
	if err := fileValidationService.ValidateFile(cleanFilename, int64(len(data)), data); err != nil {
		return models.Attachment{}, err
	}

	// Check attachment count limit
	var existingCount int64
	database.DB.Model(&models.Attachment{}).Where("message_id = ?", messageID).Count(&existingCount)
	if existingCount >= int64(MaxAttachmentsPerMsg) {
		return models.Attachment{}, BadRequest(fmt.Sprintf("Maximum %d attachments per message", MaxAttachmentsPerMsg), nil)
	}

	// Check total size limit
	var totalSize int64
	database.DB.Model(&models.Attachment{}).Where("message_id = ?", messageID).Select("COALESCE(SUM(size), 0)").Scan(&totalSize)
	if totalSize+int64(len(data)) > MaxTotalAttachSize {
		return models.Attachment{}, BadRequest("Total attachment size exceeds 25 MB limit", nil)
	}

	// Encrypt the file data
	encrypted, err := fileCryptoService.EncryptBytes(data)
	if err != nil {
		return models.Attachment{}, Internal("Failed to encrypt file", err)
	}

	// Create storage directory for this message
	msgDir := filepath.Join(GetUploadsDir(), messageID)
	if err := os.MkdirAll(msgDir, 0700); err != nil {
		return models.Attachment{}, Internal("Failed to create upload directory", err)
	}

	// Write encrypted file to disk with a random name
	storageFilename := uuid.NewString() + ".enc"
	storagePath := filepath.Join(msgDir, storageFilename)

	if err := os.WriteFile(storagePath, encrypted, 0600); err != nil {
		return models.Attachment{}, Internal("Failed to write file", err)
	}

	// Create DB record
	attachment := models.Attachment{
		MessageID:   messageID,
		Filename:    cleanFilename,
		StoragePath: storagePath,
		Size:        int64(len(data)),
		MimeType:    mimeType,
	}

	if err := database.DB.Create(&attachment).Error; err != nil {
		// Clean up the written file on DB failure
		os.Remove(storagePath)
		return models.Attachment{}, Internal("Failed to save attachment record", err)
	}

	slog.Info("File uploaded", "attachment_id", attachment.ID, "message_id", messageID, "filename", cleanFilename, "size", len(data))
	return attachment, nil
}

// Delete removes a single attachment (file + DB record)
func (s FileService) Delete(attachmentID string) error {
	var attachment models.Attachment
	if err := database.DB.First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NotFound("Attachment not found", err)
		}
		return Internal("Failed to fetch attachment", err)
	}

	// Remove file from disk
	if err := os.Remove(attachment.StoragePath); err != nil && !os.IsNotExist(err) {
		slog.Error("Failed to remove attachment file", "path", attachment.StoragePath, "error", err)
	}

	// Hard delete from DB
	if err := database.DB.Unscoped().Delete(&attachment).Error; err != nil {
		return Internal("Failed to delete attachment record", err)
	}

	slog.Info("File deleted", "attachment_id", attachmentID)
	return nil
}

// DeleteByMessageID removes all attachments for a message
func (s FileService) DeleteByMessageID(messageID string) error {
	var attachments []models.Attachment
	if err := database.DB.Where("message_id = ?", messageID).Find(&attachments).Error; err != nil {
		return Internal("Failed to fetch attachments", err)
	}

	for _, att := range attachments {
		if err := os.Remove(att.StoragePath); err != nil && !os.IsNotExist(err) {
			slog.Error("Failed to remove attachment file", "path", att.StoragePath, "error", err)
		}
	}

	// Hard delete all records
	if err := database.DB.Unscoped().Where("message_id = ?", messageID).Delete(&models.Attachment{}).Error; err != nil {
		return Internal("Failed to delete attachment records", err)
	}

	// Try to remove the message's upload directory (will fail if not empty, that's ok)
	msgDir := filepath.Join(GetUploadsDir(), messageID)
	os.Remove(msgDir)

	slog.Info("All attachments deleted for message", "message_id", messageID)
	return nil
}

// GetDecrypted reads an encrypted file from disk, decrypts it, and returns the plaintext data
func (s FileService) GetDecrypted(attachmentID string) (filename, mimeType string, data []byte, err error) {
	var attachment models.Attachment
	if err := database.DB.First(&attachment, "id = ?", attachmentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", nil, NotFound("Attachment not found", err)
		}
		return "", "", nil, Internal("Failed to fetch attachment", err)
	}

	// Read encrypted file from disk
	encrypted, err := os.ReadFile(attachment.StoragePath)
	if err != nil {
		return "", "", nil, Internal("Failed to read attachment file", err)
	}

	// Decrypt
	decrypted, err := fileCryptoService.DecryptBytes(encrypted)
	if err != nil {
		return "", "", nil, Internal("Failed to decrypt attachment", err)
	}

	return attachment.Filename, attachment.MimeType, decrypted, nil
}

// ListByMessageID returns all attachments for a given message
func (s FileService) ListByMessageID(messageID string) ([]models.Attachment, error) {
	var attachments []models.Attachment
	if err := database.DB.Where("message_id = ?", messageID).Order("created_at ASC").Find(&attachments).Error; err != nil {
		return nil, Internal("Failed to fetch attachments", err)
	}
	return attachments, nil
}

// CountByMessageID returns the number of attachments for a message
func (s FileService) CountByMessageID(messageID string) (int64, error) {
	var count int64
	if err := database.DB.Model(&models.Attachment{}).Where("message_id = ?", messageID).Count(&count).Error; err != nil {
		return 0, Internal("Failed to count attachments", err)
	}
	return count, nil
}
