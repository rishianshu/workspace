package adapters

import (
	"context"
	"strings"

	"github.com/antigravity/go-agent-service/internal/agentengine"
)

// HeuristicPlanner makes deterministic planning decisions.
type HeuristicPlanner struct{}

// NewHeuristicPlanner creates a planner with deterministic heuristics.
func NewHeuristicPlanner() *HeuristicPlanner {
	return &HeuristicPlanner{}
}

// Plan implements agentengine.Planner.
func (p *HeuristicPlanner) Plan(ctx context.Context, input agentengine.PlanInput) (agentengine.Plan, error) {
	_ = ctx

	// If we already have observations, respond directly.
	if len(input.Observations) > 0 {
		return agentengine.Plan{Type: agentengine.PlanDirect}, nil
	}

	query := strings.ToLower(input.Request.Query)

	// Explicit tool invocation: tool:<name>.<action>
	if idx := strings.Index(query, "tool:"); idx >= 0 {
		token := strings.TrimSpace(query[idx+len("tool:"):])
		fields := strings.Fields(token)
		if len(fields) > 0 {
			call := parseToolToken(fields[0], input)
			if call != nil {
				return agentengine.Plan{
					Type:      agentengine.PlanToolCalls,
					ToolCalls: []agentengine.ToolCall{*call},
				}, nil
			}
		}
		return agentengine.Plan{
			Type:          agentengine.PlanNeedClarification,
			Clarification: "Which tool and action should I use? Example: tool:app/jira/search",
		}, nil
	}

	// Keyword-based tool selection
	if len(input.Tools) > 0 {
		if call := pickToolForQuery(query, input); call != nil {
			return agentengine.Plan{
				Type:      agentengine.PlanToolCalls,
				ToolCalls: []agentengine.ToolCall{*call},
			}, nil
		}
	}

	return agentengine.Plan{Type: agentengine.PlanDirect}, nil
}

func parseToolToken(token string, input agentengine.PlanInput) *agentengine.ToolCall {
	name := token
	action := ""
	if strings.Contains(token, ".") {
		parts := strings.SplitN(token, ".", 2)
		name = parts[0]
		action = parts[1]
	}
	if strings.Contains(token, "/") && action == "" {
		// Allow app/{appId}/{action} pattern
		parts := strings.Split(token, "/")
		if len(parts) >= 3 {
			name = strings.Join(parts[:len(parts)-1], "/")
			action = parts[len(parts)-1]
		}
	}

	call := &agentengine.ToolCall{
		Name:   name,
		Action: action,
		Args:   defaultArgs(input.Request),
	}

	// If action missing, attempt to infer from tool definitions
	if call.Action == "" {
		call.Action = inferAction(name, input.Tools)
	}
	if call.Action == "" {
		return nil
	}
	return call
}

func pickToolForQuery(query string, input agentengine.PlanInput) *agentengine.ToolCall {
	// Common keyword matches
	keywords := []string{"jira", "ticket", "pr", "github", "pagerduty", "incident", "alert", "slack", "workflow"}
	for _, kw := range keywords {
		if strings.Contains(query, kw) {
			for _, tool := range input.Tools {
				if strings.Contains(strings.ToLower(tool.Name), kw) {
					action := inferAction(tool.Name, input.Tools)
					if action == "" {
						continue
					}
					return &agentengine.ToolCall{
						Name:   tool.Name,
						Action: action,
						Args:   defaultArgs(input.Request),
					}
				}
			}
		}
	}

	// Fallback: choose first tool with a search/list action
	for _, tool := range input.Tools {
		for _, action := range tool.Actions {
			if action.Name == "search" || action.Name == "list" || action.Name == "query" {
				return &agentengine.ToolCall{
					Name:   tool.Name,
					Action: action.Name,
					Args:   defaultArgs(input.Request),
				}
			}
		}
	}

	return nil
}

func inferAction(toolName string, tools []agentengine.ToolDef) string {
	for _, tool := range tools {
		if tool.Name != toolName {
			continue
		}
		if len(tool.Actions) == 0 {
			return ""
		}
		for _, action := range tool.Actions {
			if action.Name == "search" || action.Name == "list" || action.Name == "query" {
				return action.Name
			}
		}
		return tool.Actions[0].Name
	}
	return ""
}

func defaultArgs(req agentengine.Request) map[string]any {
	args := map[string]any{
		"query": req.Query,
	}
	if req.UserID != "" {
		args["userId"] = req.UserID
	}
	if req.ProjectID != "" {
		args["projectId"] = req.ProjectID
	}
	return args
}
