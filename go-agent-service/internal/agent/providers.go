// Package agent provides a unified LLM provider interface
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

// Provider represents an LLM provider
type Provider string

const (
	ProviderGemini   Provider = "gemini"
	ProviderOpenAI   Provider = "openai"
	ProviderGroq     Provider = "groq"     // Free tier available
	ProviderTogether Provider = "together" // Free tier available
)

// ModelConfig represents a configured model
type ModelConfig struct {
	Provider    Provider `json:"provider"`
	Model       string   `json:"model"`
	DisplayName string   `json:"displayName"`
	Tier        string   `json:"tier"` // free, standard, premium
	MaxTokens   int      `json:"maxTokens"`
}

// AvailableModels returns all configured models
func AvailableModels() []ModelConfig {
	return []ModelConfig{
		// Gemini models
		{Provider: ProviderGemini, Model: "gemma-3-27b-it", DisplayName: "Gemma 3 27B", Tier: "free", MaxTokens: 8192},
		{Provider: ProviderGemini, Model: "gemma-3-12b", DisplayName: "Gemma 3 12B", Tier: "free", MaxTokens: 8192},
		{Provider: ProviderGemini, Model: "gemini-2.0-flash", DisplayName: "Gemini 2.0 Flash", Tier: "standard", MaxTokens: 8192},
		{Provider: ProviderGemini, Model: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash", Tier: "premium", MaxTokens: 32768},
		
		// OpenAI models
		{Provider: ProviderOpenAI, Model: "gpt-4o-mini", DisplayName: "GPT-4o Mini", Tier: "standard", MaxTokens: 4096},
		{Provider: ProviderOpenAI, Model: "gpt-4o", DisplayName: "GPT-4o", Tier: "premium", MaxTokens: 4096},
		{Provider: ProviderOpenAI, Model: "gpt-3.5-turbo", DisplayName: "GPT-3.5 Turbo", Tier: "standard", MaxTokens: 4096},
		
		// Groq (free tier)
		{Provider: ProviderGroq, Model: "llama-3.3-70b-versatile", DisplayName: "Llama 3.3 70B (Groq)", Tier: "free", MaxTokens: 8192},
		{Provider: ProviderGroq, Model: "mixtral-8x7b-32768", DisplayName: "Mixtral 8x7B (Groq)", Tier: "free", MaxTokens: 32768},
	}
}

// LLMClient is a unified interface for LLM providers
type LLMClient interface {
	Generate(ctx context.Context, prompt string, systemPrompt string) (string, error)
	Chat(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error)
	Provider() Provider
	Model() string
}

// ChatMessage represents a conversation message
type ChatMessage struct {
	Role    string `json:"role"` // user, assistant, system
	Content string `json:"content"`
}

// MultiProviderClient manages multiple LLM providers
type MultiProviderClient struct {
	geminiKey   string
	openaiKey   string
	groqKey     string
	togetherKey string
	httpClient  *http.Client
}

// NewMultiProviderClient creates a new multi-provider client
func NewMultiProviderClient(geminiKey, openaiKey string) *MultiProviderClient {
	return &MultiProviderClient{
		geminiKey:  geminiKey,
		openaiKey:  openaiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// WithGroq adds Groq API key
func (c *MultiProviderClient) WithGroq(key string) *MultiProviderClient {
	c.groqKey = key
	return c
}

// GetClient returns an LLM client for the specified provider and model
func (c *MultiProviderClient) GetClient(provider Provider, model string) (LLMClient, error) {
	switch provider {
	case ProviderGemini:
		if c.geminiKey == "" {
			return nil, fmt.Errorf("Gemini API key not configured")
		}
		return &geminiClient{
			apiKey: c.geminiKey,
			model:  model,
			client: c.httpClient,
		}, nil
		
	case ProviderOpenAI:
		if c.openaiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured")
		}
		return &openaiClient{
			apiKey: c.openaiKey,
			model:  model,
			client: c.httpClient,
		}, nil
		
	case ProviderGroq:
		if c.groqKey == "" {
			return nil, fmt.Errorf("Groq API key not configured")
		}
		return &groqClient{
			apiKey: c.groqKey,
			model:  model,
			client: c.httpClient,
		}, nil
		
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
}

// ========== Gemini Client ==========

type geminiClient struct {
	apiKey string
	model  string
	client *http.Client
}

func (c *geminiClient) Provider() Provider { return ProviderGemini }
func (c *geminiClient) Model() string      { return c.model }

func (c *geminiClient) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	
	reqBody := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}, "role": "user"},
		},
		"generationConfig": map[string]any{
			"temperature": 0.7,
			"maxOutputTokens": 2048,
		},
	}
	
	if systemPrompt != "" {
		reqBody["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		}
	}
	
	return c.doRequest(ctx, url, reqBody)
}

