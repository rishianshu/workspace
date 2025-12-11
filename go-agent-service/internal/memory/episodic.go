// Package memory provides episodic memory with pgvector
package memory

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// EpisodicStore implements MemoryStore using PostgreSQL with pgvector
type EpisodicStore struct {
	db       *sql.DB
	embedder EmbeddingService
}

// NewEpisodicStore creates a new episodic memory store
func NewEpisodicStore(connString string, embedder EmbeddingService) (*EpisodicStore, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &EpisodicStore{
		db:       db,
		embedder: embedder,
	}, nil
}

// Close closes the database connection
func (s *EpisodicStore) Close() error {
	return s.db.Close()
}

// ==================== Session Management ====================

// GetSession retrieves a session by ID
func (s *EpisodicStore) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	query := `
		SELECT id, conversation_id, user_id, summary, state, turn_count, last_activity
		FROM sessions WHERE id = $1
	`
	
	var session Session
	var stateJSON []byte
	
	err := s.db.QueryRowContext(ctx, query, sessionID).Scan(
		&session.ID,
		&session.ConversationID,
		&session.UserID,
		&session.Summary,
		&stateJSON,
		&session.TurnCount,
		&session.LastActivity,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// UpdateSession creates or updates a session
func (s *EpisodicStore) UpdateSession(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO sessions (id, conversation_id, user_id, summary, state, turn_count, last_activity)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (id) DO UPDATE SET
			summary = EXCLUDED.summary,
			state = EXCLUDED.state,
			turn_count = EXCLUDED.turn_count,
			last_activity = NOW()
	`
	
	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.ConversationID,
		session.UserID,
		session.Summary,
		session.State,
		session.TurnCount,
	)
	
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}
	
	return nil
}

// DeleteSession removes a session
func (s *EpisodicStore) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE id = $1", sessionID)
	return err
}

// ==================== Turn Management ====================

// AddTurn adds a new turn with embedding
func (s *EpisodicStore) AddTurn(ctx context.Context, turn *Turn) error {
	// Generate ID if not provided
	if turn.ID == "" {
		turn.ID = uuid.New().String()
	}
	
	// Generate embedding if embedder is available
	var embedding []float32
	if s.embedder != nil && turn.Content != "" {
		var err error
		embedding, err = s.embedder.Embed(ctx, turn.Content)
		if err != nil {
			// Log but don't fail - embedding is optional
			fmt.Printf("Warning: failed to generate embedding: %v\n", err)
		}
	}
	turn.Embedding = embedding
	
	query := `
		INSERT INTO turns (id, session_id, role, content, summary, embedding, compressed, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		turn.ID,
		turn.SessionID,
		turn.Role,
		turn.Content,
		turn.Summary,
		pgVectorFromSlice(embedding),
		turn.Compressed,
		turn.CreatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("failed to add turn: %w", err)
	}
	
	return nil
}

// GetTurns retrieves recent turns for a session
func (s *EpisodicStore) GetTurns(ctx context.Context, sessionID string, limit int) ([]*Turn, error) {
	query := `
		SELECT id, session_id, role, content, summary, compressed, created_at
		FROM turns
		WHERE session_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get turns: %w", err)
	}
	defer rows.Close()
	
	var turns []*Turn
	for rows.Next() {
		var t Turn
		if err := rows.Scan(&t.ID, &t.SessionID, &t.Role, &t.Content, &t.Summary, &t.Compressed, &t.CreatedAt); err != nil {
			return nil, err
		}
		turns = append(turns, &t)
	}
	
	// Reverse to get chronological order
	for i, j := 0, len(turns)-1; i < j; i, j = i+1, j-1 {
		turns[i], turns[j] = turns[j], turns[i]
	}
	
	return turns, nil
}

// SearchTurns performs semantic search on turns
func (s *EpisodicStore) SearchTurns(ctx context.Context, sessionID, query string, limit int) ([]*Turn, error) {
	if s.embedder == nil {
		// Fallback to recent turns if no embedder
		return s.GetTurns(ctx, sessionID, limit)
	}
	
	// Generate query embedding
	queryEmbedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	// Vector similarity search
	searchQuery := `
		SELECT id, session_id, role, content, summary, compressed, created_at,
		       1 - (embedding <=> $1) AS similarity
		FROM turns
		WHERE session_id = $2 AND embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $3
	`
	
	rows, err := s.db.QueryContext(ctx, searchQuery, 
		pgVectorFromSlice(queryEmbedding), 
		sessionID, 
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search turns: %w", err)
	}
	defer rows.Close()
	
	var turns []*Turn
	for rows.Next() {
		var t Turn
		var similarity float64
		if err := rows.Scan(&t.ID, &t.SessionID, &t.Role, &t.Content, &t.Summary, &t.Compressed, &t.CreatedAt, &similarity); err != nil {
			return nil, err
		}
		turns = append(turns, &t)
	}
	
	return turns, nil
}

// CompressTurns compresses turns older than threshold
func (s *EpisodicStore) CompressTurns(ctx context.Context, sessionID string, olderThan time.Duration) error {
	threshold := time.Now().Add(-olderThan)
	
	// For now, just mark as compressed - actual summarization would use LLM
	query := `
		UPDATE turns
		SET compressed = TRUE,
		    summary = CASE WHEN summary = '' THEN LEFT(content, 200) || '...' ELSE summary END
		WHERE session_id = $1 AND created_at < $2 AND compressed = FALSE
	`
	
	_, err := s.db.ExecContext(ctx, query, sessionID, threshold)
	if err != nil {
		return fmt.Errorf("failed to compress turns: %w", err)
	}
	
	return nil
}

// ==================== Fact Management ====================

// StoreFact stores a fact about an entity
func (s *EpisodicStore) StoreFact(ctx context.Context, fact *Fact) error {
	if fact.ID == "" {
		fact.ID = uuid.New().String()
	}
	
	var embedding []float32
	if s.embedder != nil && fact.Content != "" {
		var err error
		embedding, err = s.embedder.Embed(ctx, fact.Content)
		if err != nil {
			fmt.Printf("Warning: failed to generate fact embedding: %v\n", err)
		}
	}
	
	query := `
		INSERT INTO facts (id, entity_id, session_id, type, content, source, embedding, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	
	_, err := s.db.ExecContext(ctx, query,
		fact.ID,
		fact.EntityID,
		fact.SessionID,
		fact.Type,
		fact.Content,
		fact.Source,
		pgVectorFromSlice(embedding),
		fact.CreatedAt,
	)
	
	return err
}

// GetEntityFacts retrieves facts about an entity
func (s *EpisodicStore) GetEntityFacts(ctx context.Context, entityID string, limit int) ([]*Fact, error) {
	query := `
		SELECT id, entity_id, session_id, type, content, source, created_at
		FROM facts
		WHERE entity_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	
	rows, err := s.db.QueryContext(ctx, query, entityID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var facts []*Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.EntityID, &f.SessionID, &f.Type, &f.Content, &f.Source, &f.CreatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, &f)
	}
	
	return facts, nil
}

// SearchFacts performs semantic search on facts
func (s *EpisodicStore) SearchFacts(ctx context.Context, query string, limit int) ([]*Fact, error) {
	if s.embedder == nil {
		return nil, nil
	}
	
	queryEmbedding, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	
	searchQuery := `
		SELECT id, entity_id, session_id, type, content, source, created_at
		FROM facts
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`
	
	rows, err := s.db.QueryContext(ctx, searchQuery, pgVectorFromSlice(queryEmbedding), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var facts []*Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.EntityID, &f.SessionID, &f.Type, &f.Content, &f.Source, &f.CreatedAt); err != nil {
			return nil, err
		}
		facts = append(facts, &f)
	}
	
	return facts, nil
}

// ==================== Helpers ====================

// pgVectorFromSlice converts a float32 slice to pgvector format
func pgVectorFromSlice(v []float32) interface{} {
	if len(v) == 0 {
		return nil
	}
	// Format as pgvector string: '[1.0,2.0,3.0]'
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%f", f)
	}
	s += "]"
	return s
}
