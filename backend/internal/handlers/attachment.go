package handlers

import (
	"io"

	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

var fileService = services.FileService{}

func UploadAttachment(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	messageID := c.Params("id")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return writeError(c, services.BadRequest("No file provided", err))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return writeError(c, services.BadRequest("Failed to read uploaded file", err))
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return writeError(c, services.BadRequest("Failed to read file data", err))
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	attachment, err := fileService.Upload(userID, messageID, fileHeader.Filename, mimeType, data)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success":    true,
		"attachment": attachment,
	})
}

func ListAttachments(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	messageID := c.Params("id")

	attachments, err := fileService.ListByMessageID(userID, messageID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(attachments)
}

func DeleteAttachment(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	attachmentID := c.Params("attachmentId")

	if err := fileService.Delete(userID, attachmentID); err != nil {
		return writeError(c, err)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Attachment deleted successfully",
	})
}
