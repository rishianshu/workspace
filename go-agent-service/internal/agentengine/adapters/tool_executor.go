package adapters

import (
	"context"

	"github.com/antigravity/go-agent-service/internal/agentengine"
	"github.com/antigravity/go-agent-service/internal/tools"
)

// ToolExecutorBackend defines the execution surface needed from tools registry.
type ToolExecutorBackend interface {
	Execute(ctx context.Context, name, action string, params map[string]any) (*tools.Result, error)
}

// RegistryExecutor executes tool calls through the tools registry.
type RegistryExecutor struct {
	backend ToolExecutorBackend
}

// NewRegistryExecutor creates a tool executor adapter.
func NewRegistryExecutor(backend ToolExecutorBackend) *RegistryExecutor {
	return &RegistryExecutor{backend: backend}
}

// Execute implements agentengine.ToolExecutor.
func (e *RegistryExecutor) Execute(ctx context.Context, call agentengine.ToolCall) (*agentengine.ToolResult, error) {
	result, err := e.backend.Execute(ctx, call.Name, call.Action, call.Args)
	if err != nil {
		return nil, err
	}

	return &agentengine.ToolResult{
		Success: result.Success,
		Data:    result.Data,
		Message: result.Message,
	}, nil
}
