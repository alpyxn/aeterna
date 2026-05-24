package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/alpyxn/aeterna/backend/internal/config/common"
	"github.com/alpyxn/aeterna/backend/internal/ports"
	"github.com/google/uuid"
)

const (
	defaultSSEMaxConnectionsGlobal  = 2000
	defaultSSEMaxConnectionsPerUser = 5
	defaultSSEClientBufferSize      = 32
)

type eventClient struct {
	id         string
	userID     string
	sessionKey string
	ch         chan ports.RealtimeEvent
	done       chan struct{}
	createdAt  time.Time
	stopOnce   sync.Once
}

func (c *eventClient) stop() {
	c.stopOnce.Do(func() {
		close(c.done)
	})
}

// EventStreamService is an in-memory user-scoped real-time event hub.
type EventStreamService struct {
	mu                    sync.RWMutex
	clients               map[string]map[string]*eventClient // userID -> clientID -> client
	totalConnections      int
	maxConnectionsGlobal  int
	maxConnectionsPerUser int
	clientBufferSize      int
}

func NewEventStreamService() *EventStreamService {
	return &EventStreamService{
		clients:               make(map[string]map[string]*eventClient),
		maxConnectionsGlobal:  common.GetPositiveInt("SSE_MAX_CONNECTIONS_GLOBAL", defaultSSEMaxConnectionsGlobal),
		maxConnectionsPerUser: common.GetPositiveInt("SSE_MAX_CONNECTIONS_PER_USER", defaultSSEMaxConnectionsPerUser),
		clientBufferSize:      common.GetPositiveInt("SSE_CLIENT_BUFFER_SIZE", defaultSSEClientBufferSize),
	}
}

func (s *EventStreamService) Subscribe(userID, clientID, sessionKey string) (<-chan ports.RealtimeEvent, <-chan struct{}, func(), error) {
	if userID == "" {
		return nil, nil, nil, fmt.Errorf("user id is required")
	}
	if clientID == "" {
		clientID = uuid.NewString()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.totalConnections >= s.maxConnectionsGlobal {
		return nil, nil, nil, fmt.Errorf("too many active event streams")
	}

	userClients := s.clients[userID]
	if userClients == nil {
		userClients = make(map[string]*eventClient)
		s.clients[userID] = userClients
	}

	if existing, ok := userClients[clientID]; ok {
		delete(userClients, clientID)
		s.totalConnections--
		existing.stop()
	}

	if len(userClients) >= s.maxConnectionsPerUser {
		return nil, nil, nil, fmt.Errorf("too many active event streams for user")
	}

	client := &eventClient{
		id:         clientID,
		userID:     userID,
		sessionKey: sessionKey,
		ch:         make(chan ports.RealtimeEvent, s.clientBufferSize),
		done:       make(chan struct{}),
		createdAt:  time.Now().UTC(),
	}
	userClients[clientID] = client
	s.totalConnections++

	cancel := func() {
		s.unsubscribe(userID, clientID)
	}

	return client.ch, client.done, cancel, nil
}

func (s *EventStreamService) Publish(userID string, event ports.RealtimeEvent) {
	if userID == "" {
		return
	}

	s.mu.RLock()
	userClients := s.clients[userID]
	targets := make([]*eventClient, 0, len(userClients))
	for _, client := range userClients {
		targets = append(targets, client)
	}
	s.mu.RUnlock()

	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	for _, client := range targets {
		if event.OriginSessionKey != "" && client.sessionKey == event.OriginSessionKey {
			continue
		}
		select {
		case client.ch <- event:
		default:
			// Slow consumers are disconnected to protect memory.
			s.unsubscribe(client.userID, client.id)
		}
	}
}

func (s *EventStreamService) unsubscribe(userID, clientID string) {
	if userID == "" || clientID == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userClients := s.clients[userID]
	if userClients == nil {
		return
	}
	client, ok := userClients[clientID]
	if !ok {
		return
	}

	delete(userClients, clientID)
	if len(userClients) == 0 {
		delete(s.clients, userID)
	}
	s.totalConnections--
	client.stop()
}