func (c *geminiClient) Chat(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)
	
	contents := make([]map[string]any, len(messages))
	for i, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents[i] = map[string]any{
			"parts": []map[string]string{{"text": msg.Content}},
			"role":  role,
		}
	}
	
	reqBody := map[string]any{
		"contents": contents,
		"generationConfig": map[string]any{
			"temperature": 0.7,
			"maxOutputTokens": 2048,
		},
	}
	
	if systemPrompt != "" {
		reqBody["systemInstruction"] = map[string]any{
			"parts": []map[string]string{{"text": systemPrompt}},
		}
	}
	
	return c.doRequest(ctx, url, reqBody)
}

func (c *geminiClient) doRequest(ctx context.Context, url string, reqBody map[string]any) (string, error) {
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Gemini error %d: %s", resp.StatusCode, string(respBody))
	}
	
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	
	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}
	
	var text string
	for _, part := range result.Candidates[0].Content.Parts {
		text += part.Text
	}
	return text, nil
}

// ========== OpenAI Client ==========

type openaiClient struct {
	apiKey string
	model  string
	client *http.Client
}

func (c *openaiClient) Provider() Provider { return ProviderOpenAI }
func (c *openaiClient) Model() string      { return c.model }

func (c *openaiClient) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []map[string]string{}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})
	
	return c.doRequest(ctx, messages)
}

func (c *openaiClient) Chat(ctx context.Context, msgs []ChatMessage, systemPrompt string) (string, error) {
	messages := []map[string]string{}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	for _, msg := range msgs {
		messages = append(messages, map[string]string{"role": msg.Role, "content": msg.Content})
	}
	
	return c.doRequest(ctx, messages)
}

func (c *openaiClient) doRequest(ctx context.Context, messages []map[string]string) (string, error) {
	reqBody := map[string]any{
		"model":    c.model,
		"messages": messages,
		"max_tokens": 2048,
		"temperature": 0.7,
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI error %d: %s", resp.StatusCode, string(respBody))
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}
	
	return result.Choices[0].Message.Content, nil
}

// ========== Groq Client (Free Tier) ==========

type groqClient struct {
	apiKey string
	model  string
	client *http.Client
}

func (c *groqClient) Provider() Provider { return ProviderGroq }
func (c *groqClient) Model() string      { return c.model }

func (c *groqClient) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	messages := []map[string]string{}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	messages = append(messages, map[string]string{"role": "user", "content": prompt})
	
	return c.doRequest(ctx, messages)
}

func (c *groqClient) Chat(ctx context.Context, msgs []ChatMessage, systemPrompt string) (string, error) {
	messages := []map[string]string{}
	if systemPrompt != "" {
		messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
	}
	for _, msg := range msgs {
		messages = append(messages, map[string]string{"role": msg.Role, "content": msg.Content})
	}
	
	return c.doRequest(ctx, messages)
}

func (c *groqClient) doRequest(ctx context.Context, messages []map[string]string) (string, error) {
	reqBody := map[string]any{
		"model":    c.model,
		"messages": messages,
		"max_tokens": 2048,
		"temperature": 0.7,
	}
	
	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	respBody, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Groq error %d: %s", resp.StatusCode, string(respBody))
	}
	
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}
	
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}
	
	return result.Choices[0].Message.Content, nil
}
