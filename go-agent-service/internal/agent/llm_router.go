// Package agent provides LLM routing
package agent

import (
	"context"
	"fmt"
)

// LLMRouter routes requests to the appropriate LLM provider
type LLMRouter struct {
	geminiClient *GeminiClient
	openaiClient *OpenAIClient
	geminiAPIKey string
	openaiAPIKey string
}

// NewLLMRouter creates a new LLM router
func NewLLMRouter(geminiAPIKey, openaiAPIKey string) *LLMRouter {
	router := &LLMRouter{
		geminiAPIKey: geminiAPIKey,
		openaiAPIKey: openaiAPIKey,
	}
	
	// Initialize clients if API keys are provided
	if geminiAPIKey != "" {
		router.geminiClient = NewGeminiClient(geminiAPIKey)
	}
	if openaiAPIKey != "" {
		router.openaiClient = NewOpenAIClient(openaiAPIKey)
	}
	
	return router
}

// HistoryMessage for conversation history
type HistoryMessage struct {
	Role    string
	Content string
}

// GenerateResponse routes to the appropriate provider
func (r *LLMRouter) GenerateResponse(ctx context.Context, provider, model, query, systemPrompt string, history []HistoryMessage) (string, error) {
	switch provider {
	case "openai":
		return r.generateOpenAI(ctx, model, query, systemPrompt, history)
	case "gemini":
		return r.generateGemini(ctx, model, query, systemPrompt, history)
	case "groq":
		// Groq uses OpenAI-compatible API
		return r.generateGroq(ctx, model, query, systemPrompt, history)
	default:
		// Default to Gemini if available
		if r.geminiClient != nil {
			return r.generateGemini(ctx, model, query, systemPrompt, history)
		}
		if r.openaiClient != nil {
			return r.generateOpenAI(ctx, model, query, systemPrompt, history)
		}
		return "", fmt.Errorf("no LLM provider configured")
	}
}

// generateGemini uses Gemini API
func (r *LLMRouter) generateGemini(ctx context.Context, model, query, systemPrompt string, history []HistoryMessage) (string, error) {
	if r.geminiClient == nil {
		return "", fmt.Errorf("Gemini API key not configured")
	}
	
	client := r.geminiClient
	if model != "" {
		client = NewGeminiClient(r.geminiAPIKey).WithModel(model)
	}
	
	// Convert history to Gemini format
	geminiHistory := make([]Content, 0, len(history))
	for _, h := range history {
		role := h.Role
		if role == "assistant" {
			role = "model"
		}
		geminiHistory = append(geminiHistory, Content{
			Parts: []Part{{Text: h.Content}},
			Role:  role,
		})
	}
	
	if len(history) > 0 {
		return client.ChatWithHistory(ctx, geminiHistory, query, systemPrompt)
	}
	return client.GenerateContent(ctx, query, systemPrompt)
}

// generateOpenAI uses OpenAI API
func (r *LLMRouter) generateOpenAI(ctx context.Context, model, query, systemPrompt string, history []HistoryMessage) (string, error) {
	if r.openaiClient == nil {
		return "", fmt.Errorf("OpenAI API key not configured")
	}
	
	client := r.openaiClient
	if model != "" {
		client = NewOpenAIClient(r.openaiAPIKey).WithModel(model)
	}
	
	// Convert history to OpenAI format
	openaiHistory := make([]OpenAIMessage, 0, len(history))
	for _, h := range history {
		openaiHistory = append(openaiHistory, OpenAIMessage{
			Role:    h.Role,
			Content: h.Content,
		})
	}
	
	return client.ChatWithHistory(ctx, openaiHistory, query, systemPrompt)
}

// generateGroq uses Groq API (OpenAI-compatible)
func (r *LLMRouter) generateGroq(ctx context.Context, model, query, systemPrompt string, history []HistoryMessage) (string, error) {
	// Groq uses OpenAI-compatible API format
	// For now, fall back to OpenAI if configured
	if r.openaiClient != nil {
		return r.generateOpenAI(ctx, model, query, systemPrompt, history)
	}
	return "", fmt.Errorf("Groq support requires OpenAI-compatible client")
}

// HasProvider checks if a provider is configured
func (r *LLMRouter) HasProvider(provider string) bool {
	switch provider {
	case "gemini":
		return r.geminiClient != nil
	case "openai":
		return r.openaiClient != nil
	default:
		return false
	}
}
