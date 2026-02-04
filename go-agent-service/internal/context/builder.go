// Package context provides the context builder for ADK
package context

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antigravity/go-agent-service/internal/memory"
)

// Builder assembles fresh context for each LLM call
type Builder struct {
	memoryStore memory.MemoryStore
	config      *memory.ContextConfig
}

// NewBuilder creates a new context builder
func NewBuilder(store memory.MemoryStore, config *memory.ContextConfig) *Builder {
	if config == nil {
		config = memory.DefaultContextConfig()
	}
	return &Builder{
		memoryStore: store,
		config:      config,
	}
}

// Build creates a fresh context string for the LLM
func (b *Builder) Build(ctx context.Context, sessionID, query string) (string, error) {
	var sections []string

	// 1. System Prompt
	if b.config.SystemPrompt != "" {
		sections = append(sections, b.config.SystemPrompt)
	}

	// 2. Session Summary (rolling conversation summary)
	session, err := b.memoryStore.GetSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	
	if session != nil && session.Summary != "" {
		sections = append(sections, fmt.Sprintf("## Conversation Summary\n%s", session.Summary))
	}

	// 3. Relevant Past Turns (semantic search)
	relevantTurns, err := b.memoryStore.SearchTurns(ctx, sessionID, query, b.config.MaxRelevantTurns)
	if err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to search turns: %v\n", err)
	}
	
	if len(relevantTurns) > 0 {
		sections = append(sections, b.formatRelevantTurns(relevantTurns))
	}

	// 4. Recent Turns (always include last N)
	recentTurns, err := b.memoryStore.GetTurns(ctx, sessionID, b.config.MaxRecentTurns)
	if err != nil {
		fmt.Printf("Warning: failed to get recent turns: %v\n", err)
	}
	
	if len(recentTurns) > 0 {
		sections = append(sections, b.formatRecentTurns(recentTurns))
	}

	// 5. Tool Descriptions
	if b.config.ToolDescriptions != "" {
		sections = append(sections, fmt.Sprintf("## Available Tools\n%s", b.config.ToolDescriptions))
	}

	// 6. Current Query
	sections = append(sections, fmt.Sprintf("## Current Request\n%s", query))

	return strings.Join(sections, "\n\n"), nil
}

// BuildWithContext creates a prompt including Knowledge Graph context from Orchestrator
func (b *Builder) BuildWithContext(ctx context.Context, sessionID, query string, kgContext *Context) (string, error) {
	var sections []string

	// 1. System Prompt
	if b.config.SystemPrompt != "" {
		sections = append(sections, b.config.SystemPrompt)
	}

	// 2. Knowledge Graph Context (from Orchestrator)
	if kgContext != nil {
		kgFormatted := kgContext.FormatForLLM()
		if kgFormatted != "" {
			sections = append(sections, kgFormatted)
		}
	}

	// 3. Session Summary
	if b.memoryStore != nil {
		session, err := b.memoryStore.GetSession(ctx, sessionID)
		if err == nil && session != nil && session.Summary != "" {
			sections = append(sections, fmt.Sprintf("## Conversation Summary\n%s", session.Summary))
		}
	}

	// 4. Recent Turns
	if b.memoryStore != nil {
		recentTurns, err := b.memoryStore.GetTurns(ctx, sessionID, b.config.MaxRecentTurns)
		if err == nil && len(recentTurns) > 0 {
			sections = append(sections, b.formatRecentTurns(recentTurns))
		}
	}

	// 5. Current Query
	sections = append(sections, fmt.Sprintf("## Current Request\n%s", query))

	return strings.Join(sections, "\n\n"), nil
}

// BuildWithHistory includes specific turn history
func (b *Builder) BuildWithHistory(ctx context.Context, sessionID, query string, turns []*memory.Turn) (string, error) {
	var sections []string

	// 1. System Prompt
	if b.config.SystemPrompt != "" {
		sections = append(sections, b.config.SystemPrompt)
	}

	// 2. Provided History
	if len(turns) > 0 {
		sections = append(sections, b.formatRecentTurns(turns))
	}

	// 3. Current Query
	sections = append(sections, fmt.Sprintf("## Current Request\n%s", query))

	return strings.Join(sections, "\n\n"), nil
}

// formatRelevantTurns formats semantically relevant turns
func (b *Builder) formatRelevantTurns(turns []*memory.Turn) string {
	if len(turns) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "## Relevant Context (from earlier in conversation)")
	
	for _, t := range turns {
		content := t.Content
		if t.Compressed && t.Summary != "" {
			content = t.Summary
		}
		
		timeAgo := formatTimeAgo(t.CreatedAt)
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", timeAgo, t.Role, truncate(content, 200)))
	}
	
	return strings.Join(lines, "\n")
}

// formatRecentTurns formats recent conversation turns
func (b *Builder) formatRecentTurns(turns []*memory.Turn) string {
	if len(turns) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "## Recent Conversation")
	
	for _, t := range turns {
		content := t.Content
		if t.Compressed && t.Summary != "" {
			content = t.Summary
		}
		
		role := "User"
		if t.Role == "assistant" {
			role = "Assistant"
		}
		
		lines = append(lines, fmt.Sprintf("%s: %s", role, content))
	}
	
	return strings.Join(lines, "\n")
}

// formatTimeAgo formats a time as relative (e.g., "5 mins ago")
func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%d mins ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d hrs ago", int(diff.Hours()))
	default:
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	}
}

// truncate truncates a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ExtractEntities extracts entity mentions from text
// For now, uses simple regex patterns - can be enhanced with NER
func ExtractEntities(text string) []string {
	var entities []string
	
	// Look for common patterns like JIRA-123, PR #45, @username
	patterns := []string{
		// TODO: Add proper regex extraction
	}
	
	_ = patterns // Placeholder
	
	return entities
}
