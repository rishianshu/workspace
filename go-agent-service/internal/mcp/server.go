// Package mcp provides the MCP server that wraps UCL endpoints as tools
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/antigravity/go-agent-service/internal/appregistry"
	"github.com/antigravity/go-agent-service/internal/keystore"
	"github.com/antigravity/go-agent-service/internal/ucl"

	"go.uber.org/zap"
)

// ActionDefinition describes a specific action within a tool
type ActionDefinition struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	InputSchema  string `json:"inputSchema,omitempty"`
	OutputSchema string `json:"outputSchema,omitempty"`
}

// ToolDefinition describes a tool for LLM function calling
type ToolDefinition struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Actions     []ActionDefinition `json:"actions"`
	EndpointID  string             `json:"endpointId,omitempty"`
	TemplateID  string             `json:"templateId,omitempty"`
}

// ToolCall represents a request to execute a tool
type ToolCall struct {
	Name       string         `json:"name"`
	Action     string         `json:"action"`
	EndpointID string         `json:"endpointId,omitempty"`
	KeyToken   string         `json:"keyToken,omitempty"`
	UserID     string         `json:"userId,omitempty"`
	ProjectID  string         `json:"projectId,omitempty"`
	Params     map[string]any `json:"params"`
}

// Result represents the result of a tool execution
type Result struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Message string         `json:"message,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// Server wraps UCL, exposing endpoints as MCP tools
type Server struct {
	uclAddr     string
	keyStore    keystore.Store
	appResolver *appregistry.Resolver
	logger      *zap.SugaredLogger
	uclClient   *ucl.Client
}

// NewServer creates a new MCP server
func NewServer(uclAddr string, keyStore keystore.Store, resolver *appregistry.Resolver, logger *zap.SugaredLogger) *Server {
	return &Server{
		uclAddr:     uclAddr,
		keyStore:    keyStore,
		appResolver: resolver,
		logger:      logger,
	}
}

// Connect establishes connection to UCL gRPC
func (s *Server) Connect(ctx context.Context) error {
	client, err := ucl.NewClient(s.uclAddr, s.logger)
	if err != nil {
		s.logger.Warnw("Failed to connect to UCL", "error", err)
		return err
	}
	s.uclClient = client
	s.logger.Infow("Connected to UCL Gateway", "addr", s.uclAddr)
	return nil
}

// Close closes the UCL connection
func (s *Server) Close() error {
	if s.uclClient != nil {
		return s.uclClient.Close()
	}
	return nil
}

// ListTools returns tools scoped to a user/project via the app registry.
func (s *Server) ListTools(ctx context.Context, userID, projectID string) ([]ToolDefinition, error) {
	if s.appResolver == nil {
		return nil, fmt.Errorf("app registry unavailable")
	}
	if userID == "" || projectID == "" {
		return nil, fmt.Errorf("userId and projectId required")
	}
	return s.listAppTools(ctx, userID, projectID)
}

func (s *Server) listAppTools(ctx context.Context, userID, projectID string) ([]ToolDefinition, error) {
	if s.uclClient == nil {
		return nil, fmt.Errorf("ucl client unavailable")
	}
	resolved, err := s.appResolver.ResolveProjectApps(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}

	tools := make([]ToolDefinition, 0, len(resolved))
	for _, app := range resolved {
		if app.TemplateID == "" {
			s.logger.Warnw("Skipping app tool without templateId", "appId", app.AppID)
			continue
		}
		actions, err := s.uclClient.ListActions(ctx, app.TemplateID)
		actionDefs := []ActionDefinition{}
		if err == nil {
			for _, a := range actions {
				actionDefs = append(actionDefs, ActionDefinition{
					Name:        a.Name,
					Description: a.Description,
					InputSchema: a.InputSchemaJSON,
				})
			}
		}
		actionDefs = addReadActionsForApp(actionDefs)

		displayName := "App tool"
		if app.Endpoint != nil && app.Endpoint.Name != "" {
			displayName = app.Endpoint.Name
		} else if app.AppInstance != nil && app.AppInstance.DisplayName != "" {
			displayName = app.AppInstance.DisplayName
		}

		for _, actionDef := range actionDefs {
			tools = append(tools, ToolDefinition{
				Name:        fmt.Sprintf("app/%s/%s", app.AppID, actionDef.Name),
				Description: fmt.Sprintf("%s action %s via Workspace registry", displayName, actionDef.Name),
				Actions:     []ActionDefinition{actionDef},
				TemplateID:  app.TemplateID,
			})
		}
	}
	return tools, nil
}

func addReadActionsForApp(actions []ActionDefinition) []ActionDefinition {
	exists := map[string]bool{}
	for _, a := range actions {
		exists[a.Name] = true
	}

	readActions := []ActionDefinition{
		{
			Name:         "list_datasets",
			Description:  "List datasets for this app",
			InputSchema:  `{"type":"object","properties":{},"required":[]}`,
			OutputSchema: `{"type":"object","properties":{"datasets":{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"},"name":{"type":"string"},"kind":{"type":"string"},"supportsIncremental":{"type":"boolean"},"cdmModelId":{"type":"string"},"ingestionStrategy":{"type":"string"}}}}}}`,
		},
		{
			Name:         "get_schema",
			Description:  "Get dataset schema and stats",
			InputSchema:  `{"type":"object","properties":{"dataset_id":{"type":"string","description":"Dataset ID"}},"required":["dataset_id"]}`,
			OutputSchema: `{"type":"object","properties":{"schema":{"type":"object","properties":{"fields":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"dataType":{"type":"string"},"nullable":{"type":"boolean"},"precision":{"type":"integer"},"scale":{"type":"integer"},"comment":{"type":"string"}}}},"constraints":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"type":{"type":"string"},"fields":{"type":"array","items":{"type":"string"}}}}},"statistics":{"type":"object","properties":{"rowCount":{"type":"integer"},"sizeBytes":{"type":"integer"},"partitions":{"type":"integer"}}}}}}`,
		},
		{
			Name:         "read_data",
			Description:  "Read rows from a dataset (used for preview)",
			InputSchema:  `{"type":"object","properties":{"dataset_id":{"type":"string","description":"Dataset ID"},"filter":{"type":"object","description":"Filter object"},"limit":{"type":"integer","description":"Max rows"}},"required":["dataset_id"]}`,
			OutputSchema: `{"type":"object","properties":{"records":{"type":"array","items":{"type":"object"}}}}`,
		},
	}

	for _, ra := range readActions {
		if !exists[ra.Name] {
			actions = append(actions, ra)
		}
	}
	return actions
}

