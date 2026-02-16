package handlers

import (
	"io"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var fileService = services.FileService{}

// UploadAttachment handles file upload for a message
func UploadAttachment(c *fiber.Ctx) error {
	messageID := c.Params("id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return writeError(c, services.BadRequest("No file provided", err))
	}

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return writeError(c, services.BadRequest("Failed to read uploaded file", err))
	}
	defer file.Close()

	// Read file data into memory
	data, err := io.ReadAll(file)
	if err != nil {
		return writeError(c, services.BadRequest("Failed to read file data", err))
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	attachment, err := fileService.Upload(messageID, fileHeader.Filename, mimeType, data)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"attachment": attachment,
	})
}

// ListAttachments returns all attachments for a message
func ListAttachments(c *fiber.Ctx) error {
	messageID := c.Params("id")

	attachments, err := fileService.ListByMessageID(messageID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(attachments)
}

// DeleteAttachment removes a single attachment
func DeleteAttachment(c *fiber.Ctx) error {
	attachmentID := c.Params("attachmentId")

	if err := fileService.Delete(attachmentID); err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Attachment deleted successfully",
	})
}
