// Package tools provides agentic tools for nucleus search and store operations
package tools

import (
	"context"

	"github.com/antigravity/go-agent-service/internal/nucleus"
)

// NucleusSearchTool provides brain search and entity resolution
type NucleusSearchTool struct {
	client *nucleus.Client
}

// NewNucleusSearchTool creates a new nucleus search tool
func NewNucleusSearchTool(client *nucleus.Client) *NucleusSearchTool {
	return &NucleusSearchTool{client: client}
}

// Definition returns the tool definition for LLM
func (t *NucleusSearchTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name: "nucleus_search",
		Description: `Search the knowledge graph and retrieve context for RAG.`,
		Actions: []ActionDefinition{
			{
				Name:        "brain_search",
				Description: "Semantic search with RAG context (query, projectId) → hits, promptPack, citations",
				InputSchema: `{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"projectId":{"type":"string","description":"Project ID"}},"required":["query"]}`,
				OutputSchema: `{"type":"object","properties":{"hits":{"type":"array","items":{"type":"object","properties":{"nodeId":{"type":"string"},"nodeType":{"type":"string"},"profileId":{"type":"string"},"score":{"type":"number"},"title":{"type":"string"},"url":{"type":"string"}}}},"episodes":{"type":"array","items":{"type":"object"}},"context":{"type":"string"},"citations":{"type":"array","items":{"type":"object"}}}}`,
			},
			{
				Name:        "list_projects",
				Description: "List available projects",
				InputSchema: `{"type":"object","properties":{}}`,
				OutputSchema: `{"type":"object","properties":{"projects":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"slug":{"type":"string"},"displayName":{"type":"string"},"description":{"type":"string"}}}}}}`,
			},
			{
				Name:        "list_endpoints",
				Description: "List metadata endpoints for a project",
				InputSchema: `{"type":"object","properties":{"projectId":{"type":"string","description":"Project ID"}}}`,
				OutputSchema: `{"type":"object","properties":{"endpoints":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"sourceId":{"type":"string"},"projectId":{"type":"string"},"templateId":{"type":"string"},"description":{"type":"string"},"verb":{"type":"string"},"url":{"type":"string"},"authPolicy":{"type":"string"},"domain":{"type":"string"},"labels":{"type":"array","items":{"type":"string"}},"capabilities":{"type":"array","items":{"type":"string"}},"delegatedConnected":{"type":"boolean"}}}}}}`,
			},
			{
				Name:        "get_entity",
				Description: "Get entity details by ID",
				InputSchema: `{"type":"object","properties":{"id":{"type":"string","description":"Entity ID"}},"required":["id"]}`,
				OutputSchema: `{"type":"object","properties":{"entity":{"type":"object","properties":{"id":{"type":"string"},"displayName":{"type":"string"},"entityType":{"type":"string"},"properties":{"type":"object"}}}}}`,
			},
		},
	}
}

// ActionDefinition describes a specific action within a tool
type ActionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"inputSchema,omitempty"`
	OutputSchema string `json:"outputSchema,omitempty"`
}

// ToolDefinition describes a tool for LLM function calling
type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Actions     []ActionDefinition `json:"actions"`
}

// Execute runs the nucleus search tool
func (t *NucleusSearchTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)

	switch action {
	case "brain_search":
		query, _ := params["query"].(string)
		projectID, _ := params["projectId"].(string)

		result, err := t.client.BrainSearch(ctx, query, projectID, nil)
		if err != nil {
			return &Result{Success: false, Message: err.Error()}, nil
		}

		return &Result{
			Success: true,
			Data: map[string]any{
				"hits":      result.Hits,
				"episodes":  result.Episodes,
				"context":   result.PromptPack.ContextMarkdown,
				"citations": result.PromptPack.Citations,
			},
		}, nil

	case "list_projects":
		projects, err := t.client.ListProjects(ctx)
		if err != nil {
			return &Result{Success: false, Message: err.Error()}, nil
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"projects": projects},
		}, nil

	case "list_endpoints":
		projectID, _ := params["projectId"].(string)
		endpoints, err := t.client.ListEndpoints(ctx, projectID)
		if err != nil {
			return &Result{Success: false, Message: err.Error()}, nil
		}
		sanitized := make([]map[string]any, 0, len(endpoints))
		for _, ep := range endpoints {
			sanitized = append(sanitized, map[string]any{
				"id":                 ep.ID,
				"name":               ep.Name,
				"sourceId":           ep.SourceID,
				"projectId":          ep.ProjectID,
				"templateId":         ep.TemplateID,
				"description":        ep.Description,
				"verb":               ep.Verb,
				"url":                ep.URL,
				"authPolicy":         ep.AuthPolicy,
				"domain":             ep.Domain,
				"labels":             ep.Labels,
				"capabilities":       ep.Capabilities,
				"delegatedConnected": ep.DelegatedConnected,
			})
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"endpoints": sanitized},
		}, nil

	case "get_entity":
		id, _ := params["id"].(string)
		nodes, err := t.client.QueryNodes(ctx, []string{id})
		if err != nil {
			return &Result{Success: false, Message: err.Error()}, nil
		}
		if len(nodes) == 0 {
			return &Result{Success: false, Message: "entity not found"}, nil
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"entity": nodes[0]},
		}, nil

	default:
		return &Result{
			Success: false,
			Message: "unknown action: " + action,
		}, nil
	}
}
