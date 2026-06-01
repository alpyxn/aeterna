package services

import (
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/models"
	"github.com/alpyxn/aeterna/backend/internal/ports"
)

type realtimeE2EMessageService struct{}

func (s realtimeE2EMessageService) Create(userID, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	return models.Message{ID: "msg-e2e", UserID: userID, LastSeen: time.Now().UTC(), Status: models.StatusActive}, nil
}

func (s realtimeE2EMessageService) GetPublicByID(id string) (models.Message, error) {
	return models.Message{ID: id, LastSeen: time.Now().UTC(), Status: models.StatusActive}, nil
}

func (s realtimeE2EMessageService) GetByID(userID, id string) (models.Message, error) {
	return models.Message{ID: id, UserID: userID, LastSeen: time.Now().UTC(), Status: models.StatusActive}, nil
}

func (s realtimeE2EMessageService) List(userID string) ([]models.Message, error) {
	return []models.Message{}, nil
}

func (s realtimeE2EMessageService) Heartbeat(userID, id string) (models.Message, error) {
	return models.Message{ID: id, UserID: userID, LastSeen: time.Now().UTC(), Status: models.StatusActive}, nil
}

func (s realtimeE2EMessageService) BulkHeartbeat(userID string) error { return nil }

func (s realtimeE2EMessageService) Delete(userID, id string) error { return nil }

func (s realtimeE2EMessageService) Update(userID, id, content string, recipientEmails []string, triggerDuration int, reminders []int) (models.Message, error) {
	return models.Message{ID: id, UserID: userID, LastSeen: time.Now().UTC(), Status: models.StatusActive}, nil
}

func TestRealtimeEventsE2E_HeartbeatBroadcastsToAllDevicesOfSameUser(t *testing.T) {
	stream := NewEventStreamService()
	svc := NewNotifyingMessageService(realtimeE2EMessageService{}, stream)

	webCh, _, webCancel, err := stream.Subscribe("u1", "web", "sess-web")
	if err != nil {
		t.Fatalf("subscribe web failed: %v", err)
	}
	defer webCancel()

	androidCh, _, androidCancel, err := stream.Subscribe("u1", "android", "sess-android")
	if err != nil {
		t.Fatalf("subscribe android failed: %v", err)
	}
	defer androidCancel()

	if _, err := svc.Heartbeat("u1", "msg-1"); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	waitForRealtimeEventType(t, webCh, ports.EventTypeMessagesChanged, 2*time.Second)
	waitForRealtimeEventType(t, androidCh, ports.EventTypeMessagesChanged, 2*time.Second)
}

func TestRealtimeEventsE2E_HeartbeatDoesNotLeakAcrossUsers(t *testing.T) {
	stream := NewEventStreamService()
	svc := NewNotifyingMessageService(realtimeE2EMessageService{}, stream)

	user1Ch, _, cancel1, err := stream.Subscribe("u1", "web", "sess-u1")
	if err != nil {
		t.Fatalf("subscribe u1 failed: %v", err)
	}
	defer cancel1()

	user2Ch, _, cancel2, err := stream.Subscribe("u2", "android", "sess-u2")
	if err != nil {
		t.Fatalf("subscribe u2 failed: %v", err)
	}
	defer cancel2()

	if _, err := svc.Heartbeat("u1", "msg-1"); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	waitForRealtimeEventType(t, user1Ch, ports.EventTypeMessagesChanged, 2*time.Second)
	assertNoRealtimeEventType(t, user2Ch, ports.EventTypeMessagesChanged, 500*time.Millisecond)
}

func TestRealtimeEventsE2E_HeartbeatSkipsOriginSession(t *testing.T) {
	stream := NewEventStreamService()
	base := NewNotifyingMessageService(realtimeE2EMessageService{}, stream)

	scoped, ok := base.(interface {
		WithOriginSession(sessionKey string) ports.MessageServicePort
	})
	if !ok {
		t.Fatal("notifying message service is expected to support origin session scoping")
	}

	originCh, _, originCancel, err := stream.Subscribe("u1", "web", "sess-origin")
	if err != nil {
		t.Fatalf("subscribe origin failed: %v", err)
	}
	defer originCancel()

	otherCh, _, otherCancel, err := stream.Subscribe("u1", "android", "sess-other")
	if err != nil {
		t.Fatalf("subscribe other failed: %v", err)
	}
	defer otherCancel()

	if _, err := scoped.WithOriginSession("sess-origin").Heartbeat("u1", "msg-1"); err != nil {
		t.Fatalf("heartbeat failed: %v", err)
	}

	assertNoRealtimeEventType(t, originCh, ports.EventTypeMessagesChanged, 500*time.Millisecond)
	waitForRealtimeEventType(t, otherCh, ports.EventTypeMessagesChanged, 2*time.Second)
}

func waitForRealtimeEventType(t *testing.T, ch <-chan ports.RealtimeEvent, eventType string, timeout time.Duration) {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case evt := <-ch:
			if evt.Type == eventType {
				return
			}
		case <-timer.C:
			t.Fatalf("did not receive event type %q within %s", eventType, timeout)
		}
	}
}

func assertNoRealtimeEventType(t *testing.T, ch <-chan ports.RealtimeEvent, eventType string, duration time.Duration) {
	t.Helper()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case evt := <-ch:
			if evt.Type == eventType {
				t.Fatalf("unexpected event type %q received", eventType)
			}
		case <-timer.C:
			return
		}
	}
}
