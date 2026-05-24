package services

import (
	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type NotifyingSettingsService struct {
	base     ports.SettingsServicePort
	notifier eventNotifier
}

func NewNotifyingSettingsService(base ports.SettingsServicePort, stream ports.EventStreamPort) ports.SettingsServicePort {
	return &NotifyingSettingsService{base: base, notifier: newEventNotifier(stream)}
}

func (s *NotifyingSettingsService) WithOriginSession(sessionKey string) ports.SettingsServicePort {
	return &NotifyingSettingsService{
		base:     s.base,
		notifier: s.notifier.withOriginSession(sessionKey),
	}
}

func (s *NotifyingSettingsService) Get(userID string) (models.Settings, error) {
	return s.base.Get(userID)
}

func (s *NotifyingSettingsService) GetByHeartbeatToken(token string) (models.Settings, error) {
	return s.base.GetByHeartbeatToken(token)
}

func (s *NotifyingSettingsService) Save(userID string, req models.Settings) error {
	err := s.base.Save(userID, req)
	if err == nil {
		s.notifier.publish(userID, ports.EventTypeSettingsChanged, ports.EventCodeSettingsSaved, "settings", "", "saved")
	}
	return err
}

func (s *NotifyingSettingsService) TestSMTP(req models.Settings) error {
	return s.base.TestSMTP(req)
}
