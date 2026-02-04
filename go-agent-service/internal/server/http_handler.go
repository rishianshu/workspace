// Package server provides HTTP handlers for the agent service
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/antigravity/go-agent-service/internal/appregistry"
	"github.com/antigravity/go-agent-service/internal/workflow"
	"go.uber.org/zap"
)

// HTTPHandler wraps the AgentServer for HTTP requests
type HTTPHandler struct {
	agent  *AgentServer
	logger *zap.SugaredLogger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(agent *AgentServer, logger *zap.SugaredLogger) *HTTPHandler {
	return &HTTPHandler{
		agent:  agent,
		logger: logger,
	}
}

// ChatHTTPRequest matches the rust-gateway request format
type ChatHTTPRequest struct {
	Query           string           `json:"query"`
	ConversationID  string           `json:"conversation_id"`
	ContextEntities []string         `json:"context_entities"`
	SessionID       *string          `json:"session_id,omitempty"`
	Provider        *string          `json:"provider,omitempty"`
	Model           *string          `json:"model,omitempty"`
	UserID          *string          `json:"userId,omitempty"`
	ProjectID       *string          `json:"projectId,omitempty"`
	History         []HistoryMessage `json:"history"`
	AttachedFiles   []AttachedFile   `json:"attachedFiles,omitempty"`
}

// AttachedFile represents a file attached by the user (HTTP layer).
type AttachedFile struct {
	Name     string `json:"name"`
	FileType string `json:"type"`
	Content  string `json:"content"`
}

// ChatHTTPResponse matches the rust-gateway response format
type ChatHTTPResponse struct {
	Response  string              `json:"response"`
	Reasoning []ReasoningStepJSON `json:"reasoning"`
	Artifacts []ArtifactJSON      `json:"artifacts,omitempty"`
	Citations []string            `json:"citations,omitempty"`
}

// ReasoningStepJSON for HTTP JSON response
type ReasoningStepJSON struct {
	Step       int32  `json:"step"`
	Type       string `json:"type"`
	Content    string `json:"content"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
}

// ArtifactJSON for HTTP JSON response
type ArtifactJSON struct {
	ID       string  `json:"id"`
	Type     string  `json:"type"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Language *string `json:"language,omitempty"`
}

// HandleChat handles HTTP POST /chat requests
func (h *HTTPHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorw("Failed to decode request", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.logger.Infow("HTTP Chat request",
		"query", req.Query,
		"conversation_id", req.ConversationID,
		"provider", req.Provider,
		"model", req.Model,
	)

	// If attached files exist, embed them in the query for LLM context.
	if len(req.AttachedFiles) > 0 {
		var fileParts []string
		for _, f := range req.AttachedFiles {
			label := f.Name
			if f.FileType != "" {
				label = fmt.Sprintf("%s (%s)", f.Name, f.FileType)
			}
			fileParts = append(fileParts, fmt.Sprintf("File: %s\n%s", label, f.Content))
		}
		fileContext := strings.Join(fileParts, "\n\n")
		req.Query = fmt.Sprintf("The user has attached the following file(s) for analysis:\n\n%s\n\n---\nUser query: %s", fileContext, req.Query)
	}

	// Convert to gRPC request format
	grpcReq := &ChatRequest{
		Query:           req.Query,
		ConversationId:  req.ConversationID,
		ContextEntities: req.ContextEntities,
	}

	if req.SessionID != nil {
		grpcReq.SessionId = req.SessionID
	}
	if req.Provider != nil {
		grpcReq.Provider = req.Provider
	}
	if req.Model != nil {
		grpcReq.Model = req.Model
	}

	// Convert history
	for i := range req.History {
		h := &req.History[i]
		grpcReq.History = append(grpcReq.History, &HistoryMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	ctx := r.Context()
	if req.UserID != nil || req.ProjectID != nil {
		userID := ""
		projectID := ""
		if req.UserID != nil {
			userID = *req.UserID
		}
		if req.ProjectID != nil {
			projectID = *req.ProjectID
		}
		ctx = withUserProject(ctx, userID, projectID)
	}

	// Call the gRPC handler internally
	resp, err := h.agent.Chat(ctx, grpcReq)
	if err != nil {
		h.logger.Errorw("Chat failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert response to JSON format
	httpResp := ChatHTTPResponse{
		Response:  resp.Response,
		Reasoning: make([]ReasoningStepJSON, 0),
		Citations: resp.Citations,
	}

	for _, r := range resp.Reasoning {
		httpResp.Reasoning = append(httpResp.Reasoning, ReasoningStepJSON{
			Step:       r.Step,
			Type:       r.Type,
			Content:    r.Content,
			DurationMs: r.DurationMs,
		})
	}

	for _, a := range resp.Artifacts {
		httpResp.Artifacts = append(httpResp.Artifacts, ArtifactJSON{
			ID:       a.Id,
			Type:     a.Type,
			Title:    a.Title,
			Content:  a.Content,
			Language: a.Language,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(httpResp)
}

// HandleListWorkflows handles GET /workflows
func (h *HTTPHandler) HandleListWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.agent.GetWorkflowEngine() == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	executions, err := h.agent.GetWorkflowEngine().ListWorkflows(r.Context())
	if err != nil {
		h.logger.Errorw("Failed to list workflows", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

// HandleCancelWorkflow handles POST /workflows/cancel
func (h *HTTPHandler) HandleCancelWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.agent.GetWorkflowEngine() == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		ExecutionID string `json:"execution_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := h.agent.GetWorkflowEngine().CancelWorkflow(r.Context(), req.ExecutionID); err != nil {
		h.logger.Errorw("Failed to cancel workflow", "id", req.ExecutionID, "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success": true}`))
}

// HandleCreateWorkflow handles POST /workflows/create
func (h *HTTPHandler) HandleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.agent.GetWorkflowEngine() == nil {
		http.Error(w, "Workflow engine not available", http.StatusServiceUnavailable)
		return
	}

	var definition workflow.WorkflowDefinition
	if err := json.NewDecoder(r.Body).Decode(&definition); err != nil {
		h.logger.Errorw("Failed to decode workflow definition", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	execution, err := h.agent.GetWorkflowEngine().CreateWorkflow(r.Context(), &definition)
	if err != nil {
		h.logger.Errorw("Failed to create workflow", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// ========================
// Tools API Handlers
// ========================

// HandleListTools handles GET /tools - lists all available tools
func (h *HTTPHandler) HandleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	userID := query.Get("userId")
	projectID := query.Get("projectId")
	var toolsList interface{}
	if userID != "" || projectID != "" {
		toolsList = h.agent.GetToolRegistry().ListToolsFor(r.Context(), userID, projectID)
	} else {
		toolsList = h.agent.GetToolRegistry().ListTools(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toolsList)
}

// ExecuteRequest for HTTP API
type ExecuteRequest struct {
	Name       string         `json:"name"`
	Action     string         `json:"action"`
	Params     map[string]any `json:"params"`
	EndpointID string         `json:"endpointId,omitempty"`
	KeyToken   string         `json:"keyToken,omitempty"`
	UserID     string         `json:"userId,omitempty"`
	ProjectID  string         `json:"projectId,omitempty"`
}

// ActionHTTPRequest for legacy /action endpoint
type ActionHTTPRequest struct {
	ActionType string `json:"action_type"`
	EntityID   string `json:"entity_id"`
	Payload    string `json:"payload"`
}

// HandleExecuteTool handles POST /tools/execute
func (h *HTTPHandler) HandleExecuteTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorw("Failed to decode execute request", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.logger.Infow("Executing tool", "name", req.Name, "action", req.Action)

	// Add action to params
	if req.Params == nil {
		req.Params = make(map[string]any)
	}
	req.Params["action"] = req.Action
	if req.EndpointID != "" {
		req.Params["endpointId"] = req.EndpointID
	}
	if req.KeyToken != "" {
		req.Params["keyToken"] = req.KeyToken
	}
	if req.UserID != "" {
		req.Params["userId"] = req.UserID
	}
	if req.ProjectID != "" {
		req.Params["projectId"] = req.ProjectID
	}

	result, err := h.agent.GetToolRegistry().Execute(r.Context(), req.Name, req.Action, req.Params)
	if err != nil {
		h.logger.Errorw("Tool execution failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleExecuteAction handles POST /action (legacy)
func (h *HTTPHandler) HandleExecuteAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ActionHTTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorw("Failed to decode action request", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	resp, err := h.agent.ExecuteAction(r.Context(), &ActionRequest{
		ActionType: req.ActionType,
		EntityId:   req.EntityID,
		Payload:    req.Payload,
	})
	if err != nil {
		h.logger.Errorw("Action execution failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// BrainSearchRequest for HTTP API
type BrainSearchRequest struct {
	Query     string `json:"query"`
	ProjectID string `json:"projectId,omitempty"`
}

// App registry HTTP requests
type AppInstanceRequest struct {
	ID          string         `json:"id,omitempty"`
	TemplateID  string         `json:"templateId"`
	InstanceKey string         `json:"instanceKey"`
	DisplayName string         `json:"displayName,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

type UserAppRequest struct {
	ID            string `json:"id,omitempty"`
	UserID        string `json:"userId"`
	AppInstanceID string `json:"appInstanceId"`
	CredentialRef string `json:"credentialRef,omitempty"`
}

type ProjectAppRequest struct {
	ID         string `json:"id,omitempty"`
	ProjectID  string `json:"projectId"`
	UserAppID  string `json:"userAppId"`
	EndpointID string `json:"endpointId"`
	Alias      string `json:"alias,omitempty"`
	IsDefault  bool   `json:"isDefault,omitempty"`
}

// HandleBrainSearch handles POST /brain/search - calls Nucleus brain search
func (h *HTTPHandler) HandleBrainSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BrainSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Errorw("Failed to decode brain search request", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	h.logger.Infow("Brain search", "query", req.Query, "projectId", req.ProjectID)

	// Call nucleus_search tool with brain_search action
	params := map[string]any{
		"action":    "brain_search",
		"query":     req.Query,
		"projectId": req.ProjectID,
	}

	result, err := h.agent.GetToolRegistry().Execute(r.Context(), "nucleus_search", "brain_search", params)
	if err != nil {
		h.logger.Errorw("Brain search failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleListProjects handles GET /projects
func (h *HTTPHandler) HandleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Call nucleus_search tool with list_projects action
	params := map[string]any{
		"action": "list_projects",
	}

	result, err := h.agent.GetToolRegistry().Execute(r.Context(), "nucleus_search", "list_projects", params)
	if err != nil {
		h.logger.Errorw("List projects failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ========================
// App Registry Handlers
// ========================

func (h *HTTPHandler) HandleAppInstances(w http.ResponseWriter, r *http.Request) {
	store := h.agent.GetAppRegistry()
	if store == nil {
		http.Error(w, "App registry unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query()
		id := query.Get("id")
		templateID := query.Get("templateId")
		instanceKey := query.Get("instanceKey")

		var instance *appregistry.AppInstance
		var err error
		if id != "" {
			instance, err = store.GetAppInstance(r.Context(), id)
		} else if templateID != "" && instanceKey != "" {
			instance, err = store.FindAppInstance(r.Context(), templateID, instanceKey)
		} else {
			http.Error(w, "Missing id or templateId+instanceKey", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(instance)
	case http.MethodPost:
		var req AppInstanceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if req.TemplateID == "" || req.InstanceKey == "" {
			http.Error(w, "templateId and instanceKey are required", http.StatusBadRequest)
			return
		}
		instance, err := store.UpsertAppInstance(r.Context(), appregistry.AppInstance{
			ID:          req.ID,
			TemplateID:  req.TemplateID,
			InstanceKey: req.InstanceKey,
			DisplayName: req.DisplayName,
			Config:      req.Config,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(instance)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) HandleUserApps(w http.ResponseWriter, r *http.Request) {
	store := h.agent.GetAppRegistry()
	if store == nil {
		http.Error(w, "App registry unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		userID := r.URL.Query().Get("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}
		apps, err := store.ListUserApps(r.Context(), userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apps)
	case http.MethodPost:
		var req UserAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if req.UserID == "" || req.AppInstanceID == "" {
			http.Error(w, "userId and appInstanceId are required", http.StatusBadRequest)
			return
		}
		app, err := store.UpsertUserApp(r.Context(), appregistry.UserApp{
			ID:            req.ID,
			UserID:        req.UserID,
			AppInstanceID: req.AppInstanceID,
			CredentialRef: req.CredentialRef,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPHandler) HandleProjectApps(w http.ResponseWriter, r *http.Request) {
	store := h.agent.GetAppRegistry()
	if store == nil {
		http.Error(w, "App registry unavailable", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query()
		projectID := query.Get("projectId")
		userID := query.Get("userId")
		if projectID == "" {
			http.Error(w, "projectId is required", http.StatusBadRequest)
			return
		}
		var (
			apps []*appregistry.ProjectApp
			err  error
		)
		if userID != "" {
			apps, err = store.ListProjectAppsForUser(r.Context(), projectID, userID)
		} else {
			apps, err = store.ListProjectApps(r.Context(), projectID)
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(apps)
	case http.MethodPost:
		var req ProjectAppRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		if req.ProjectID == "" || req.UserAppID == "" || req.EndpointID == "" {
			http.Error(w, "projectId, userAppId, and endpointId are required", http.StatusBadRequest)
			return
		}
		app, err := store.UpsertProjectApp(r.Context(), appregistry.ProjectApp{
			ID:         req.ID,
			ProjectID:  req.ProjectID,
			UserAppID:  req.UserAppID,
			EndpointID: req.EndpointID,
			Alias:      req.Alias,
			IsDefault:  req.IsDefault,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(app)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
