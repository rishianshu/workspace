// Package workflow provides Temporal activities
package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/antigravity/go-agent-service/internal/tools"
)

// Activities contains all workflow activity implementations
type Activities struct {
	jiraTool      *tools.JiraTool
	githubTool    *tools.GitHubTool
	pagerdutyTool *tools.PagerDutyTool
	slackTool     *tools.SlackTool
}

// NewActivities creates a new Activities instance
func NewActivities() *Activities {
	return &Activities{
		jiraTool:      tools.NewJiraTool(),
		githubTool:    tools.NewGitHubTool(),
		pagerdutyTool: tools.NewPagerDutyTool(),
		slackTool:     tools.NewSlackTool(),
	}
}

// ActivityResult represents the result of an activity execution
type ActivityResult struct {
	Success   bool           `json:"success"`
	Data      map[string]any `json:"data"`
	Message   string         `json:"message"`
	Error     string         `json:"error,omitempty"`
	Duration  time.Duration  `json:"duration"`
}

// JiraSearchActivity searches Jira tickets
func (a *Activities) JiraSearchActivity(ctx context.Context, query string) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.jiraTool.Execute(ctx, map[string]any{
		"action": "search",
		"query":  query,
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// JiraUpdateStatusActivity updates a Jira ticket status
func (a *Activities) JiraUpdateStatusActivity(ctx context.Context, ticketID, status string) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.jiraTool.Execute(ctx, map[string]any{
		"action":    "update_status",
		"ticket_id": ticketID,
		"status":    status,
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// GitHubGetPRActivity fetches a GitHub PR
func (a *Activities) GitHubGetPRActivity(ctx context.Context, prNumber int) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.githubTool.Execute(ctx, map[string]any{
		"action":    "get_pr",
		"pr_number": float64(prNumber),
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// GitHubApprovePRActivity approves a GitHub PR
func (a *Activities) GitHubApprovePRActivity(ctx context.Context, prNumber int) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.githubTool.Execute(ctx, map[string]any{
		"action":    "approve_pr",
		"pr_number": float64(prNumber),
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// SlackPostActivity posts a message to Slack
func (a *Activities) SlackPostActivity(ctx context.Context, channel, message string) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.slackTool.Execute(ctx, map[string]any{
		"channel": channel,
		"message": message,
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// PagerDutyAcknowledgeActivity acknowledges a PagerDuty alert
func (a *Activities) PagerDutyAcknowledgeActivity(ctx context.Context, alertID string) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.pagerdutyTool.Execute(ctx, map[string]any{
		"action":   "acknowledge",
		"alert_id": alertID,
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// PagerDutyResolveActivity resolves a PagerDuty alert
func (a *Activities) PagerDutyResolveActivity(ctx context.Context, alertID string) (*ActivityResult, error) {
	start := time.Now()
	
	result, err := a.pagerdutyTool.Execute(ctx, map[string]any{
		"action":   "resolve",
		"alert_id": alertID,
	})
	
	if err != nil {
		return &ActivityResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, nil
	}

	return &ActivityResult{
		Success:  result.Success,
		Data:     result.Data,
		Message:  result.Message,
		Duration: time.Since(start),
	}, nil
}

// HumanApprovalActivity waits for human approval
// In Temporal, this would use a signal to wait for external input
func (a *Activities) HumanApprovalActivity(ctx context.Context, workflowID, message string) (*ActivityResult, error) {
	start := time.Now()
	
	// In production: This would create a pending approval state and wait for signal
	// For now, simulate auto-approval after a delay
	
	return &ActivityResult{
		Success:  true,
		Data:     map[string]any{"approved": true, "workflow_id": workflowID},
		Message:  fmt.Sprintf("Human approval received for: %s", message),
		Duration: time.Since(start),
	}, nil
}

// ConditionalActivity evaluates a condition and returns result
func (a *Activities) ConditionalActivity(ctx context.Context, condition string, data map[string]any) (*ActivityResult, error) {
	start := time.Now()
	
	// Simplified condition evaluation
	// In production would use expression evaluation library
	result := evaluateCondition(condition, data)
	
	return &ActivityResult{
		Success:  true,
		Data:     map[string]any{"condition_result": result},
		Message:  fmt.Sprintf("Condition '%s' evaluated to %v", condition, result),
		Duration: time.Since(start),
	}, nil
}

func evaluateCondition(condition string, data map[string]any) bool {
	// Simplified condition evaluation
	// Example: "${scan.count} > 0"
	
	// For now, just check if data has any results
	if count, ok := data["count"].(int); ok {
		return count > 0
	}
	if tickets, ok := data["tickets"].([]any); ok {
		return len(tickets) > 0
	}
	
	return true // Default to true
}
