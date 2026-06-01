package services

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type NotifyingMessageService struct {
	base     ports.MessageServicePort
	notifier eventNotifier
}

func NewNotifyingMessageService(base ports.MessageServicePort, stream ports.EventStreamPort) ports.MessageServicePort {
	return &NotifyingMessageService{base: base, notifier: newEventNotifier(stream)}
}

func (s *NotifyingMessageService) WithOriginSession(sessionKey string) ports.MessageServicePort {
	return &NotifyingMessageService{
		base:     s.base,
		notifier: s.notifier.withOriginSession(sessionKey),
	}
}

func (s *NotifyingMessageService) Create(userID, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	msg, err := s.base.Create(userID, content, recipientEmails, triggerDuration, reminders)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageCreated, "message", msg.ID, "created")
	}
	return msg, err
}

func (s *NotifyingMessageService) GetPublicByID(id string) (models.Message, error) {
	return s.base.GetPublicByID(id)
}

func (s *NotifyingMessageService) GetByID(userID, id string) (models.Message, error) {
	return s.base.GetByID(userID, id)
}

func (s *NotifyingMessageService) List(userID string) ([]models.Message, error) {
	return s.base.List(userID)
}

func (s *NotifyingMessageService) Heartbeat(userID, id string) (models.Message, error) {
	msg, err := s.base.Heartbeat(userID, id)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageHeartbeat, "message", msg.ID, "heartbeat")
	}
	return msg, err
}

func (s *NotifyingMessageService) BulkHeartbeat(userID string) error {
	err := s.base.BulkHeartbeat(userID)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageBulkHeartbeat, "message", "", "bulk_heartbeat")
	}
	return err
}

func (s *NotifyingMessageService) Delete(userID, id string) error {
	err := s.base.Delete(userID, id)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageDeleted, "message", id, "deleted")
	}
	return err
}

func (s *NotifyingMessageService) Update(userID, id, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	msg, err := s.base.Update(userID, id, content, recipientEmails, triggerDuration, reminders)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageUpdated, "message", msg.ID, "updated")
	}
	return msg, err
}
