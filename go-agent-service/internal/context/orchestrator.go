// Package context provides the context orchestrator for the agent
package context

import (
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// Orchestrator manages context assembly for agent queries
type Orchestrator struct {
	logger *zap.SugaredLogger
}

// NewOrchestrator creates a new context orchestrator
func NewOrchestrator(logger *zap.SugaredLogger) *Orchestrator {
	return &Orchestrator{
		logger: logger,
	}
}

// Entity represents an extracted entity from query
type Entity struct {
	Type  string `json:"type"`  // ticket, pr, file, service, user
	ID    string `json:"id"`
	Value string `json:"value"`
}

// Context represents the assembled context for agent processing
type Context struct {
	Query          string            `json:"query"`
	Entities       []Entity          `json:"entities"`
	RetrievedNodes []map[string]any  `json:"retrieved_nodes"`
	Metadata       map[string]any    `json:"metadata"`
}

// Process extracts entities and builds context from a query
func (o *Orchestrator) Process(query string, contextEntities []string) *Context {
	o.logger.Debugw("Processing query for context",
		"query", query,
		"provided_entities", len(contextEntities),
	)

	// Extract entities from query
	entities := o.extractEntities(query)

	// Add provided context entities
	for _, e := range contextEntities {
		entities = append(entities, Entity{
			Type:  "reference",
			ID:    e,
			Value: e,
		})
	}

	o.logger.Infow("Extracted entities",
		"count", len(entities),
	)

	return &Context{
		Query:          query,
		Entities:       entities,
		RetrievedNodes: nil, // Will be populated from Nucleus
		Metadata:       make(map[string]any),
	}
}

// extractEntities extracts structured entities from natural language
func (o *Orchestrator) extractEntities(query string) []Entity {
	entities := []Entity{}

	// Pattern for ticket IDs (JIRA-style)
	ticketPattern := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)
	for _, match := range ticketPattern.FindAllString(query, -1) {
		entities = append(entities, Entity{
			Type:  "ticket",
			ID:    match,
			Value: match,
		})
	}

	// Pattern for PR numbers
	prPattern := regexp.MustCompile(`\b(?:PR|pull request)?\s*#?(\d+)\b`)
	for _, match := range prPattern.FindAllStringSubmatch(query, -1) {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Type:  "pr",
				ID:    match[1],
				Value: match[0],
			})
		}
	}

	// Pattern for file paths
	filePattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*\.[a-z]+)\b`)
	for _, match := range filePattern.FindAllString(query, -1) {
		// Filter out common false positives
		if !isCommonWord(match) {
			entities = append(entities, Entity{
				Type:  "file",
				ID:    match,
				Value: match,
			})
		}
	}

	// Pattern for service names (common infra terms)
	serviceTerms := []string{"api", "service", "server", "gateway", "auth", "database", "cache"}
	lower := strings.ToLower(query)
	for _, term := range serviceTerms {
		if strings.Contains(lower, term) {
			entities = append(entities, Entity{
				Type:  "service",
				ID:    term,
				Value: term,
			})
		}
	}

	return entities
}

func isCommonWord(s string) bool {
	common := map[string]bool{
		"the": true, "and": true, "for": true, "with": true,
		"this": true, "that": true, "from": true,
	}
	return common[strings.ToLower(s)]
}

// Compress reduces context to fit within token limits
func (o *Orchestrator) Compress(ctx *Context, maxTokens int) *Context {
	// Simplified compression - in production would use actual token counting
	o.logger.Debugw("Compressing context",
		"entities", len(ctx.Entities),
		"max_tokens", maxTokens,
	)

	// Keep most relevant entities (limit to 10)
	if len(ctx.Entities) > 10 {
		ctx.Entities = ctx.Entities[:10]
	}

	return ctx
}
