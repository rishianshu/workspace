// Package tools provides the tool registry for agentic operations
package tools

import (
	"context"
	"fmt"

	"github.com/antigravity/go-agent-service/internal/config"
	"github.com/antigravity/go-agent-service/internal/mcp"

	"go.uber.org/zap"
)

// Registry manages all available tools
type Registry struct {
	mcpClient *mcp.Client
	storeTool *StoreTool
	logger    *zap.SugaredLogger
}

// NewRegistry creates a new tool registry
func NewRegistry(cfg *config.Config, logger *zap.SugaredLogger) *Registry {
	// Create MCP client
	mcpClient := mcp.NewClientWithConfig(mcp.ClientConfig{
		BaseURL:   cfg.MCPServerURL,
		AuthToken: cfg.MCPAuthToken,
	}, logger)

	// Store Core URL (default to localhost:9099)
	storeURL := "localhost:9099"

	return &Registry{
		mcpClient: mcpClient,
		storeTool: NewStoreTool(storeURL, logger),
		logger:    logger,
	}
}

// Connect establishes connections to backend services
func (r *Registry) Connect(ctx context.Context) error {
	_ = ctx
	return nil
}

// Close closes backend connections
func (r *Registry) Close() error {
	if err := r.storeTool.Close(); err != nil {
		r.logger.Warnw("Failed to close StoreTool", "error", err)
	}
	return nil
}

// ListTools returns all available tools for LLM function calling
func (r *Registry) ListTools(ctx context.Context) []ToolDefinition {
	return r.ListToolsFor(ctx, "", "")
}

// ListToolsFor returns tools scoped to a user/project if provided.
func (r *Registry) ListToolsFor(ctx context.Context, userID, projectID string) []ToolDefinition {
	tools := []ToolDefinition{}

	// 1. Tools via MCP (UCL + Nucleus Brain)
	if userID != "" && projectID != "" {
		uclTools, err := r.mcpClient.ListTools(ctx, userID, projectID)
		if err != nil {
			r.logger.Warnw("Failed to list MCP tools", "error", err)
		} else {
			for _, t := range uclTools {
				// Map MCP actions to Tool actions
				actions := make([]ActionDefinition, 0, len(t.Actions))
				for _, a := range t.Actions {
					actions = append(actions, ActionDefinition{
						Name:         a.Name,
						Description:  a.Description,
						InputSchema:  a.InputSchema,
						OutputSchema: a.OutputSchema,
					})
				}

				tools = append(tools, ToolDefinition{
					Name:        t.Name,
					Description: t.Description,
					Actions:     actions,
				})
			}
		}
	}

	// 2. Store tool
	tools = append(tools, r.storeTool.Definition())

	return tools
}

// Execute runs a tool by name
func (r *Registry) Execute(ctx context.Context, name, action string, params map[string]any) (*Result, error) {
	r.logger.Infow("Executing tool", "name", name, "action", action)

	// Check if it's an MCP/UCL tool (e.g., "http.jira")
	if name != "store" {
		userID := getStringParam(params, "userId")
		projectID := getStringParam(params, "projectId")
		if params != nil {
			delete(params, "userId")
			delete(params, "projectId")
		}
		mcpResult, err := r.mcpClient.ExecuteTool(ctx, mcp.ToolCall{
			Name:       name,
			Action:     action,
			EndpointID: getStringParam(params, "endpointId"),
			KeyToken:   getStringParam(params, "keyToken"),
			UserID:     userID,
			ProjectID:  projectID,
			Params:     params,
		})
		if err != nil {
			return nil, err
		}
		// Convert mcp.Result to tools.Result
		return &Result{
			Success: mcpResult.Success,
			Data:    mcpResult.Data,
			Message: mcpResult.Message,
		}, nil
	}

	// Built-in tools
	switch name {
	case "store":
		params["action"] = action
		return r.storeTool.Execute(ctx, params)
	default:
		return &Result{
			Success: false,
			Message: fmt.Sprintf("unknown tool: %s", name),
		}, nil
	}
}

func getStringParam(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}
