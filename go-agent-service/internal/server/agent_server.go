// Package server implements the gRPC agent service
package server

import (
	"context"

	"go.uber.org/zap"

	"github.com/antigravity/go-agent-service/internal/agent"
	"github.com/antigravity/go-agent-service/internal/config"
	agentctx "github.com/antigravity/go-agent-service/internal/context"
	"github.com/antigravity/go-agent-service/internal/memory"
	"github.com/antigravity/go-agent-service/internal/nucleus"
	"github.com/antigravity/go-agent-service/internal/tools"
)

// AgentServer implements the gRPC AgentService
type AgentServer struct {
	UnimplementedAgentServiceServer
	config       *config.Config
	logger       *zap.SugaredLogger
	runner       *agent.Runner
	orchestrator *agentctx.Orchestrator
	memory       memory.Store
	nucleus      *nucleus.Client
	tools        []tools.Tool
}

// NewAgentServer creates a new agent server instance
func NewAgentServer(cfg *config.Config, logger *zap.SugaredLogger) *AgentServer {
	// Initialize components
	runner := agent.NewRunner(cfg.GeminiAPIKey, logger)
	orchestrator := agentctx.NewOrchestrator(logger)
	memStore := memory.NewShortTermStore()
	nucleusClient := nucleus.NewClient(cfg.NucleusURL, logger)

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

	_ = episodicStore // Will be used for explicit memory operations

	return &AgentServer{
		config:       cfg,
		logger:       logger,
		runner:       runner,
		orchestrator: orchestrator,
		memory:       memStore,
		nucleus:      nucleusClient,
		tools:        uclTools,
	}
}

// Chat handles a chat request
func (s *AgentServer) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	s.logger.Infow("Chat request received",
		"query", req.Query,
		"conversation_id", req.ConversationId,
	)

	// Build context
	agentCtx := s.orchestrator.Process(req.Query, req.ContextEntities)
	s.logger.Debugw("Context built", "entities", len(agentCtx.Entities))

	// Run agent
	agentReq := &agent.ChatRequest{
		Query:           req.Query,
		ConversationID:  req.ConversationId,
		ContextEntities: req.ContextEntities,
		SessionID:       req.GetSessionId(),
	}

	agentResp, err := s.runner.Chat(ctx, agentReq)
	if err != nil {
		s.logger.Errorw("Agent chat failed", "error", err)
		return nil, err
	}

	// Convert response
	reasoning := make([]*ReasoningStep, len(agentResp.Reasoning))
	for i, step := range agentResp.Reasoning {
		reasoning[i] = &ReasoningStep{
			Step:       int32(step.Step),
			Type:       step.Type,
			Content:    step.Content,
			DurationMs: step.DurationMs,
		}
	}

	artifacts := make([]*Artifact, len(agentResp.Artifacts))
	for i, art := range agentResp.Artifacts {
		lang := art.Language
		artifacts[i] = &Artifact{
			Id:       art.ID,
			Type:     art.Type,
			Title:    art.Title,
			Content:  art.Content,
			Language: &lang,
		}
	}

	return &ChatResponse{
		Response:  agentResp.Response,
		Reasoning: reasoning,
		Artifacts: artifacts,
		Citations: agentResp.Citations,
	}, nil
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
		if err := stream.Send(&ChatChunk{
			Reasoning: &ReasoningStep{
				Step:       int32(step.Step),
				Type:       step.Type,
				Content:    step.Content,
				DurationMs: step.DurationMs,
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
