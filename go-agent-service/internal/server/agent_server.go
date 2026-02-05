// Package server implements the gRPC agent service
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/antigravity/go-agent-service/internal/agent"
	"github.com/antigravity/go-agent-service/internal/appregistry"
	"github.com/antigravity/go-agent-service/internal/config"
	agentctx "github.com/antigravity/go-agent-service/internal/context"
	"github.com/antigravity/go-agent-service/internal/memory"
	"github.com/antigravity/go-agent-service/internal/nucleus"
	"github.com/antigravity/go-agent-service/internal/tools"
	"github.com/antigravity/go-agent-service/internal/ucl"
	"github.com/antigravity/go-agent-service/internal/workflow"
)

// AgentServer implements the gRPC AgentService
type AgentServer struct {
	UnimplementedAgentServiceServer
	config         *config.Config
	logger         *zap.SugaredLogger
	runner         *agent.Runner
	llmRouter      *agent.LLMRouter
	orchestrator   *agentctx.Orchestrator
	memory         memory.Store
	nucleus        *nucleus.Client
	uclRegistry    *ucl.StubToolRegistry
	workflowEngine *workflow.Engine
	tools          []tools.Tool
	toolRegistry   *tools.Registry // Dynamic tool registry (MCP + Store)
	appRegistry    appregistry.Store
	appRegistryDB  *sql.DB
}

// NewAgentServer creates a new agent server instance
func NewAgentServer(cfg *config.Config, logger *zap.SugaredLogger) *AgentServer {
	// Initialize components
	runner := agent.NewRunner(cfg.GeminiAPIKey, logger)
	llmRouter := agent.NewLLMRouter(cfg.GeminiAPIKey, cfg.OpenAIAPIKey)
	memStore := memory.NewShortTermStore()
	nucleusClient := nucleus.NewClient(cfg.NucleusURL, logger)
	orchestrator := agentctx.NewOrchestrator(nucleusClient, logger)

	// Try to initialize episodic memory with pgvector (optional)
	var episodicStore memory.MemoryStore
	if cfg.PostgresURL != "" && cfg.GeminiAPIKey != "" {
		embedder := memory.NewGeminiEmbedder(cfg.GeminiAPIKey)
		store, err := memory.NewEpisodicStore(cfg.PostgresURL, embedder)
		if err != nil {
			logger.Warnw("Failed to initialize episodic memory, using short-term only", "error", err)
		} else {
			episodicStore = store
			logger.Info("Episodic memory initialized with pgvector")

			// Wire memory to runner for context-aware chat
			runner.WithMemory(store, nil)
		}
	}

	// Register tools
	uclTools := []tools.Tool{
		tools.NewJiraTool(),
		tools.NewGitHubTool(),
		tools.NewPagerDutyTool(),
		tools.NewSlackTool(),
	}

	_ = episodicStore // Suppress unused warning until memory implemented

	// Initialize legacy UCL registry (not exposed to LLM tools)
	uclRegistry := ucl.NewStubToolRegistry(logger)
	logger.Infow("UCL tools loaded", "actions", uclRegistry.GetActions())

	// Initialize Workflow Engine
	wfEngine, err := workflow.NewEngine(cfg.TemporalHost, logger)
	if err != nil {
		logger.Warnw("Failed to initialize Workflow Engine", "error", err)
	}

	// Initialize Tool Registry (MCP/UCL + Nucleus + Store)
	toolRegistry := tools.NewRegistry(cfg, logger)
	// Connect to backend services (UCL, etc.)
	if err := toolRegistry.Connect(context.Background()); err != nil {
		logger.Warnw("Tool registry connection failed", "error", err)
	}
	logger.Infow("Tool registry initialized")

	var appRegistry appregistry.Store
	var appRegistryDB *sql.DB
	if cfg.PostgresURL != "" {
		db, err := sql.Open("postgres", cfg.PostgresURL)
		if err != nil {
			logger.Warnw("Failed to connect to Postgres for app registry", "error", err)
		} else {
			appRegistry = appregistry.NewPostgresStore(db)
			appRegistryDB = db
		}
	}

	return &AgentServer{
		config:         cfg,
		logger:         logger,
		runner:         runner,
		llmRouter:      llmRouter,
		orchestrator:   orchestrator,
		memory:         memStore,
		nucleus:        nucleusClient,
		uclRegistry:    uclRegistry,
		workflowEngine: wfEngine,
		tools:          uclTools,
		toolRegistry:   toolRegistry,
		appRegistry:    appRegistry,
		appRegistryDB:  appRegistryDB,
	}
}

