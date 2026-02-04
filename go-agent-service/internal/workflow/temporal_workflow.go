package workflow

import (
	"strings"
	"time"

	"go.temporal.io/sdk/workflow"
)

// DynamicWorkflow executes a workflow based on a definition
func DynamicWorkflow(ctx workflow.Context, def WorkflowDefinition) (map[string]any, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Dynamic Workflow", "name", def.Name)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5, // Default timeout
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	results := make(map[string]any)

	// Activities struct wrapper (to use string name for invocation)
	var activities *Activities

	for _, step := range def.Steps {
		logger.Info("Processing step", "step_id", step.ID, "action", step.Action)

		var output any
		var err error

		switch {
		// Handle UCL Actions (ucl.service.action)
		case isUCLAction(step.Action):
			endpoint, actionName := parseUCLAction(step.Action)
			var result map[string]any
			err = workflow.ExecuteActivity(ctx, activities.CallUCLActivity, endpoint, actionName, step.Params).Get(ctx, &result)
			output = result

		// Handle Agent Actions (agent.ask, agent.think)
		case isAgentAction(step.Action):
			prompt, _ := step.Params["prompt"].(string)
			contextData, _ := step.Params["context"].(map[string]any)
			var result string
			err = workflow.ExecuteActivity(ctx, activities.CallLLMActivity, prompt, contextData).Get(ctx, &result)
			output = result

		// Handle Approvals
		case step.Action == "approval":
			// 1. Send Request
			req := ApprovalRequest{
				WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
				Summary:    step.Params["summary"].(string),
				Action:     step.ID,
			}
			err = workflow.ExecuteActivity(ctx, activities.RequestApprovalActivity, req).Get(ctx, nil)
			if err != nil {
				return nil, err
			}

			// 2. Wait for Signal
			logger.Info("Waiting for approval signal...")
			var approved bool
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(workflow.GetSignalChannel(ctx, "approval_signal"), func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &approved)
			})
			selector.Select(ctx)

			if !approved {
				logger.Warn("Workflow rejected by user")
				return results, nil // Exit early
			}
			output = "approved"

		default:
			logger.Warn("Unknown action type", "action", step.Action)
		}

		if err != nil {
			logger.Error("Step failed", "step_id", step.ID, "error", err)
			return nil, err
		}

		results[step.ID] = output
	}

	logger.Info("Workflow completed successfully")
	return results, nil
}

func isUCLAction(action string) bool {
	return len(action) > 4 && action[:4] == "ucl."
}

func parseUCLAction(action string) (string, string) {
	// ucl.jira.search -> endpoint=jira, action=search
	// Simple parsing for demo
	parts := split3(action[4:], ".")
	return parts[0], parts[1]
}

func isAgentAction(action string) bool {
	return len(action) > 6 && action[:6] == "agent."
}

func split3(s, sep string) []string {
	return strings.Split(s, sep)
}
