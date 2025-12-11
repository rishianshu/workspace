// Package memory provides the three-tier memory system
package memory

import (
	"context"
	"sync"
	"time"
)

// Store is the interface for memory storage
type Store interface {
	Get(ctx context.Context, key string) (map[string]any, error)
	Set(ctx context.Context, key string, value map[string]any, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Search(ctx context.Context, query string, limit int) ([]map[string]any, error)
}

// ShortTermStore implements in-memory storage with TTL (like Redis)
type ShortTermStore struct {
	mu    sync.RWMutex
	data  map[string]entry
	stop  chan struct{}
}

type entry struct {
	value     map[string]any
	expiresAt time.Time
}

// NewShortTermStore creates a new in-memory store
func NewShortTermStore() *ShortTermStore {
	s := &ShortTermStore{
		data: make(map[string]entry),
		stop: make(chan struct{}),
	}
	go s.cleanup()
	return s
}

// Get retrieves a value by key
func (s *ShortTermStore) Get(ctx context.Context, key string) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	e, ok := s.data[key]
	if !ok || (!e.expiresAt.IsZero() && time.Now().After(e.expiresAt)) {
		return nil, nil
	}
	return e.value, nil
}

// Set stores a value with optional TTL
func (s *ShortTermStore) Set(ctx context.Context, key string, value map[string]any, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	s.data[key] = entry{
		value:     value,
		expiresAt: expiresAt,
	}
	return nil
}

// Delete removes a key
func (s *ShortTermStore) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
	return nil
}

// Search is not implemented for short-term store
func (s *ShortTermStore) Search(ctx context.Context, query string, limit int) ([]map[string]any, error) {
	// Short-term store doesn't support semantic search
	return nil, nil
}

// Close stops the cleanup goroutine
func (s *ShortTermStore) Close() {
	close(s.stop)
}

func (s *ShortTermStore) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for k, e := range s.data {
				if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
					delete(s.data, k)
				}
			}
			s.mu.Unlock()
		case <-s.stop:
			return
		}
	}
}

// SessionState represents the current state of a conversation session
type SessionState struct {
	SessionID      string         `json:"session_id"`
	ConversationID string         `json:"conversation_id"`
	UserID         string         `json:"user_id"`
	Intent         string         `json:"intent"`
	Entities       []string       `json:"entities"`
	Scratchpad     string         `json:"scratchpad"`
	TurnCount      int            `json:"turn_count"`
	LastUpdated    time.Time      `json:"last_updated"`
	Metadata       map[string]any `json:"metadata"`
}

// ToMap converts SessionState to a map for storage
func (s *SessionState) ToMap() map[string]any {
	return map[string]any{
		"session_id":      s.SessionID,
		"conversation_id": s.ConversationID,
		"user_id":         s.UserID,
		"intent":          s.Intent,
		"entities":        s.Entities,
		"scratchpad":      s.Scratchpad,
		"turn_count":      s.TurnCount,
		"last_updated":    s.LastUpdated,
		"metadata":        s.Metadata,
	}
}

// SessionStateFromMap creates a SessionState from a map
func SessionStateFromMap(m map[string]any) *SessionState {
	if m == nil {
		return nil
	}
	
	state := &SessionState{}
	if v, ok := m["session_id"].(string); ok {
		state.SessionID = v
	}
	if v, ok := m["conversation_id"].(string); ok {
		state.ConversationID = v
	}
	if v, ok := m["user_id"].(string); ok {
		state.UserID = v
	}
	if v, ok := m["intent"].(string); ok {
		state.Intent = v
	}
	if v, ok := m["scratchpad"].(string); ok {
		state.Scratchpad = v
	}
	if v, ok := m["turn_count"].(int); ok {
		state.TurnCount = v
	}
	if v, ok := m["last_updated"].(time.Time); ok {
		state.LastUpdated = v
	}
	return state
}
