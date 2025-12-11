// Package tools provides UCL tool implementations
package tools

import (
	"context"
	"fmt"
	"time"
)

// Tool is the interface for UCL tools
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]any) (*Result, error)
}

// Result represents the result of a tool execution
type Result struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data"`
	Message string         `json:"message"`
}

// JiraTool implements Jira operations
type JiraTool struct{}

func NewJiraTool() *JiraTool {
	return &JiraTool{}
}

func (t *JiraTool) Name() string {
	return "jira"
}

func (t *JiraTool) Description() string {
	return "Search and manage Jira tickets. Can search by query, update status, assign tickets, and add comments."
}

func (t *JiraTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)
	
	switch action {
	case "search":
		query, _ := params["query"].(string)
		return &Result{
			Success: true,
			Data: map[string]any{
				"tickets": []map[string]any{
					{"id": "MOBILE-1234", "title": "Login 401 error on mobile", "status": "In Progress"},
					{"id": "API-567", "title": "Rate limiting not working", "status": "Open"},
				},
				"total": 2,
			},
			Message: fmt.Sprintf("Found 2 tickets matching: %s", query),
		}, nil
		
	case "update_status":
		ticketID, _ := params["ticket_id"].(string)
		status, _ := params["status"].(string)
		return &Result{
			Success: true,
			Data: map[string]any{
				"ticket_id": ticketID,
				"new_status": status,
				"updated_at": time.Now().Format(time.RFC3339),
			},
			Message: fmt.Sprintf("Updated %s status to %s", ticketID, status),
		}, nil
		
	default:
		return &Result{
			Success: false,
			Message: fmt.Sprintf("Unknown action: %s", action),
		}, nil
	}
}

// GitHubTool implements GitHub operations
type GitHubTool struct{}

func NewGitHubTool() *GitHubTool {
	return &GitHubTool{}
}

func (t *GitHubTool) Name() string {
	return "github"
}

func (t *GitHubTool) Description() string {
	return "Interact with GitHub. Can fetch PR details, file contents, commits, and repository information."
}

func (t *GitHubTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)
	
	switch action {
	case "get_pr":
		prNumber, _ := params["pr_number"].(float64)
		return &Result{
			Success: true,
			Data: map[string]any{
				"number":    int(prNumber),
				"title":     "Fix authentication token validation",
				"status":    "open",
				"author":    "developer",
				"additions": 45,
				"deletions": 12,
				"files": []string{"auth.ts", "login.ts"},
			},
			Message: fmt.Sprintf("Retrieved PR #%d", int(prNumber)),
		}, nil
		
	case "approve_pr":
		prNumber, _ := params["pr_number"].(float64)
		return &Result{
			Success: true,
			Data: map[string]any{
				"pr_number": int(prNumber),
				"action":    "approved",
			},
			Message: fmt.Sprintf("Approved PR #%d", int(prNumber)),
		}, nil
		
	default:
		return &Result{
			Success: false,
			Message: fmt.Sprintf("Unknown action: %s", action),
		}, nil
	}
}

// PagerDutyTool implements PagerDuty operations
type PagerDutyTool struct{}

func NewPagerDutyTool() *PagerDutyTool {
	return &PagerDutyTool{}
}

func (t *PagerDutyTool) Name() string {
	return "pagerduty"
}

func (t *PagerDutyTool) Description() string {
	return "Manage PagerDuty alerts. Can acknowledge, resolve, and escalate incidents."
}

func (t *PagerDutyTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)
	alertID, _ := params["alert_id"].(string)
	
	switch action {
	case "acknowledge":
		return &Result{
			Success: true,
			Data: map[string]any{
				"alert_id": alertID,
				"status":   "acknowledged",
			},
			Message: fmt.Sprintf("Acknowledged alert %s", alertID),
		}, nil
		
	case "resolve":
		return &Result{
			Success: true,
			Data: map[string]any{
				"alert_id": alertID,
				"status":   "resolved",
			},
			Message: fmt.Sprintf("Resolved alert %s", alertID),
		}, nil
		
	default:
		return &Result{
			Success: false,
			Message: fmt.Sprintf("Unknown action: %s", action),
		}, nil
	}
}

// SlackTool implements Slack operations
type SlackTool struct{}

func NewSlackTool() *SlackTool {
	return &SlackTool{}
}

func (t *SlackTool) Name() string {
	return "slack"
}

func (t *SlackTool) Description() string {
	return "Send Slack messages. Can post to channels, send DMs, and create threads."
}

func (t *SlackTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	channel, _ := params["channel"].(string)
	message, _ := params["message"].(string)
	
	return &Result{
		Success: true,
		Data: map[string]any{
			"channel":    channel,
			"message_ts": time.Now().UnixNano(),
		},
		Message: fmt.Sprintf("Posted message to %s: %s", channel, truncate(message, 50)),
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
