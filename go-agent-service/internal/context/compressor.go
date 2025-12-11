// Package context provides the compressor for session summarization
package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/antigravity/go-agent-service/internal/memory"
)

// SessionCompressor handles summarization of old conversation turns
type SessionCompressor struct {
	memoryStore memory.MemoryStore
	llm         LLMSummarizer
}

// LLMSummarizer interface for summarization
type LLMSummarizer interface {
	Summarize(ctx context.Context, prompt string) (string, error)
}

// NewCompressor creates a new session compressor
func NewCompressor(store memory.MemoryStore, llm LLMSummarizer) *SessionCompressor {
	return &SessionCompressor{
		memoryStore: store,
		llm:         llm,
	}
}

// Summarize compresses multiple turns into a summary
func (c *SessionCompressor) Summarize(ctx context.Context, turns []*memory.Turn) (string, error) {
	if len(turns) == 0 {
		return "", nil
	}

	// If no LLM available, create a simple concatenation
	if c.llm == nil {
		return c.simpleSummarize(turns), nil
	}

	// Build prompt for LLM summarization
	prompt := buildSummarizationPrompt(turns)
	
	summary, err := c.llm.Summarize(ctx, prompt)
	if err != nil {
		// Fallback to simple summary on error
		return c.simpleSummarize(turns), nil
	}
	
	return summary, nil
}

// UpdateRollingSummary adds new information to existing summary
func (c *SessionCompressor) UpdateRollingSummary(ctx context.Context, existingSummary string, newTurns []*memory.Turn) (string, error) {
	if len(newTurns) == 0 {
		return existingSummary, nil
	}

	newSummary, err := c.Summarize(ctx, newTurns)
	if err != nil {
		return existingSummary, err
	}

	if existingSummary == "" {
		return newSummary, nil
	}

	// If we have an LLM, merge summaries intelligently
	if c.llm != nil {
		prompt := fmt.Sprintf(`Merge these two conversation summaries into one concise summary:

Previous Summary:
%s

New Information:
%s

Merged Summary:`, existingSummary, newSummary)
		
		merged, err := c.llm.Summarize(ctx, prompt)
		if err == nil {
			return merged, nil
		}
	}

	// Simple merge fallback
	return existingSummary + "\n\n" + newSummary, nil
}

// CompressOldTurns compresses turns older than the specified session
func (c *SessionCompressor) CompressOldTurns(ctx context.Context, sessionID string) error {
	// Get old uncompressed turns
	turns, err := c.memoryStore.GetTurns(ctx, sessionID, 100)
	if err != nil {
		return err
	}

	// Filter to only uncompressed old turns (keep last 5 uncompressed)
	oldTurns := make([]*memory.Turn, 0)
	if len(turns) > 5 {
		for _, t := range turns[:len(turns)-5] {
			if !t.Compressed {
				oldTurns = append(oldTurns, t)
			}
		}
	}

	if len(oldTurns) == 0 {
		return nil
	}

	// Summarize old turns
	summary, err := c.Summarize(ctx, oldTurns)
	if err != nil {
		return err
	}

	// Update session with new rolling summary
	session, err := c.memoryStore.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if session != nil {
		if session.Summary != "" {
			session.Summary = session.Summary + "\n\n" + summary
		} else {
			session.Summary = summary
		}
		
		if err := c.memoryStore.UpdateSession(ctx, session); err != nil {
			return err
		}
	}

	return nil
}

// simpleSummarize creates a basic summary without LLM
func (c *SessionCompressor) simpleSummarize(turns []*memory.Turn) string {
	var points []string
	
	for _, t := range turns {
		// Extract first sentence or truncate
		content := t.Content
		if idx := strings.Index(content, "."); idx > 0 && idx < 100 {
			content = content[:idx+1]
		} else if len(content) > 100 {
			content = content[:100] + "..."
		}
		
		points = append(points, fmt.Sprintf("- %s: %s", t.Role, content))
	}
	
	return strings.Join(points, "\n")
}

// buildSummarizationPrompt creates a prompt for LLM summarization
func buildSummarizationPrompt(turns []*memory.Turn) string {
	var conversation []string
	
	for _, t := range turns {
		role := "User"
		if t.Role == "assistant" {
			role = "Assistant"
		}
		conversation = append(conversation, fmt.Sprintf("%s: %s", role, t.Content))
	}
	
	return fmt.Sprintf(`Summarize this conversation into a brief, factual summary. Focus on:
- Key topics discussed
- Decisions made
- Actions taken or requested
- Important entities mentioned (tickets, PRs, etc.)

Conversation:
%s

Summary:`, strings.Join(conversation, "\n"))
}
