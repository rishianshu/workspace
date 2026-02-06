// Package adapters provides bridge implementations for AgentEngine.
package adapters

import (
	"context"

	"github.com/antigravity/go-agent-service/internal/agent"
	"github.com/antigravity/go-agent-service/internal/agentengine"
)

// RouterLLMClient adapts LLMRouter to the AgentEngine interface.
type RouterLLMClient struct {
	router *agent.LLMRouter
}

// NewRouterLLMClient creates an adapter for LLMRouter.
func NewRouterLLMClient(router *agent.LLMRouter) *RouterLLMClient {
	return &RouterLLMClient{router: router}
}

// Respond implements agentengine.LLMClient.
func (c *RouterLLMClient) Respond(ctx context.Context, input agentengine.LLMRequest) (agentengine.LLMResponse, error) {
	history := make([]agent.HistoryMessage, 0, len(input.History))
	for _, h := range input.History {
		history = append(history, agent.HistoryMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}

	text, err := c.router.GenerateResponse(ctx, input.Provider, input.Model, input.Query, input.Prompt, history)
	if err != nil {
		return agentengine.LLMResponse{}, err
	}

	return agentengine.LLMResponse{
		Text:     text,
		Provider: input.Provider,
		Model:    input.Model,
	}, nil
}
