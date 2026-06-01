package services

import (
	"fmt"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type NotifyingWebhookStore struct {
	base     ports.WebhookStorePort
	notifier eventNotifier
}

func NewNotifyingWebhookStore(base ports.WebhookStorePort, stream ports.EventStreamPort) ports.WebhookStorePort {
	return &NotifyingWebhookStore{base: base, notifier: newEventNotifier(stream)}
}

func (s *NotifyingWebhookStore) WithOriginSession(sessionKey string) ports.WebhookStorePort {
	return &NotifyingWebhookStore{
		base:     s.base,
		notifier: s.notifier.withOriginSession(sessionKey),
	}
}

func (s *NotifyingWebhookStore) List(userID string) ([]models.Webhook, error) {
	return s.base.List(userID)
}

func (s *NotifyingWebhookStore) ListEnabledForUser(userID string) ([]models.Webhook, error) {
	return s.base.ListEnabledForUser(userID)
}

func (s *NotifyingWebhookStore) Create(userID string, item models.Webhook) (models.Webhook, error) {
	created, err := s.base.Create(userID, item)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeWebhooksChanged, ports.EventCodeWebhookCreated, "webhook", fmt.Sprint(created.ID), "created")
	}
	return created, err
}

func (s *NotifyingWebhookStore) Update(userID, id string, input models.Webhook) (models.Webhook, error) {
	updated, err := s.base.Update(userID, id, input)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeWebhooksChanged, ports.EventCodeWebhookUpdated, "webhook", fmt.Sprint(updated.ID), "updated")
	}
	return updated, err
}

func (s *NotifyingWebhookStore) Delete(userID, id string) error {
	err := s.base.Delete(userID, id)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeWebhooksChanged, ports.EventCodeWebhookDeleted, "webhook", id, "deleted")
	}
	return err
}
