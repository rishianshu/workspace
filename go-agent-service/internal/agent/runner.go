// Package agent implements the ADK agent runner
package agent

import (
	"context"
	"fmt"
	"time"

	agentctx "github.com/antigravity/go-agent-service/internal/context"
	"github.com/antigravity/go-agent-service/internal/memory"
	"go.uber.org/zap"
)

// Runner manages the ADK agent execution
type Runner struct {
	logger         *zap.SugaredLogger
	apiKey         string
	modelName      string
	tools          []Tool
	geminiClient   *GeminiClient
	memoryStore    memory.MemoryStore
	contextBuilder *agentctx.Builder
}

// Tool represents a callable tool for the agent
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (*ToolResult, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool
	Data    map[string]any
	Message string
}

// ReasoningStep represents a step in the agent's reasoning process
type ReasoningStep struct {
	Step      int    `json:"step"`
	Type      string `json:"type"` // retrieval, analysis, synthesis, action
	Content   string `json:"content"`
	DurationMs int64 `json:"duration_ms,omitempty"`
}

// ChatRequest represents an incoming chat request
type ChatRequest struct {
	Query           string   `json:"query"`
	ConversationID  string   `json:"conversation_id"`
	ContextEntities []string `json:"context_entities,omitempty"`
	SessionID       string   `json:"session_id,omitempty"`
}

// ChatResponse represents the agent's response
type ChatResponse struct {
	Response        string          `json:"response"`
	Reasoning       []ReasoningStep `json:"reasoning"`
	Artifacts       []Artifact      `json:"artifacts,omitempty"`
	Citations       []string        `json:"citations,omitempty"`
	ProposedActions []Action        `json:"proposed_actions,omitempty"`
}

// Artifact represents a generated artifact
type Artifact struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // code, diff, markdown, yaml, json
	Title    string `json:"title"`
	Content  string `json:"content"`
	Language string `json:"language,omitempty"`
}

// Action represents a proposed action
type Action struct {
	ID          string `json:"id"`
	Type        string `json:"type"` // approve, execute, confirm
	Title       string `json:"title"`
	Description string `json:"description"`
}

// NewRunner creates a new agent runner
func NewRunner(apiKey string, logger *zap.SugaredLogger) *Runner {
	r := &Runner{
		logger:    logger,
		apiKey:    apiKey,
		modelName: "gemini-2.0-flash",
		tools:     make([]Tool, 0),
	}
	
	// Initialize Gemini client if API key provided
	if apiKey != "" {
		r.geminiClient = NewGeminiClient(apiKey)
	}
	
	return r
}

// WithMemory sets the memory store for the runner
func (r *Runner) WithMemory(store memory.MemoryStore, config *memory.ContextConfig) *Runner {
	r.memoryStore = store
	if config == nil {
		config = memory.DefaultContextConfig()
	}
	r.contextBuilder = agentctx.NewBuilder(store, config)
	return r
}

// RegisterTool adds a tool to the agent
func (r *Runner) RegisterTool(tool Tool) {
	r.tools = append(r.tools, tool)
	r.logger.Infow("Registered tool", "name", tool.Name())
}

// Chat processes a chat request and returns a response
func (r *Runner) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	r.logger.Infow("Processing chat request",
		"query", req.Query,
		"conversation_id", req.ConversationID,
	)

	// Build reasoning steps
	reasoning := []ReasoningStep{}

	// Step 1: Analyze query
	reasoning = append(reasoning, ReasoningStep{
		Step:       1,
		Type:       "analysis",
		Content:    fmt.Sprintf("Analyzing query: %s", req.Query),
		DurationMs: 50,
	})

	// Determine session ID
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = req.ConversationID
	}

	// Step 2: Memory-based context retrieval (if memory available)
	if r.memoryStore != nil && sessionID != "" {
		reasoning = append(reasoning, ReasoningStep{
			Step:       2,
			Type:       "retrieval",
			Content:    "Searching memory for relevant context",
			DurationMs: 100,
		})

		// Store the user turn
		userTurn := &memory.Turn{
			SessionID: sessionID,
			Role:      "user",
			Content:   req.Query,
			CreatedAt: time.Now(),
		}
		if err := r.memoryStore.AddTurn(ctx, userTurn); err != nil {
			r.logger.Warnw("Failed to store user turn", "error", err)
		}

		// Build context using memory
		if r.contextBuilder != nil {
			contextStr, err := r.contextBuilder.Build(ctx, sessionID, req.Query)
			if err != nil {
				r.logger.Warnw("Failed to build context", "error", err)
			} else {
				r.logger.Debugw("Built context", "length", len(contextStr))
			}
		}
	} else if len(req.ContextEntities) > 0 {
		// Fallback: Retrieve context from entities
		reasoning = append(reasoning, ReasoningStep{
			Step:       2,
			Type:       "retrieval",
			Content:    fmt.Sprintf("Retrieving context for %d entities", len(req.ContextEntities)),
			DurationMs: 100,
		})
	}

	// Step 3: Synthesize response
	reasoning = append(reasoning, ReasoningStep{
		Step:       len(reasoning) + 1,
		Type:       "synthesis",
		Content:    "Generating response based on analysis",
		DurationMs: 200,
	})

	// Generate response (pattern matching for now)
	response := r.generateResponse(req.Query)

	// Store agent turn in memory
	if r.memoryStore != nil && sessionID != "" {
		agentTurn := &memory.Turn{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   response.text,
			CreatedAt: time.Now(),
		}
		if err := r.memoryStore.AddTurn(ctx, agentTurn); err != nil {
			r.logger.Warnw("Failed to store agent turn", "error", err)
		}

		// Update session turn count
		session, _ := r.memoryStore.GetSession(ctx, sessionID)
		if session != nil {
			session.TurnCount++
			session.LastActivity = time.Now()
			r.memoryStore.UpdateSession(ctx, session)
		}
	}

	return &ChatResponse{
		Response:  response.text,
		Reasoning: reasoning,
		Artifacts: response.artifacts,
		Citations: response.citations,
	}, nil
}

