// Package memory provides the three-tier ADK memory system
package memory

import (
	"context"
	"time"
)

// ================= Core Types =================

// Session represents the current state of a conversation session
type Session struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversation_id"`
	UserID         string         `json:"user_id"`
	Summary        string         `json:"summary"`          // Rolling conversation summary
	State          map[string]any `json:"state"`            // Structured state (not raw messages)
	LastActivity   time.Time      `json:"last_activity"`
	TurnCount      int            `json:"turn_count"`
}

// Turn represents a single conversation turn (message)
type Turn struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	Role       string    `json:"role"`       // "user" | "assistant"
	Content    string    `json:"content"`    // Original content
	Summary    string    `json:"summary"`    // Compressed version (for old turns)
	Embedding  []float32 `json:"embedding"`  // Vector for semantic search
	Compressed bool      `json:"compressed"` // True if content was summarized
	CreatedAt  time.Time `json:"created_at"`
}

// Fact represents a structured fact about an entity
type Fact struct {
	ID        string    `json:"id"`
	EntityID  string    `json:"entity_id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`    // "resolved", "mentioned", "acted_on", "created"
	Content   string    `json:"content"` // The fact content
	Source    string    `json:"source"`  // "jira", "github", "agent", "user"
	CreatedAt time.Time `json:"created_at"`
}

// ================= Memory Interface =================

// MemoryStore is the unified interface for the 3-tier memory system
type MemoryStore interface {
	// Session Management (Short-term)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error

	// Turn Management (Episodic)
	AddTurn(ctx context.Context, turn *Turn) error
	GetTurns(ctx context.Context, sessionID string, limit int) ([]*Turn, error)
	SearchTurns(ctx context.Context, sessionID, query string, limit int) ([]*Turn, error)
	CompressTurns(ctx context.Context, sessionID string, olderThan time.Duration) error

	// Fact Management (Semantic)
	StoreFact(ctx context.Context, fact *Fact) error
	GetEntityFacts(ctx context.Context, entityID string, limit int) ([]*Fact, error)
	SearchFacts(ctx context.Context, query string, limit int) ([]*Fact, error)
}

// ================= Context Builder =================

// ContextBuilder assembles fresh context for each LLM call
type ContextBuilder interface {
	// Build creates a fresh context string for the LLM
	Build(ctx context.Context, sessionID, query string) (string, error)
	
	// BuildWithHistory includes specific turn history
	BuildWithHistory(ctx context.Context, sessionID, query string, turns []*Turn) (string, error)
}

// ContextConfig holds configuration for context building
type ContextConfig struct {
	MaxTokens         int           // Maximum tokens for context
	MaxRelevantTurns  int           // How many turns to retrieve via semantic search
	MaxRecentTurns    int           // How many recent turns to always include
	CompressionAge    time.Duration // When to compress old turns
	SystemPrompt      string        // Base system prompt
	ToolDescriptions  string        // Available tools description
}

// DefaultContextConfig returns sensible defaults
func DefaultContextConfig() *ContextConfig {
	return &ContextConfig{
		MaxTokens:        4096,
		MaxRelevantTurns: 5,
		MaxRecentTurns:   3,
		CompressionAge:   10 * time.Minute,
	}
}

// ================= Embedding Service =================

// EmbeddingService generates vector embeddings for semantic search
type EmbeddingService interface {
	// Embed generates an embedding vector for the given text
	Embed(ctx context.Context, text string) ([]float32, error)
	
	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// ================= Compressor =================

// Compressor handles summarization of old conversation turns
type Compressor interface {
	// Summarize compresses multiple turns into a summary
	Summarize(ctx context.Context, turns []*Turn) (string, error)
	
	// UpdateRollingSummary adds new information to existing summary
	UpdateRollingSummary(ctx context.Context, existingSummary string, newTurns []*Turn) (string, error)
}
