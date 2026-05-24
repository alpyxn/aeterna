package services

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type NotifyingFarewellService struct {
	base     ports.FarewellServicePort
	notifier eventNotifier
}

func NewNotifyingFarewellService(base ports.FarewellServicePort, stream ports.EventStreamPort) ports.FarewellServicePort {
	return &NotifyingFarewellService{base: base, notifier: newEventNotifier(stream)}
}

func (s *NotifyingFarewellService) WithOriginSession(sessionKey string) ports.FarewellServicePort {
	return &NotifyingFarewellService{
		base:     s.base,
		notifier: s.notifier.withOriginSession(sessionKey),
	}
}

func (s *NotifyingFarewellService) Create(userID, messageID, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error) {
	letter, err := s.base.Create(userID, messageID, recipientEmail, subject, content, delayMinutes)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellCreated, "farewell", letter.ID, "created")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageFarewellCreated, "message", messageID, "farewell_created")
	}
	return letter, err
}

func (s *NotifyingFarewellService) List(userID, messageID string) ([]models.FarewellLetter, error) {
	return s.base.List(userID, messageID)
}

func (s *NotifyingFarewellService) Update(userID, messageID, id, recipientEmail, subject, content string, delayMinutes int) (models.FarewellLetter, error) {
	letter, err := s.base.Update(userID, messageID, id, recipientEmail, subject, content, delayMinutes)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellUpdated, "farewell", letter.ID, "updated")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageFarewellUpdated, "message", messageID, "farewell_updated")
	}
	return letter, err
}

func (s *NotifyingFarewellService) Delete(userID, messageID, id string) error {
	err := s.base.Delete(userID, messageID, id)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellDeleted, "farewell", id, "deleted")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageFarewellDeleted, "message", messageID, "farewell_deleted")
	}
	return err
}

func (s *NotifyingFarewellService) CancelPending(userID, messageID, id string) error {
	err := s.base.CancelPending(userID, messageID, id)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellDeleted, "farewell", id, "canceled")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageFarewellDeleted, "message", messageID, "farewell_canceled")
	}
	return err
}

func (s *NotifyingFarewellService) CancelPendingByMessageID(userID, messageID string) (int64, error) {
	count, err := s.base.CancelPendingByMessageID(userID, messageID)
	if err == nil && count > 0 {
		s.notifier.publish(userID, ports.EventTypeFarewellsChanged, ports.EventCodeFarewellDeleted, "farewell", "", "canceled_all")
		s.notifier.publish(userID, ports.EventTypeMessagesChanged, ports.EventCodeMessageFarewellDeleted, "message", messageID, "farewells_canceled")
	}
	return count, err
}
