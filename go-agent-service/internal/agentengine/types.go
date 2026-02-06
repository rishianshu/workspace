// Package agentengine defines shared types for the agent runtime.
package agentengine

import (
	"context"
	"time"
)

// Request is the normalized input for the agent engine.
type Request struct {
	Query           string
	SessionID       string
	UserID          string
	ProjectID       string
	ContextEntities []string
	History         []HistoryMessage
	Provider        string
	Model           string
}

// HistoryMessage is a normalized chat history entry.
type HistoryMessage struct {
	Role    string
	Content string
}

// Response represents the agent output.
type Response struct {
	Text         string
	Provider     string
	Model        string
	Observations []Observation
	Trace        *Trace
}

// PlanType describes the planner decision.
type PlanType string

const (
	PlanDirect            PlanType = "direct"
	PlanToolCalls         PlanType = "tool_calls"
	PlanNeedClarification PlanType = "need_clarification"
)

// Plan describes a planner decision.
type Plan struct {
	Type          PlanType
	ToolCalls     []ToolCall
	Clarification string
}

// PlanInput is passed to the planner.
type PlanInput struct {
	Request      Request
	Prompt       string
	Tools        []ToolDef
	Observations []Observation
	Step         int
}

// ToolDef describes a tool available to the agent.
type ToolDef struct {
	Name        string
	Description string
	Actions     []ToolAction
}

// ToolAction describes a single action within a tool.
type ToolAction struct {
	Name        string
	Description string
	InputSchema string
}

// ToolCall is a structured tool invocation.
type ToolCall struct {
	Name   string
	Action string
	Args   map[string]any
}

// ToolResult is the structured result of a tool invocation.
type ToolResult struct {
	Success bool
	Data    map[string]any
	Message string
}

// Observation records the result of a tool call.
type Observation struct {
	ToolName string
	Result   *ToolResult
	Error    string
}

// LLMRequest is the payload for LLM inference.
type LLMRequest struct {
	Query        string
	Prompt       string
	Observations []Observation
	History      []HistoryMessage
	Provider     string
	Model        string
}

// LLMResponse is the output of LLM inference.
type LLMResponse struct {
	Text     string
	Provider string
	Model    string
}

// Planner decides whether and how to use tools.
type Planner interface {
	Plan(ctx context.Context, input PlanInput) (Plan, error)
}

// LLMClient generates a response from a prompt.
type LLMClient interface {
	Respond(ctx context.Context, input LLMRequest) (LLMResponse, error)
}

// ToolRegistry provides available tools for a user/project.
type ToolRegistry interface {
	ListTools(ctx context.Context, userID, projectID string) ([]ToolDef, error)
}

// ToolExecutor executes a tool call.
type ToolExecutor interface {
	Execute(ctx context.Context, call ToolCall) (*ToolResult, error)
}

// MemoryStore persists turns and facts.
type MemoryStore interface {
	AddTurn(ctx context.Context, sessionID, content, role string, timestamp time.Time) error
	StoreFact(ctx context.Context, sessionID string, observation Observation) error
}

// ContextAssembler builds prompt context.
type ContextAssembler interface {
	Build(ctx context.Context, req Request, tools []ToolDef) (string, error)
	AppendObservations(prompt string, observations []Observation) (string, error)
}

// Policy controls tool access and budgets.
type Policy interface {
	AllowTool(name string) bool
}
