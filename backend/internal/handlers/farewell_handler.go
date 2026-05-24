package handlers

import (
	"io"

	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/alpyxn/aeterna/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// FarewellHandlers groups farewell letter and farewell attachment route handlers.
type FarewellHandlers struct {
	farewell ports.FarewellServicePort
	files    ports.FileServicePort
}

type farewellPayload struct {
	RecipientEmail string `json:"recipient_email"`
	Subject        string `json:"subject"`
	Content        string `json:"content"`
	DelayMinutes   *int   `json:"delay_minutes"`
}

func parseFarewellPayload(c *fiber.Ctx) (farewellPayload, error) {
	var body farewellPayload
	if err := c.BodyParser(&body); err != nil {
		return farewellPayload{}, services.BadRequest("Invalid request body", err)
	}
	if body.DelayMinutes == nil {
		return farewellPayload{}, services.BadRequest("delay_minutes is required and must be a number", nil)
	}
	return body, nil
}

func NewFarewellHandlers(farewell ports.FarewellServicePort, files ports.FileServicePort) *FarewellHandlers {
	return &FarewellHandlers{farewell: farewell, files: files}
}

func (h *FarewellHandlers) List(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	messageID := c.Params("id")

	letters, err := h.farewell.List(userID, messageID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(letters)
}

func (h *FarewellHandlers) Create(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	farewell := withOriginSession(c, h.farewell)
	messageID := c.Params("id")

	body, err := parseFarewellPayload(c)
	if err != nil {
		return writeError(c, err)
	}

	letter, err := farewell.Create(
		userID,
		messageID,
		body.RecipientEmail,
		body.Subject,
		body.Content,
		*body.DelayMinutes,
	)
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(letter)
}

func (h *FarewellHandlers) Update(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	farewell := withOriginSession(c, h.farewell)
	messageID := c.Params("id")
	letterID := c.Params("letterId")

	body, err := parseFarewellPayload(c)
	if err != nil {
		return writeError(c, err)
	}

	letter, err := farewell.Update(
		userID,
		messageID,
		letterID,
		body.RecipientEmail,
		body.Subject,
		body.Content,
		*body.DelayMinutes,
	)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(letter)
}

func (h *FarewellHandlers) Delete(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	farewell := withOriginSession(c, h.farewell)
	messageID := c.Params("id")
	letterID := c.Params("letterId")

	if err := farewell.Delete(userID, messageID, letterID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Farewell letter deleted"})
}

func (h *FarewellHandlers) CancelPending(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	farewell := withOriginSession(c, h.farewell)
	messageID := c.Params("id")
	letterID := c.Params("letterId")

	if err := farewell.CancelPending(userID, messageID, letterID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Pending farewell letter canceled"})
}

func (h *FarewellHandlers) CancelAllPending(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	farewell := withOriginSession(c, h.farewell)
	messageID := c.Params("id")

	count, err := farewell.CancelPendingByMessageID(userID, messageID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "canceled": count})
}

func (h *FarewellHandlers) UploadAttachment(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	files := withOriginSession(c, h.files)
	letterID := c.Params("letterId")

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

	attachment, err := files.UploadFarewellAttachment(userID, letterID, fileHeader.Filename, mimeType, data)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "attachment": attachment})
}

func (h *FarewellHandlers) ListAttachments(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	letterID := c.Params("letterId")

	attachments, err := h.files.ListFarewellAttachmentsByLetterID(userID, letterID)
	if err != nil {
		return writeError(c, err)
	}

	return c.JSON(attachments)
}

func (h *FarewellHandlers) DeleteAttachment(c *fiber.Ctx) error {
	userID, err := currentUserID(c)
	if err != nil {
		return writeError(c, err)
	}
	files := withOriginSession(c, h.files)
	attachmentID := c.Params("attachmentId")

	if err := files.DeleteFarewellAttachment(userID, attachmentID); err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"success": true, "message": "Farewell attachment deleted"})
}
