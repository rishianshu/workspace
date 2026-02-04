package workflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.uber.org/zap"
)

// Activities holds all workflow activities
type Activities struct {
	logger *zap.SugaredLogger
	// UCL Registry interface for executing actions
	uclExecutor UCLExecutor
}

// UCLExecutor interface for executing generic UCL tools
type UCLExecutor interface {
	Execute(ctx context.Context, endpointID, actionName string, params map[string]any) (map[string]any, error)
}

// NewActivities creates a new Activities instance
func NewActivities(executor UCLExecutor, logger *zap.SugaredLogger) *Activities {
	return &Activities{
		uclExecutor: executor,
		logger:      logger,
	}
}

// CallUCLActivity executes a UCL tool
func (a *Activities) CallUCLActivity(ctx context.Context, endpointID, actionName string, params map[string]any) (map[string]any, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing UCL Activity", "endpoint", endpointID, "action", actionName)

	result, err := a.uclExecutor.Execute(ctx, endpointID, actionName, params)
	if err != nil {
		return nil, fmt.Errorf("UCL execution failed: %w", err)
	}

	return result, nil
}

// CallLLMActivity uses the LLM to reason about data (Simulated for now)
func (a *Activities) CallLLMActivity(ctx context.Context, prompt string, contextData map[string]any) (string, error) {
	// In a real implementation, this would call the LLM Router
	// For now, we stub it to demonstrate "LLM as a Tool"
	return fmt.Sprintf("AI Analysis of input: Identified %d items. Recommendation: Proceed.", len(contextData)), nil
}

// ApprovalRequest represents a request for human approval
type ApprovalRequest struct {
	WorkflowID string
	Summary    string
	Action     string
}

// RequestApprovalActivity simulates modifying state to "Pending Approval"
func (a *Activities) RequestApprovalActivity(ctx context.Context, req ApprovalRequest) error {
	logger := activity.GetLogger(ctx)
	logger.Info("⚠️  Requesting Approval", "workflow", req.WorkflowID, "summary", req.Summary)
	
	// In a real system, this might send a Slack DM, Email, or update a DB table
	// The workflow will then block on "WaitForSignal"
	return nil
}
