package services

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/ports"
)

func newTestEventStreamService(maxGlobal, maxPerUser, buffer int) *EventStreamService {
	return &EventStreamService{
		clients:               make(map[string]map[string]*eventClient),
		maxConnectionsGlobal:  maxGlobal,
		maxConnectionsPerUser: maxPerUser,
		clientBufferSize:      buffer,
	}
}

func TestEventStreamService_EnforcesPerUserLimit(t *testing.T) {
	svc := newTestEventStreamService(10, 1, 4)

	_, _, cancel1, err := svc.Subscribe("u1", "c1", "")
	if err != nil {
		t.Fatalf("unexpected error on first subscribe: %v", err)
	}
	defer cancel1()

	_, _, _, err = svc.Subscribe("u1", "c2", "")
	if err == nil {
		t.Fatal("expected per-user limit error, got nil")
	}
}

func TestEventStreamService_EnforcesGlobalLimit(t *testing.T) {
	svc := newTestEventStreamService(1, 5, 4)

	_, _, cancel1, err := svc.Subscribe("u1", "c1", "")
	if err != nil {
		t.Fatalf("unexpected error on first subscribe: %v", err)
	}
	defer cancel1()

	_, _, _, err = svc.Subscribe("u2", "c2", "")
	if err == nil {
		t.Fatal("expected global limit error, got nil")
	}
}

func TestEventStreamService_DuplicateClientIDStopsPreviousStream(t *testing.T) {
	svc := newTestEventStreamService(10, 5, 4)

	_, doneOld, cancelOld, err := svc.Subscribe("u1", "same-client", "")
	if err != nil {
		t.Fatalf("unexpected error on first subscribe: %v", err)
	}
	defer cancelOld()

	_, doneNew, cancelNew, err := svc.Subscribe("u1", "same-client", "")
	if err != nil {
		t.Fatalf("unexpected error on second subscribe: %v", err)
	}
	defer cancelNew()

	select {
	case <-doneOld:
		// expected
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected previous stream to be stopped on duplicate client id")
	}

	select {
	case <-doneNew:
		t.Fatal("new stream should remain active")
	default:
	}
}

func TestEventStreamService_SlowConsumerIsDisconnected(t *testing.T) {
	svc := newTestEventStreamService(10, 5, 1)

	_, done, cancel, err := svc.Subscribe("u1", "slow", "")
	if err != nil {
		t.Fatalf("unexpected error on subscribe: %v", err)
	}
	defer cancel()

	svc.Publish("u1", ports.RealtimeEvent{Type: ports.EventTypeMessagesChanged})
	svc.Publish("u1", ports.RealtimeEvent{Type: ports.EventTypeMessagesChanged})

	select {
	case <-done:
		// expected disconnect
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected slow consumer to be disconnected")
	}
}

func TestEventStreamService_PublishAndUnsubscribeConcurrently_NoPanic(t *testing.T) {
	svc := newTestEventStreamService(100, 100, 4)

	const total = 50
	cancels := make([]func(), 0, total)
	for i := 0; i < total; i++ {
		_, _, cancel, err := svc.Subscribe("u1", fmt.Sprintf("c-%d", i), "")
		if err != nil {
			t.Fatalf("subscribe failed: %v", err)
		}
		cancels = append(cancels, cancel)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			svc.Publish("u1", ports.RealtimeEvent{Type: ports.EventTypePing})
		}
	}()

	go func() {
		defer wg.Done()
		for _, cancel := range cancels {
			cancel()
		}
	}()

	wg.Wait()
}

func TestEventStreamService_ExcludesOriginSession(t *testing.T) {
	svc := newTestEventStreamService(10, 10, 4)

	originCh, _, cancelOrigin, err := svc.Subscribe("u1", "origin-client", "sess-origin")
	if err != nil {
		t.Fatalf("subscribe origin failed: %v", err)
	}
	defer cancelOrigin()

	otherCh, _, cancelOther, err := svc.Subscribe("u1", "other-client", "sess-other")
	if err != nil {
		t.Fatalf("subscribe other failed: %v", err)
	}
	defer cancelOther()

	svc.Publish("u1", ports.RealtimeEvent{
		Type:             ports.EventTypeMessagesChanged,
		OriginSessionKey: "sess-origin",
	})

	select {
	case evt := <-originCh:
		if evt.Type == ports.EventTypeMessagesChanged {
			t.Fatal("origin session should not receive its own event")
		}
	case <-time.After(300 * time.Millisecond):
	}

	select {
	case evt := <-otherCh:
		if evt.Type != ports.EventTypeMessagesChanged {
			t.Fatalf("expected %q, got %q", ports.EventTypeMessagesChanged, evt.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("other session did not receive event")
	}
}