type generatedResponse struct {
	text      string
	artifacts []Artifact
	citations []string
}

func (r *Runner) generateResponse(query string) generatedResponse {
	// Match query patterns to scenarios (like the existing agent-scenarios.ts)
	
	// Bug fix pattern
	if containsAny(query, []string{"bug", "fix", "error", "login", "401"}) {
		return generatedResponse{
			text: "I've analyzed the login error and found the issue in the authentication flow. The session token validation is failing due to an incorrect expiry check. Here's my proposed fix:",
			artifacts: []Artifact{
				{
					ID:       "fix-001",
					Type:     "code",
					Title:    "auth.ts fix",
					Content:  "// Fix: Correct token expiry validation\nfunction validateToken(token: string): boolean {\n  const decoded = jwt.decode(token);\n  const now = Math.floor(Date.now() / 1000);\n  return decoded.exp > now; // Fixed: was using < instead of >\n}",
					Language: "typescript",
				},
			},
			citations: []string{"[MOBILE-1234]", "[auth.ts:45]"},
		}
	}

	// PR review pattern
	if containsAny(query, []string{"review", "pr", "pull request", "changes"}) {
		return generatedResponse{
			text: "I've reviewed the pull request and found 2 potential issues:\n\n1. Missing null check on line 23\n2. Potential performance issue with nested loops\n\nOverall the changes look good with minor improvements needed.",
			artifacts: []Artifact{
				{
					ID:      "review-001",
					Type:    "markdown",
					Title:   "PR Review Comments",
					Content: "## Review Summary\n\n### Issues Found\n- [ ] Add null check for `user` object\n- [ ] Consider using `Map` instead of nested array lookup\n\n### Approved with changes",
				},
			},
			citations: []string{"[PR-4423]"},
		}
	}

	// Documentation pattern
	if containsAny(query, []string{"doc", "documentation", "api", "spec"}) {
		return generatedResponse{
			text: "I've generated the API documentation based on the auth module:",
			artifacts: []Artifact{
				{
					ID:       "doc-001",
					Type:     "yaml",
					Title:    "API Documentation",
					Content:  "openapi: 3.0.0\ninfo:\n  title: Auth API\n  version: 1.0.0\npaths:\n  /login:\n    post:\n      summary: User login\n      requestBody:\n        content:\n          application/json:\n            schema:\n              type: object\n              properties:\n                email:\n                  type: string\n                password:\n                  type: string",
					Language: "yaml",
				},
			},
			citations: []string{"[auth.ts]"},
		}
	}

	// Workflow synthesis pattern
	if containsAny(query, []string{"workflow", "automate", "schedule", "every morning", "cron", "alert me"}) {
		workflowYAML := generateWorkflowYAML(query)
		return generatedResponse{
			text: "I've synthesized a workflow based on your request. Here's the YAML definition for your review and approval:",
			artifacts: []Artifact{
				{
					ID:       "workflow-001",
					Type:     "yaml",
					Title:    "Workflow Definition",
					Content:  workflowYAML,
					Language: "yaml",
				},
			},
			citations: nil,
		}
	}

	// Default response
	return generatedResponse{
		text:      fmt.Sprintf("I understand you're asking about: %s. Let me help you with that.", query),
		artifacts: nil,
		citations: nil,
	}
}

func generateWorkflowYAML(intent string) string {
	name := "Custom Workflow"
	schedule := "event: manual"
	
	lower := toLower(intent)
	if contains(lower, "bug") || contains(lower, "critical") {
		name = "Daily Bug Scanner"
	}
	if contains(lower, "morning") || contains(lower, "9") {
		schedule = "schedule: \"0 9 * * *\"  # Daily at 9 AM"
	} else if contains(lower, "hour") {
		schedule = "schedule: \"0 * * * *\"  # Every hour"
	}

	yaml := "# Auto-generated workflow\nname: " + name + "\n\ntrigger:\n  " + schedule + "\n\nsteps:\n"
	
	stepNum := 1
	if contains(lower, "bug") || contains(lower, "ticket") || contains(lower, "jira") || contains(lower, "critical") {
		yaml += fmt.Sprintf("  - id: step%d\n    action: ucl.jira.search\n    params:\n      query: \"priority = Critical AND status = Open\"\n\n", stepNum)
		stepNum++
	}
	
	if contains(lower, "slack") || contains(lower, "notify") || contains(lower, "alert") {
		dependsOn := ""
		if stepNum > 1 {
			dependsOn = fmt.Sprintf("    depends_on: [step%d]\n", stepNum-1)
		}
		yaml += fmt.Sprintf("  - id: step%d\n    action: ucl.slack.post\n%s    params:\n      channel: \"#dev-alerts\"\n      body: |\n        🚨 Daily Bug Report\n        {{ if step1.data.tickets }}\n        Found {{ len step1.data.tickets }} critical tickets\n        {{ end }}\n", stepNum, dependsOn)
		stepNum++
	}
	
	if stepNum == 1 {
		yaml += "  - id: step1\n    action: log.info\n    params:\n      message: \"Workflow executed\"\n"
	}
	
	return yaml
}

func containsAny(s string, substrs []string) bool {
	lower := toLower(s)
	for _, sub := range substrs {
		if contains(lower, toLower(sub)) {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr) >= 0)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
