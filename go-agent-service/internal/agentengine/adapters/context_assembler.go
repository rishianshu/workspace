package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antigravity/go-agent-service/internal/agent"
	"github.com/antigravity/go-agent-service/internal/agentengine"
	agentctx "github.com/antigravity/go-agent-service/internal/context"
	"github.com/antigravity/go-agent-service/internal/memory"
	"go.uber.org/zap"
)

// DefaultContextAssembler builds prompts from memory, KG context, tools, and query.
type DefaultContextAssembler struct {
	orchestrator *agentctx.Orchestrator
	memoryStore  memory.MemoryStore
	logger       *zap.SugaredLogger
}

// NewDefaultContextAssembler creates a context assembler adapter.
func NewDefaultContextAssembler(orchestrator *agentctx.Orchestrator, store memory.MemoryStore, logger *zap.SugaredLogger) *DefaultContextAssembler {
	return &DefaultContextAssembler{
		orchestrator: orchestrator,
		memoryStore:  store,
		logger:       logger,
	}
}

// Build composes the system prompt and memory context.
func (a *DefaultContextAssembler) Build(ctx context.Context, req agentengine.Request, tools []agentengine.ToolDef) (string, error) {
	systemPrompt := agent.SystemPrompt

	// Inject KG context if available
	if a.orchestrator != nil {
		kgCtx, err := a.orchestrator.Process(ctx, req.Query, req.ContextEntities)
		if err != nil {
			if a.logger != nil {
				a.logger.Warnw("KG context processing failed", "error", err)
			}
		} else if kgCtx != nil {
			formatted := kgCtx.FormatForLLM()
			if formatted != "" {
				systemPrompt = systemPrompt + "\n\n" + formatted
			}
		}
	}

	toolDescriptions := formatToolsForPrompt(tools)

	// Use memory-aware builder when available
	if a.memoryStore != nil {
		cfg := memory.DefaultContextConfig()
		cfg.SystemPrompt = systemPrompt
		cfg.ToolDescriptions = toolDescriptions

		builder := agentctx.NewBuilder(a.memoryStore, cfg)
		return builder.Build(ctx, req.SessionID, req.Query)
	}

	// Fallback when no memory store is configured
	var sections []string
	if systemPrompt != "" {
		sections = append(sections, systemPrompt)
	}
	if toolDescriptions != "" {
		sections = append(sections, "## Available Tools\n"+toolDescriptions)
	}
	sections = append(sections, fmt.Sprintf("## Current Request\n%s", req.Query))
	return strings.Join(sections, "\n\n"), nil
}

// AppendObservations appends tool observations to the prompt.
func (a *DefaultContextAssembler) AppendObservations(prompt string, observations []agentengine.Observation) (string, error) {
	if len(observations) == 0 {
		return prompt, nil
	}

	var sb strings.Builder
	sb.WriteString(prompt)
	sb.WriteString("\n\n## Tool Observations\n")
	for _, obs := range observations {
		if obs.Error != "" {
			sb.WriteString(fmt.Sprintf("- %s: error=%s\n", obs.ToolName, obs.Error))
			continue
		}
		if obs.Result != nil {
			payload, _ := json.Marshal(obs.Result.Data)
			sb.WriteString(fmt.Sprintf("- %s: %s\n", obs.ToolName, string(payload)))
			if obs.Result.Message != "" {
				sb.WriteString(fmt.Sprintf("  message: %s\n", obs.Result.Message))
			}
		}
	}

	return sb.String(), nil
}

func formatToolsForPrompt(tools []agentengine.ToolDef) string {
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, t := range tools {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n", t.Name, t.Description))
		if len(t.Actions) > 0 {
			sb.WriteString("Actions:\n")
			for _, action := range t.Actions {
				sb.WriteString(fmt.Sprintf("- %s: %s", action.Name, action.Description))
				if action.InputSchema != "" {
					sb.WriteString(fmt.Sprintf(" Schema: %s", action.InputSchema))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}
	return strings.TrimSpace(sb.String())
}
