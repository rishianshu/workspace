// Package agent provides LLM client implementations
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIClient wraps the OpenAI API
type OpenAIClient struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOpenAIClient creates a new OpenAI API client
func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:  apiKey,
		model:   "gpt-4o-mini",
		baseURL: "https://api.openai.com/v1",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// WithModel sets the model to use
func (c *OpenAIClient) WithModel(model string) *OpenAIClient {
	c.model = model
	return c
}

// OpenAIRequest for chat completions
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

// OpenAIMessage represents a chat message
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse from chat completions
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage,omitempty"`
}

// OpenAIChoice represents a response choice
type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// OpenAIUsage tracks token usage
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatWithHistory sends a message with conversation history
func (c *OpenAIClient) ChatWithHistory(ctx context.Context, history []OpenAIMessage, newMessage string, systemPrompt string) (string, error) {
	url := c.baseURL + "/chat/completions"

	// Build messages with system prompt and history
	messages := make([]OpenAIMessage, 0, len(history)+2)
	
	if systemPrompt != "" {
		messages = append(messages, OpenAIMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	
	// Add history
	messages = append(messages, history...)
	
	// Add new user message
	messages = append(messages, OpenAIMessage{
		Role:    "user",
		Content: newMessage,
	})

	request := OpenAIRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   2048,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("rate limited (429): OpenAI quota exceeded")
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var response OpenAIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}

// GenerateContent simple single-turn generation
func (c *OpenAIClient) GenerateContent(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	return c.ChatWithHistory(ctx, nil, prompt, systemPrompt)
}
