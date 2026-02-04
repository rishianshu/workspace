// Package tools provides agentic tools for store operations
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/antigravity/go-agent-service/internal/store"

	"go.uber.org/zap"
)

// StoreTool provides KV and graph storage operations
// Note: For semantic search, use nucleus_search.brain_search instead
// which handles embeddings internally via Nucleus
type StoreTool struct {
	client   *store.Client
	logger   *zap.SugaredLogger
	tenantID string
}

// NewStoreTool creates a new store tool
func NewStoreTool(storeURL string, logger *zap.SugaredLogger) *StoreTool {
	tool := &StoreTool{
		logger:   logger,
		tenantID: "default",
	}

	// Try to connect to Store Core
	if storeURL != "" {
		client, err := store.NewClient(storeURL, logger)
		if err != nil {
			logger.Warnw("Failed to connect to Store Core, will use stubs", "error", err)
		} else {
			tool.client = client
		}
	}

	return tool
}

// Close closes the store client
func (t *StoreTool) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// Definition returns the tool definition for LLM
func (t *StoreTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name: "store",
		Description: `Key-value and graph storage utilities.`,
		Actions: []ActionDefinition{
			{
				Name:        "kv_get",
				Description: "Get value by key (key, projectId) → value",
				InputSchema: `{"type":"object","properties":{"key":{"type":"string","description":"Key to retrieve"},"projectId":{"type":"string","description":"Project ID"}},"required":["key"]}`,
			},
			{
				Name:        "kv_put",
				Description: "Store value by key (key, value, projectId)",
				InputSchema: `{"type":"object","properties":{"key":{"type":"string","description":"Key to store"},"value":{"type":"string","description":"Value to store"},"projectId":{"type":"string","description":"Project ID"}},"required":["key","value"]}`,
			},
			{
				Name:        "graph_query",
				Description: "Graph traversal queries (nodeId, depth, projectId) → nodes, edges",
				InputSchema: `{"type":"object","properties":{"nodeId":{"type":"string","description":"Starting Node ID"},"depth":{"type":"integer","description":"Traversal depth"},"projectId":{"type":"string","description":"Project ID"}},"required":["nodeId"]}`,
			},
		},
	}
}

// Execute runs the store tool
func (t *StoreTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)
	projectID, _ := params["projectId"].(string)
	if projectID == "" {
		projectID = "default"
	}

	switch action {
	case "kv_get":
		return t.kvGet(ctx, projectID, params)
	case "kv_put":
		return t.kvPut(ctx, projectID, params)
	case "graph_query":
		return t.graphQuery(ctx, projectID, params)
	default:
		return &Result{
			Success: false,
			Message: "unknown action: " + action + " (use nucleus_search.brain_search for semantic search)",
		}, nil
	}
}

func (t *StoreTool) kvGet(ctx context.Context, projectID string, params map[string]any) (*Result, error) {
	key, _ := params["key"].(string)

	if t.client != nil {
		value, err := t.client.KVGet(ctx, t.tenantID, projectID, key)
		if err != nil {
			return &Result{
				Success: false,
				Message: err.Error(),
			}, nil
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"key": key, "value": string(value)},
		}, nil
	}

	// Stub response
	return &Result{
		Success: true,
		Data:    map[string]any{"key": key, "value": nil, "stub": true},
		Message: fmt.Sprintf("KV get: %s (stub)", key),
	}, nil
}

func (t *StoreTool) kvPut(ctx context.Context, projectID string, params map[string]any) (*Result, error) {
	key, _ := params["key"].(string)
	value := params["value"]

	valueBytes, _ := json.Marshal(value)

	if t.client != nil {
		err := t.client.KVPut(ctx, t.tenantID, projectID, key, valueBytes)
		if err != nil {
			return &Result{
				Success: false,
				Message: err.Error(),
			}, nil
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"key": key, "stored": true},
			Message: fmt.Sprintf("Stored key: %s", key),
		}, nil
	}

	// Stub response
	return &Result{
		Success: true,
		Data:    map[string]any{"key": key, "stored": true, "stub": true},
		Message: fmt.Sprintf("Stored key: %s (stub)", key),
	}, nil
}

func (t *StoreTool) graphQuery(ctx context.Context, projectID string, params map[string]any) (*Result, error) {
	nodeID, _ := params["nodeId"].(string)
	depth, _ := params["depth"].(float64)
	if depth == 0 {
		depth = 1
	}

	if t.client != nil {
		nodes, edges, err := t.client.GraphQuery(ctx, t.tenantID, projectID, nodeID, int(depth))
		if err != nil {
			return &Result{
				Success: false,
				Message: err.Error(),
			}, nil
		}
		return &Result{
			Success: true,
			Data: map[string]any{
				"nodeId": nodeID,
				"depth":  int(depth),
				"nodes":  nodes,
				"edges":  edges,
			},
		}, nil
	}

	// Stub response
	return &Result{
		Success: true,
		Data: map[string]any{
			"nodeId": nodeID,
			"depth":  int(depth),
			"nodes":  []map[string]any{},
			"edges":  []map[string]any{},
			"stub":   true,
		},
		Message: fmt.Sprintf("Graph query from %s depth %d (stub)", nodeID, int(depth)),
	}, nil
}
