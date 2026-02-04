// Package context provides the context orchestrator for the agent
package context

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/antigravity/go-agent-service/internal/nucleus"
)

// Orchestrator manages context assembly for agent queries
type Orchestrator struct {
	nucleus *nucleus.Client
	logger  *zap.SugaredLogger
}

// NewOrchestrator creates a new context orchestrator
func NewOrchestrator(nucleusClient *nucleus.Client, logger *zap.SugaredLogger) *Orchestrator {
	return &Orchestrator{
		nucleus: nucleusClient,
		logger:  logger,
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
	RetrievedNodes []nucleus.Node    `json:"retrieved_nodes"`
	RelatedNodes   []nucleus.Node    `json:"related_nodes"`
	Edges          []nucleus.Edge    `json:"edges"`
	Metadata       map[string]any    `json:"metadata"`
}

// Process extracts entities and builds context from a query
func (o *Orchestrator) Process(ctx context.Context, query string, contextEntities []string) (*Context, error) {
	o.logger.Debugw("Processing query for context",
		"query", query,
		"provided_entities", len(contextEntities),
	)

	// Step 1: Extract entities from query text
	entities := o.extractEntities(query)

	// Add provided context entities
	for _, e := range contextEntities {
		entities = append(entities, Entity{
			Type:  "reference",
			ID:    e,
			Value: e,
		})
	}

	o.logger.Infow("Extracted entities", "count", len(entities))

	// Step 2: Fetch entity data from Nucleus
	retrievedNodes, err := o.fetchNodes(ctx, entities)
	if err != nil {
		o.logger.Warnw("Failed to fetch nodes", "error", err)
		// Continue with empty nodes rather than failing
	}

	// Step 3: Get related nodes for primary entities
	var relatedNodes []nucleus.Node
	var edges []nucleus.Edge
	for _, e := range entities {
		if e.Type == "ticket" || e.Type == "pr" {
			related, relEdges, err := o.nucleus.GetRelatedNodes(ctx, e.ID)
			if err != nil {
				o.logger.Warnw("Failed to get related nodes", "entity", e.ID, "error", err)
				continue
			}
			relatedNodes = append(relatedNodes, related...)
			edges = append(edges, relEdges...)
		}
	}

	return &Context{
		Query:          query,
		Entities:       entities,
		RetrievedNodes: retrievedNodes,
		RelatedNodes:   relatedNodes,
		Edges:          edges,
		Metadata:       make(map[string]any),
	}, nil
}

// fetchNodes retrieves node data from Nucleus for all entities
func (o *Orchestrator) fetchNodes(ctx context.Context, entities []Entity) ([]nucleus.Node, error) {
	if o.nucleus == nil || len(entities) == 0 {
		return nil, nil
	}

	// Collect all entity IDs
	ids := make([]string, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, e.ID)
	}

	return o.nucleus.QueryNodes(ctx, ids)
}

// extractEntities extracts structured entities from natural language
func (o *Orchestrator) extractEntities(query string) []Entity {
	entities := []Entity{}

	// Pattern for ticket IDs (JIRA-style: PROJ-123)
	ticketPattern := regexp.MustCompile(`\b([A-Z]+-\d+)\b`)
	for _, match := range ticketPattern.FindAllString(query, -1) {
		entities = append(entities, Entity{
			Type:  "ticket",
			ID:    match,
			Value: match,
		})
	}

	// Pattern for PR references (#123 or PR-123)
	prPattern := regexp.MustCompile(`\bPR[-#]?(\d+)\b`)
	for _, match := range prPattern.FindAllStringSubmatch(query, -1) {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Type:  "pr",
				ID:    "PR-" + match[1],
				Value: match[0],
			})
		}
	}

	// Pattern for file paths with extensions
	filePattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_/.-]*\.(ts|js|go|py|rs|tsx|jsx|json|yaml|yml|md))\b`)
	for _, match := range filePattern.FindAllStringSubmatch(query, -1) {
		if len(match) > 0 && !isCommonWord(match[1]) {
			entities = append(entities, Entity{
				Type:  "file",
				ID:    match[1],
				Value: match[1],
			})
		}
	}

	// Pattern for @mentions
	mentionPattern := regexp.MustCompile(`@([a-zA-Z][a-zA-Z0-9_-]+)`)
	for _, match := range mentionPattern.FindAllStringSubmatch(query, -1) {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Type:  "user",
				ID:    match[1],
				Value: match[0],
			})
		}
	}

	// Pattern for service names (common infra terms)
	serviceTerms := []string{"api", "auth", "gateway", "database", "cache", "redis", "postgres", "kafka"}
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
		"this": true, "that": true, "from": true, "are": true,
		"used": true, "using": true, "into": true,
	}
	return common[strings.ToLower(s)]
}

// FormatForLLM converts the context into a formatted string for the LLM prompt
func (c *Context) FormatForLLM() string {
	var sections []string

	// Format retrieved nodes
	if len(c.RetrievedNodes) > 0 {
		var lines []string
		lines = append(lines, "## Knowledge Graph Context")
		for _, n := range c.RetrievedNodes {
			props := formatProperties(n.Properties)
			lines = append(lines, fmt.Sprintf("- [%s] **%s**: %s %s",
				n.EntityType, n.ID, n.DisplayName, props))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Format related nodes
	if len(c.RelatedNodes) > 0 {
		var lines []string
		lines = append(lines, "## Related Items")
		for i, n := range c.RelatedNodes {
			relationship := ""
			if i < len(c.Edges) {
				relationship = c.Edges[i].Relationship
			}
			lines = append(lines, fmt.Sprintf("- [%s] %s (%s)",
				n.EntityType, n.DisplayName, relationship))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	if len(sections) == 0 {
		return ""
	}

	return strings.Join(sections, "\n\n")
}

func formatProperties(props map[string]any) string {
	if len(props) == 0 {
		return ""
	}

	var parts []string
	for k, v := range props {
		if k == "source" && v == "stub" {
			continue // Skip stub marker
		}
		parts = append(parts, fmt.Sprintf("%s: %v", k, v))
	}

	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

// Compress reduces context to fit within token limits
func (o *Orchestrator) Compress(ctx *Context, maxTokens int) *Context {
	o.logger.Debugw("Compressing context",
		"entities", len(ctx.Entities),
		"nodes", len(ctx.RetrievedNodes),
		"max_tokens", maxTokens,
	)

	// Limit entities
	if len(ctx.Entities) > 10 {
		ctx.Entities = ctx.Entities[:10]
	}

	// Limit retrieved nodes
	if len(ctx.RetrievedNodes) > 5 {
		ctx.RetrievedNodes = ctx.RetrievedNodes[:5]
	}

	// Limit related nodes
	if len(ctx.RelatedNodes) > 5 {
		ctx.RelatedNodes = ctx.RelatedNodes[:5]
		ctx.Edges = ctx.Edges[:5]
	}

	return ctx
}