// GetWorkflowEngine returns the workflow engine instance
func (s *AgentServer) GetWorkflowEngine() *workflow.Engine {
	return s.workflowEngine
}

// GetToolRegistry returns the tool registry instance
func (s *AgentServer) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}

// GetAppRegistry returns the app registry store instance (if configured).
func (s *AgentServer) GetAppRegistry() appregistry.Store {
	return s.appRegistry
}

// GetNucleusClient returns the Nucleus client instance.
func (s *AgentServer) GetNucleusClient() *nucleus.Client {
	return s.nucleus
}

// Chat handles a chat request
func (s *AgentServer) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	provider := req.GetProvider()
	model := req.GetModel()

	s.logger.Infow("Chat request received",
		"query", req.Query,
		"conversation_id", req.ConversationId,
		"provider", provider,
		"model", model,
	)

	// Convert history from proto to agent format
	history := make([]agent.HistoryMessage, 0, len(req.History))
	for _, h := range req.History {
		history = append(history, agent.HistoryMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	// Build context from Knowledge Graph
	agentCtx, err := s.orchestrator.Process(ctx, req.Query, req.ContextEntities)
	if err != nil {
		s.logger.Warnw("Context processing failed", "error", err)
		// Continue without KG context
	}
	s.logger.Debugw("Context built", "entities", len(agentCtx.Entities), "nodes", len(agentCtx.RetrievedNodes))

	// Build system prompt with Knowledge Graph context and UCL tools
	kgContext := ""
	if agentCtx != nil {
		kgContext = agentCtx.FormatForLLM()
	}

	// Get dynamic tools from registry (MCP + Store)
	dynamicToolsPrompt := s.formatDynamicToolsForLLM(ctx)

	systemPrompt := "You are a helpful AI assistant for developers. Be concise and helpful."
	if kgContext != "" {
		systemPrompt = systemPrompt + "\n\n" + kgContext
	}
	if dynamicToolsPrompt != "" {
		systemPrompt = systemPrompt + "\n\n## Available Tools\n" + dynamicToolsPrompt
	}

	// Use LLM router for provider selection
	response, err := s.llmRouter.GenerateResponse(ctx, provider, model, req.Query, systemPrompt, history)
	if err != nil {
		s.logger.Errorw("LLM router failed", "error", err, "provider", provider)
		return nil, err
	}

	// Mock artifact generation for workflow creation (for UI verification)
	var artifacts []*Artifact
	if strings.Contains(strings.ToLower(req.Query), "create workflow") || strings.Contains(strings.ToLower(req.Query), "draft workflow") {
		artifacts = append(artifacts, &Artifact{
			Id:       fmt.Sprintf("art-%d", time.Now().UnixNano()),
			Type:     "workflow_draft",
			Title:    "Workflow Draft",
			Content:  `{"summary": "Data Pipeline Workflow", "schedule": "@daily", "status": "draft", "yaml": "name: data-pipeline\nsteps:\n  - name: fetch-data\n    action: ucl.fetch\n"}`,
			Language: stringPtr("json"),
		})
		response += "\n\nI've created a draft workflow for you. You can review it below."
	}

	return &ChatResponse{
		Response:  response,
		Reasoning: nil,
		Artifacts: artifacts,
		Citations: nil,
	}, nil
}

func stringPtr(s string) *string {
	return &s
}

// StreamChat handles a streaming chat request
func (s *AgentServer) StreamChat(req *ChatRequest, stream AgentService_StreamChatServer) error {
	s.logger.Infow("Stream chat request received", "query", req.Query)

	// Run agent
	agentReq := &agent.ChatRequest{
		Query:           req.Query,
		ConversationID:  req.ConversationId,
		ContextEntities: req.ContextEntities,
	}

	// Use context.Background() as stream context doesn't implement full Context interface
	ctx := context.Background()
	agentResp, err := s.runner.Chat(ctx, agentReq)
	if err != nil {
		return err
	}

	// Stream reasoning steps
	for _, step := range agentResp.Reasoning {
		durationMs := step.DurationMs
		if err := stream.Send(&ChatChunk{
			Reasoning: &ReasoningStep{
				Step:       int32(step.Step),
				Type:       step.Type,
				Content:    step.Content,
				DurationMs: &durationMs,
			},
		}); err != nil {
			return err
		}
	}

	// Stream response
	words := splitWords(agentResp.Response)
	for i, word := range words {
		if err := stream.Send(&ChatChunk{
			Content: word + " ",
			Done:    i == len(words)-1,
		}); err != nil {
			return err
		}
	}

	return nil
}

// ExecuteAction handles an action execution request
func (s *AgentServer) ExecuteAction(ctx context.Context, req *ActionRequest) (*ActionResponse, error) {
	s.logger.Infow("Action request received",
		"action_type", req.ActionType,
		"entity_id", req.EntityId,
	)

	// 1. Try unified Tool Registry first (Dynamic/Unified path)
	if s.toolRegistry != nil {
		// Parse ActionType logic
		// If it's a "classic" tool (e.g. jira.search), it splits.
		toolName, actionName := parseToolAction(req.ActionType)

		// Parse Payload (JSON string to Map)
		params := parsePayload(req.Payload)
		if params == nil {
			params = make(map[string]any)
		}
		// If payload didn't specify action, use the one from ActionType
		if _, ok := params["action"]; !ok && actionName != "" {
			params["action"] = actionName
		}

		// Execute
		result, err := s.toolRegistry.Execute(ctx, toolName, actionName, params)
		if err == nil {
			return &ActionResponse{
				Success:    result.Success,
				ActionType: req.ActionType,
				EntityId:   req.EntityId,
				Message:    result.Message,
			}, nil
		}

		// If error is "unknown tool", match legacy s.tools.
		// But Execute() usually returns error for other reasons too.
		// We'll fallback only if tool is not found.
		if !strings.Contains(err.Error(), "unknown tool") {
			return nil, err
		}
	}

	// 2. Legacy fallback
	// Find matching tool
	for _, tool := range s.tools {
		if matchesTool(req.ActionType, tool.Name()) {
			result, err := tool.Execute(ctx, map[string]any{
				"action":    req.ActionType,
				"entity_id": req.EntityId,
				"payload":   req.Payload,
			})
			if err != nil {
				return nil, err
			}

			return &ActionResponse{
				Success:    result.Success,
				ActionType: req.ActionType,
				EntityId:   req.EntityId,
				Message:    result.Message,
			}, nil
		}
	}

	return &ActionResponse{
		Success:    false,
		ActionType: req.ActionType,
		EntityId:   req.EntityId,
		Message:    "No matching tool found",
	}, nil
}

func parseToolAction(actionType string) (toolName, actionName string) {
	// Heuristic: Last segment is action, split by dot.
	// E.g. "nucleus_search.list_projects" -> tool "nucleus_search", action "list_projects"
	// E.g. "http.jira.create_issue" -> tool "http.jira", action "create_issue"
	// E.g. "store" -> tool "store", action ""

	lastDot := strings.LastIndex(actionType, ".")
	if lastDot == -1 {
		return actionType, ""
	}
	return actionType[:lastDot], actionType[lastDot+1:]
}

func parsePayload(payload string) map[string]any {
	if payload == "" {
		return nil
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(payload), &params); err != nil {
		return nil
	}
	return params
}

func splitWords(s string) []string {
	words := []string{}
	current := ""
	for _, c := range s {
		if c == ' ' || c == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		words = append(words, current)
	}
	return words
}

func matchesTool(actionType, toolName string) bool {
	if len(actionType) >= len(toolName) {
		return actionType[:len(toolName)] == toolName
	}
	return false
}

// formatDynamicToolsForLLM formats dynamic tools from registry for LLM system prompt
func (s *AgentServer) formatDynamicToolsForLLM(ctx context.Context) string {
	if s.toolRegistry == nil {
		return ""
	}

	userID, projectID := getUserProject(ctx)
	toolDefs := s.toolRegistry.ListToolsFor(ctx, userID, projectID)
	if len(toolDefs) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, t := range toolDefs {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n", t.Name, t.Description))
		if len(t.Actions) > 0 {
			sb.WriteString("Actions:\n")
			for _, action := range t.Actions {
				sb.WriteString(fmt.Sprintf("- %s: %s", action.Name, action.Description))
				if action.InputSchema != "" {
					sb.WriteString(fmt.Sprintf(" Schema: %s", action.InputSchema))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
