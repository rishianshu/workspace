// Package agentengine defines the reusable ADK-style agent runtime.
package agentengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Engine orchestrates the ReAct loop.
type Engine struct {
	planner     Planner
	llm         LLMClient
	tools       ToolRegistry
	executor    ToolExecutor
	memory      MemoryStore
	context     ContextAssembler
	policy      Policy
	clock       func() time.Time
	toolTimeout time.Duration
	maxSteps    int
}

// Config wires engine dependencies.
type Config struct {
	Planner     Planner
	LLM         LLMClient
	Tools       ToolRegistry
	Executor    ToolExecutor
	Memory      MemoryStore
	Context     ContextAssembler
	Policy      Policy
	ToolTimeout time.Duration
	MaxSteps    int
	Clock       func() time.Time
}

// NewEngine creates an engine with the provided config.
func NewEngine(cfg Config) (*Engine, error) {
	if cfg.Planner == nil {
		return nil, errors.New("planner is required")
	}
	if cfg.LLM == nil {
		return nil, errors.New("llm client is required")
	}
	if cfg.Tools == nil {
		return nil, errors.New("tool registry is required")
	}
	if cfg.Executor == nil {
		return nil, errors.New("tool executor is required")
	}
	if cfg.Context == nil {
		return nil, errors.New("context assembler is required")
	}
	if cfg.MaxSteps <= 0 {
		cfg.MaxSteps = 4
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.ToolTimeout <= 0 {
		cfg.ToolTimeout = 20 * time.Second
	}

	return &Engine{
		planner:     cfg.Planner,
		llm:         cfg.LLM,
		tools:       cfg.Tools,
		executor:    cfg.Executor,
		memory:      cfg.Memory,
		context:     cfg.Context,
		policy:      cfg.Policy,
		clock:       cfg.Clock,
		toolTimeout: cfg.ToolTimeout,
		maxSteps:    cfg.MaxSteps,
	}, nil
}

// Run executes the ReAct loop and returns a response.
func (e *Engine) Run(ctx context.Context, req Request) (*Response, error) {
	if req.Query == "" {
		return nil, errors.New("query is required")
	}

	trace := NewTrace(req.SessionID)
	tools, err := e.tools.ListTools(ctx, req.UserID, req.ProjectID)
	toolWarning := ""
	if err != nil {
		trace.AddEvent("tools.list.failed", err.Error())
		toolWarning = fmt.Sprintf("Tool discovery failed; proceeding without tools: %s", err.Error())
	}

	prompt, err := e.context.Build(ctx, req, tools)
	if err != nil {
		return nil, err
	}
	if toolWarning != "" {
		prompt = prompt + "\n\n## System Notes\n" + toolWarning
	}

	step := 0
	var observations []Observation
	for step < e.maxSteps {
		step++

		plan, err := e.planner.Plan(ctx, PlanInput{
			Request:      req,
			Prompt:       prompt,
			Tools:        tools,
			Observations: observations,
			Step:         step,
		})
		if err != nil {
			return nil, err
		}

		if plan.Type == PlanDirect {
			reply, err := e.llm.Respond(ctx, LLMRequest{
				Query:        req.Query,
				Prompt:       prompt,
				Observations: observations,
				History:      req.History,
				Provider:     req.Provider,
				Model:        req.Model,
			})
			if err != nil {
				return nil, err
			}
			return e.finalize(ctx, req, reply, observations, trace), nil
		}

		if plan.Type == PlanNeedClarification {
			return e.finalize(ctx, req, LLMResponse{
				Text: plan.Clarification,
			}, observations, trace), nil
		}

		if len(plan.ToolCalls) == 0 {
			return nil, fmt.Errorf("planner returned tool plan with no calls")
		}

		for _, call := range plan.ToolCalls {
			if validationErr := validateToolCall(call, tools); validationErr != "" {
				observations = append(observations, Observation{
					ToolName: call.Name,
					Error:    validationErr,
				})
				continue
			}
			if e.policy != nil && !e.policy.AllowTool(call.Name) {
				observations = append(observations, Observation{
					ToolName: call.Name,
					Error:    "tool blocked by policy",
				})
				continue
			}

			execCtx := ctx
			var cancel context.CancelFunc
			if e.toolTimeout > 0 {
				execCtx, cancel = context.WithTimeout(ctx, e.toolTimeout)
			}
			result, err := e.executor.Execute(execCtx, call)
			if cancel != nil {
				cancel()
			}
			if err != nil {
				observations = append(observations, Observation{
					ToolName: call.Name,
					Error:    err.Error(),
				})
				continue
			}
			observations = append(observations, Observation{
				ToolName: call.Name,
				Result:   result,
			})
		}

		prompt, err = e.context.AppendObservations(prompt, observations)
		if err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max steps exceeded (%d)", e.maxSteps)
}

func (e *Engine) finalize(ctx context.Context, req Request, reply LLMResponse, observations []Observation, trace *Trace) *Response {
	if e.memory != nil {
		_ = e.memory.AddTurn(ctx, req.SessionID, req.Query, "user", e.clock())
		_ = e.memory.AddTurn(ctx, req.SessionID, reply.Text, "assistant", e.clock())
		if len(observations) > 0 {
			for _, obs := range observations {
				if obs.Result != nil {
					_ = e.memory.StoreFact(ctx, req.SessionID, obs)
				}
			}
		}
	}

	return &Response{
		Text:         reply.Text,
		Provider:     reply.Provider,
		Model:        reply.Model,
		Observations: observations,
		Trace:        trace,
	}
}

func validateToolCall(call ToolCall, tools []ToolDef) string {
	if call.Name == "" {
		return "missing tool name"
	}
	var tool *ToolDef
	for i := range tools {
		if tools[i].Name == call.Name {
			tool = &tools[i]
			break
		}
	}
	if tool == nil {
		return fmt.Sprintf("unknown tool: %s", call.Name)
	}
	if call.Action == "" {
		return "missing tool action"
	}
	var action *ToolAction
	for i := range tool.Actions {
		if tool.Actions[i].Name == call.Action {
			action = &tool.Actions[i]
			break
		}
	}
	if action == nil {
		return fmt.Sprintf("unknown action: %s", call.Action)
	}
	if action.InputSchema == "" {
		return ""
	}
	return validateRequiredFields(action.InputSchema, call.Args)
}

func validateRequiredFields(schema string, args map[string]any) string {
	if args == nil {
		args = map[string]any{}
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(schema), &payload); err != nil {
		return ""
	}
	requiredRaw, ok := payload["required"]
	if !ok {
		return ""
	}
	reqSlice, ok := requiredRaw.([]any)
	if !ok || len(reqSlice) == 0 {
		return ""
	}
	missing := make([]string, 0)
	for _, item := range reqSlice {
		name, ok := item.(string)
		if !ok {
			continue
		}
		if _, exists := args[name]; !exists {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "missing required params: " + strings.Join(missing, ", ")
}
