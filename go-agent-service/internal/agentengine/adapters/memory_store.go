package adapters

import (
	"context"
	"encoding/json"
	"time"

	"github.com/antigravity/go-agent-service/internal/agentengine"
	"github.com/antigravity/go-agent-service/internal/memory"
)

// MemoryAdapter wraps a memory.MemoryStore for AgentEngine.
type MemoryAdapter struct {
	store memory.MemoryStore
}

// NewMemoryAdapter creates a new memory adapter.
func NewMemoryAdapter(store memory.MemoryStore) *MemoryAdapter {
	return &MemoryAdapter{store: store}
}

// AddTurn stores a turn when memory is configured.
func (m *MemoryAdapter) AddTurn(ctx context.Context, sessionID, content, role string, timestamp time.Time) error {
	if m == nil || m.store == nil {
		return nil
	}

	turn := &memory.Turn{
		SessionID: sessionID,
		Role:      role,
		Content:   content,
		CreatedAt: timestamp,
	}
	return m.store.AddTurn(ctx, turn)
}

// StoreFact records an observation as a fact when possible.
func (m *MemoryAdapter) StoreFact(ctx context.Context, sessionID string, observation agentengine.Observation) error {
	if m == nil || m.store == nil || observation.Result == nil {
		return nil
	}

	payload, _ := json.Marshal(observation.Result)
	fact := &memory.Fact{
		EntityID:  observation.ToolName,
		SessionID: sessionID,
		Type:      "tool_observation",
		Content:   string(payload),
		Source:    observation.ToolName,
		CreatedAt: time.Now(),
	}
	return m.store.StoreFact(ctx, fact)
}