// ExecuteTool runs an action on a UCL endpoint with user credentials
func (s *Server) ExecuteTool(ctx context.Context, call ToolCall) (*Result, error) {
	s.logger.Infow("Executing tool", "name", call.Name, "action", call.Action, "endpoint", call.EndpointID)

	if s.uclClient == nil {
		return nil, fmt.Errorf("ucl client unavailable")
	}

	endpointID := ""
	keyToken := call.KeyToken
	if !strings.HasPrefix(call.Name, "app/") {
		return nil, fmt.Errorf("tool requires app binding")
	}

	appRef := strings.TrimPrefix(call.Name, "app/")
	parts := strings.SplitN(appRef, "/", 2)
	appID := parts[0]
	actionFromName := ""
	if len(parts) > 1 {
		actionFromName = parts[1]
	}
	if appID == "" {
		return nil, fmt.Errorf("missing app id")
	}
	if actionFromName != "" {
		if call.Action == "" {
			call.Action = actionFromName
		} else if call.Action != actionFromName {
			return nil, fmt.Errorf("action mismatch: %s vs %s", call.Action, actionFromName)
		}
	}
	if call.UserID == "" || call.ProjectID == "" {
		return nil, fmt.Errorf("missing userId or projectId")
	}
	if s.appResolver == nil {
		return nil, fmt.Errorf("app resolver unavailable")
	}
	resolved, err := s.appResolver.ResolveApp(ctx, call.UserID, call.ProjectID, appID)
	if err != nil {
		return nil, err
	}
	endpointID = resolved.EndpointID
	if endpointID == "" {
		return nil, fmt.Errorf("missing endpoint for app")
	}
	if keyToken == "" {
		keyToken = resolved.CredentialRef
	}

	// Get credentials from Key Store if needed
	if keyToken != "" && s.keyStore != nil {
		creds, err := s.keyStore.Get(ctx, keyToken)
		if err != nil {
			s.logger.Warnw("Failed to get credentials", "error", err)
		} else {
			s.logger.Debugw("Got credentials", "endpointId", creds.EndpointID)
			// Inject credentials into call params for UCL
			if call.Params == nil {
				call.Params = make(map[string]any)
			}
			call.Params["_credentials"] = map[string]any{
				"endpoint_id":     creds.EndpointID,
				"credential_type": creds.CredentialType,
				"access_token":    creds.Credentials.AccessToken,
				"api_key":         creds.Credentials.APIKey,
			}
		}
	}

	// Read APIs
	switch call.Action {
	case "list_datasets":
		datasets, err := s.uclClient.ListDatasets(ctx, endpointID)
		if err != nil {
			return nil, err
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"datasets": datasets},
			Message: "Listed datasets",
		}, nil
	case "get_schema":
		datasetID := getStringParam(call.Params, "dataset_id")
		if datasetID == "" {
			return nil, fmt.Errorf("missing dataset_id")
		}
		schema, err := s.uclClient.GetSchema(ctx, endpointID, datasetID)
		if err != nil {
			return nil, err
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"schema": schema},
			Message: "Fetched schema",
		}, nil
	case "read_data":
		datasetID := getStringParam(call.Params, "dataset_id")
		if datasetID == "" {
			return nil, fmt.Errorf("missing dataset_id")
		}
		limit := getInt64Param(call.Params, "limit", 50)
		filter := getMapParam(call.Params, "filter")
		records, err := s.uclClient.ReadData(ctx, endpointID, datasetID, filter, limit)
		if err != nil {
			return nil, err
		}
		return &Result{
			Success: true,
			Data:    map[string]any{"records": records},
			Message: fmt.Sprintf("Read %d records", len(records)),
		}, nil
	}

	// Default: write/action API
	resp, err := s.uclClient.ExecuteAction(ctx, endpointID, call.Action, call.Params, false)
	if err != nil {
		s.logger.Warnw("UCL ExecuteAction failed", "error", err)
		return &Result{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	data := make(map[string]any)
	if resp.Result != nil {
		data = resp.Result
	}
	data["execution_id"] = resp.ExecutionId

	return &Result{
		Success: true,
		Data:    data,
		Message: fmt.Sprintf("Executed %s.%s (id: %s)", call.Name, call.Action, resp.ExecutionId),
	}, nil
}

func resolveEndpointID(call ToolCall) string {
	if call.EndpointID != "" {
		return call.EndpointID
	}
	if call.Params == nil {
		return ""
	}
	if v, ok := call.Params["endpoint_id"].(string); ok {
		return v
	}
	if v, ok := call.Params["endpointId"].(string); ok {
		return v
	}
	return ""
}

func getStringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func getInt64Param(params map[string]any, key string, fallback int64) int64 {
	if params == nil {
		return fallback
	}
	switch v := params[key].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return fallback
	}
}

func getMapParam(params map[string]any, key string) map[string]any {
	if params == nil {
		return nil
	}
	if v, ok := params[key].(map[string]any); ok {
		return v
	}
	return nil
}

// GetToolByName returns a specific tool definition
func (s *Server) GetToolByName(ctx context.Context, name string) (*ToolDefinition, error) {
	tools, err := s.ListTools(ctx, "", "")
	if err != nil {
		return nil, err
	}

	for _, tool := range tools {
		if tool.Name == name || tool.TemplateID == name {
			return &tool, nil
		}
	}
	return nil, fmt.Errorf("tool not found: %s", name)
}
