package services

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type NotifyingFileService struct {
	base     ports.FileServicePort
	notifier eventNotifier
}

func NewNotifyingFileService(base ports.FileServicePort, stream ports.EventStreamPort) ports.FileServicePort {
	return &NotifyingFileService{base: base, notifier: newEventNotifier(stream)}
}

func (s *NotifyingFileService) WithOriginSession(sessionKey string) ports.FileServicePort {
	return &NotifyingFileService{
		base:     s.base,
		notifier: s.notifier.withOriginSession(sessionKey),
	}
}

func (s *NotifyingFileService) Upload(userID, messageID, filename, mimeType string, data []byte) (models.Attachment, error) {
	attachment, err := s.base.Upload(userID, messageID, filename, mimeType, data)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeAttachmentsChanged, ports.EventCodeAttachmentUploaded, "attachment", attachment.ID, "uploaded")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageAttachmentUploaded, "message", messageID, "attachment_uploaded")
	}
	return attachment, err
}

func (s *NotifyingFileService) Delete(userID, attachmentID string) error {
	err := s.base.Delete(userID, attachmentID)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeAttachmentsChanged, ports.EventCodeAttachmentDeleted, "attachment", attachmentID, "deleted")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageAttachmentDeleted, "attachment", attachmentID, "attachment_deleted")
	}
	return err
}

func (s *NotifyingFileService) DeleteByMessageID(userID, messageID string) error {
	return s.base.DeleteByMessageID(userID, messageID)
}

func (s *NotifyingFileService) GetDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error) {
	return s.base.GetDecrypted(userID, attachmentID)
}

func (s *NotifyingFileService) ListByMessageID(userID, messageID string) ([]models.Attachment, error) {
	return s.base.ListByMessageID(userID, messageID)
}

func (s *NotifyingFileService) CountByMessageID(userID, messageID string) (int64, error) {
	return s.base.CountByMessageID(userID, messageID)
}

func (s *NotifyingFileService) UploadFarewellAttachment(userID, letterID, filename, mimeType string, data []byte) (models.FarewellAttachment, error) {
	attachment, err := s.base.UploadFarewellAttachment(userID, letterID, filename, mimeType, data)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellAttachmentUploaded, "farewell_attachment", attachment.ID, "attachment_uploaded")
	}
	return attachment, err
}

func (s *NotifyingFileService) ListFarewellAttachmentsByLetterID(userID, letterID string) ([]models.FarewellAttachment, error) {
	return s.base.ListFarewellAttachmentsByLetterID(userID, letterID)
}

func (s *NotifyingFileService) CountFarewellAttachmentsByLetterID(userID, letterID string) (int64, error) {
	return s.base.CountFarewellAttachmentsByLetterID(userID, letterID)
}

func (s *NotifyingFileService) DeleteFarewellAttachment(userID, attachmentID string) error {
	err := s.base.DeleteFarewellAttachment(userID, attachmentID)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellAttachmentDeleted, "farewell_attachment", attachmentID, "attachment_deleted")
	}
	return err
}

func (s *NotifyingFileService) DeleteFarewellAttachmentsByLetterID(userID, letterID string) error {
	return s.base.DeleteFarewellAttachmentsByLetterID(userID, letterID)
}

func (s *NotifyingFileService) GetFarewellAttachmentDecrypted(userID, attachmentID string) (filename, mimeType string, data []byte, err error) {
	return s.base.GetFarewellAttachmentDecrypted(userID, attachmentID)
}
