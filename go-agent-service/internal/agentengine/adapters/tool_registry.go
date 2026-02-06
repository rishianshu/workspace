package adapters

import (
	"context"

	"github.com/antigravity/go-agent-service/internal/agentengine"
	"github.com/antigravity/go-agent-service/internal/tools"
)

// ToolListProvider describes the minimal interface needed from the tool registry.
type ToolListProvider interface {
	ListToolsFor(ctx context.Context, userID, projectID string) []tools.ToolDefinition
}

// RegistryToolSource adapts tools.Registry to AgentEngine ToolRegistry.
type RegistryToolSource struct {
	registry ToolListProvider
}

// NewRegistryToolSource creates a new tool registry adapter.
func NewRegistryToolSource(registry ToolListProvider) *RegistryToolSource {
	return &RegistryToolSource{registry: registry}
}

// ListTools implements agentengine.ToolRegistry.
func (r *RegistryToolSource) ListTools(ctx context.Context, userID, projectID string) ([]agentengine.ToolDef, error) {
	toolDefs := r.registry.ListToolsFor(ctx, userID, projectID)
	out := make([]agentengine.ToolDef, 0, len(toolDefs))

	for _, t := range toolDefs {
		actions := make([]agentengine.ToolAction, 0, len(t.Actions))
		for _, a := range t.Actions {
			actions = append(actions, agentengine.ToolAction{
				Name:        a.Name,
				Description: a.Description,
				InputSchema: a.InputSchema,
			})
		}
		out = append(out, agentengine.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			Actions:     actions,
		})
	}

	return out, nil
}
