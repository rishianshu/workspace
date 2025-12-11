// Package tools provides the workflow synthesis tool
package tools

import (
	"context"
	"fmt"
	"strings"
)

// WorkflowTool implements workflow synthesis and execution
type WorkflowTool struct{}

func NewWorkflowTool() *WorkflowTool {
	return &WorkflowTool{}
}

func (t *WorkflowTool) Name() string {
	return "workflow"
}

func (t *WorkflowTool) Description() string {
	return "Create and execute automated workflows. Can synthesize YAML definitions from natural language, submit for approval, and execute workflows."
}

func (t *WorkflowTool) Execute(ctx context.Context, params map[string]any) (*Result, error) {
	action, _ := params["action"].(string)

	switch action {
	case "synthesize":
		intent, _ := params["intent"].(string)
		return t.synthesize(intent)
		
	case "get_yaml":
		intent, _ := params["intent"].(string)
		return t.getYAML(intent)
		
	case "execute":
		name, _ := params["name"].(string)
		return t.execute(name, params)
		
	case "approve":
		workflowID, _ := params["workflow_id"].(string)
		return t.approve(workflowID)
		
	case "deny":
		workflowID, _ := params["workflow_id"].(string)
		reason, _ := params["reason"].(string)
		return t.deny(workflowID, reason)

	default:
		return &Result{
			Success: false,
			Message: fmt.Sprintf("Unknown workflow action: %s", action),
		}, nil
	}
}

func (t *WorkflowTool) synthesize(intent string) (*Result, error) {
	// Generate workflow from natural language intent
	yaml := t.generateYAML(intent)
	
	return &Result{
		Success: true,
		Data: map[string]any{
			"workflow_id": "wf-" + sanitize(intent)[:20],
			"yaml":        yaml,
			"steps":       extractStepCount(intent),
		},
		Message: "Workflow synthesized successfully",
	}, nil
}

func (t *WorkflowTool) getYAML(intent string) (*Result, error) {
	yaml := t.generateYAML(intent)
	
	return &Result{
		Success: true,
		Data: map[string]any{
			"yaml": yaml,
		},
		Message: "Workflow YAML generated",
	}, nil
}

func (t *WorkflowTool) generateYAML(intent string) string {
	var sb strings.Builder
	
	// Header
	sb.WriteString("# Auto-generated workflow from: " + intent + "\n")
	sb.WriteString("name: ")
	sb.WriteString(extractWorkflowTitle(intent))
	sb.WriteString("\n\n")
	
	// Trigger
	sb.WriteString("trigger:\n")
	if strings.Contains(strings.ToLower(intent), "morning") || strings.Contains(strings.ToLower(intent), "9") {
		sb.WriteString("  schedule: \"0 9 * * *\"  # Daily at 9 AM\n")
	} else if strings.Contains(strings.ToLower(intent), "hour") {
		sb.WriteString("  schedule: \"0 * * * *\"  # Every hour\n")
	} else {
		sb.WriteString("  event: manual\n")
	}
	sb.WriteString("\n")
	
	// Steps
	sb.WriteString("steps:\n")
	
	// Add steps based on intent
	stepNum := 1
	
	if strings.Contains(strings.ToLower(intent), "bug") || strings.Contains(strings.ToLower(intent), "critical") || strings.Contains(strings.ToLower(intent), "jira") {
		sb.WriteString(fmt.Sprintf("  - id: step%d\n", stepNum))
		sb.WriteString("    action: ucl.jira.search\n")
		sb.WriteString("    params:\n")
		sb.WriteString("      query: \"priority = Critical AND status = Open\"\n")
		stepNum++
	}
	
	if strings.Contains(strings.ToLower(intent), "github") || strings.Contains(strings.ToLower(intent), "pr") {
		sb.WriteString(fmt.Sprintf("  - id: step%d\n", stepNum))
		sb.WriteString("    action: ucl.github.list_prs\n")
		sb.WriteString("    params:\n")
		sb.WriteString("      state: open\n")
		sb.WriteString("      labels: [\"needs-review\"]\n")
		stepNum++
	}
	
	if strings.Contains(strings.ToLower(intent), "slack") || strings.Contains(strings.ToLower(intent), "notify") || strings.Contains(strings.ToLower(intent), "alert") {
		sb.WriteString(fmt.Sprintf("  - id: step%d\n", stepNum))
		sb.WriteString("    action: ucl.slack.post\n")
		if stepNum > 1 {
			sb.WriteString(fmt.Sprintf("    depends_on: [step%d]\n", stepNum-1))
		}
		sb.WriteString("    params:\n")
		sb.WriteString("      channel: \"#dev-alerts\"\n")
		sb.WriteString("      body: |\n")
		sb.WriteString("        🚨 Workflow Alert\n")
		sb.WriteString("        {{ if step1.data.tickets }}\n")
		sb.WriteString("        Found {{ len step1.data.tickets }} critical tickets\n")
		sb.WriteString("        {{ end }}\n")
		stepNum++
	}
	
	// If no specific steps, add a default
	if stepNum == 1 {
		sb.WriteString("  - id: step1\n")
		sb.WriteString("    action: log.info\n")
		sb.WriteString("    params:\n")
		sb.WriteString("      message: \"Workflow executed\"\n")
	}
	
	return sb.String()
}

func (t *WorkflowTool) execute(name string, params map[string]any) (*Result, error) {
	return &Result{
		Success: true,
		Data: map[string]any{
			"execution_id": "exec-001",
			"workflow":     name,
			"status":       "running",
		},
		Message: fmt.Sprintf("Workflow '%s' started", name),
	}, nil
}

func (t *WorkflowTool) approve(workflowID string) (*Result, error) {
	return &Result{
		Success: true,
		Data: map[string]any{
			"workflow_id": workflowID,
			"status":      "approved",
		},
		Message: fmt.Sprintf("Workflow %s approved", workflowID),
	}, nil
}

func (t *WorkflowTool) deny(workflowID, reason string) (*Result, error) {
	return &Result{
		Success: true,
		Data: map[string]any{
			"workflow_id": workflowID,
			"status":      "denied",
			"reason":      reason,
		},
		Message: fmt.Sprintf("Workflow %s denied: %s", workflowID, reason),
	}, nil
}

func extractWorkflowTitle(intent string) string {
	lower := strings.ToLower(intent)
	if strings.Contains(lower, "bug") || strings.Contains(lower, "critical") {
		return "Daily Bug Scanner"
	}
	if strings.Contains(lower, "deploy") {
		return "Deployment Checker"
	}
	if strings.Contains(lower, "pr") || strings.Contains(lower, "review") {
		return "PR Review Notifier"
	}
	return "Custom Workflow"
}

func extractStepCount(intent string) int {
	count := 0
	lower := strings.ToLower(intent)
	if strings.Contains(lower, "jira") || strings.Contains(lower, "bug") {
		count++
	}
	if strings.Contains(lower, "github") || strings.Contains(lower, "pr") {
		count++
	}
	if strings.Contains(lower, "slack") || strings.Contains(lower, "notify") {
		count++
	}
	if count == 0 {
		count = 1
	}
	return count
}

func sanitize(s string) string {
	// Replace spaces with hyphens and remove special chars
	result := strings.ToLower(s)
	result = strings.ReplaceAll(result, " ", "-")
	// Keep only alphanumeric and hyphens
	var clean strings.Builder
	for _, c := range result {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			clean.WriteRune(c)
		}
	}
	return clean.String()
}
