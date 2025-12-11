// Package agent provides the Gemini API client
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

// GeminiClient wraps the Gemini REST API
type GeminiClient struct {
	apiKey    string
	model     string
	baseURL   string
	client    *http.Client
}

// NewGeminiClient creates a new Gemini API client
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:  apiKey,
		model:   "gemma-3-27b-it", // Using Gemma for better free tier quota
		baseURL: "https://generativelanguage.googleapis.com/v1beta",
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// WithModel sets the model to use
func (c *GeminiClient) WithModel(model string) *GeminiClient {
	c.model = model
	return c
}

// GenerateContentRequest for Gemini API
type GenerateContentRequest struct {
	Contents         []Content         `json:"contents"`
	SystemInstruction *Content         `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

// Content represents a message content
type Content struct {
	Parts []Part  `json:"parts"`
	Role  string  `json:"role,omitempty"` // user, model
}

// Part represents a content part
type Part struct {
	Text string `json:"text,omitempty"`
}

// GenerationConfig for response tuning
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GenerateContentResponse from Gemini API
type GenerateContentResponse struct {
	Candidates    []Candidate    `json:"candidates"`
	UsageMetadata *UsageMetadata `json:"usageMetadata,omitempty"`
}

// Candidate represents a response candidate
type Candidate struct {
	Content       Content `json:"content"`
	FinishReason  string  `json:"finishReason"`
	Index         int     `json:"index"`
}

// UsageMetadata tracks token usage
type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GenerateContent calls the Gemini API
func (c *GeminiClient) GenerateContent(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)

	request := GenerateContentRequest{
		Contents: []Content{
			{
				Parts: []Part{{Text: prompt}},
				Role:  "user",
			},
		},
		GenerationConfig: &GenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 2048,
		},
	}

	// Add system instruction if provided
	if systemPrompt != "" {
		request.SystemInstruction = &Content{
			Parts: []Part{{Text: systemPrompt}},
		}
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
		// Check for rate limit
		if resp.StatusCode == 429 {
			return "", fmt.Errorf("rate limited (429): quota exceeded, retry later")
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var response GenerateContentResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	// Extract text from first candidate
	var result string
	for _, part := range response.Candidates[0].Content.Parts {
		result += part.Text
	}

	return result, nil
}

// ChatWithHistory maintains conversation history
func (c *GeminiClient) ChatWithHistory(ctx context.Context, history []Content, newMessage string, systemPrompt string) (string, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)

	// Build conversation with history
	contents := make([]Content, len(history)+1)
	copy(contents, history)
	contents[len(history)] = Content{
		Parts: []Part{{Text: newMessage}},
		Role:  "user",
	}

	request := GenerateContentRequest{
		Contents: contents,
		GenerationConfig: &GenerationConfig{
			Temperature:     0.7,
			MaxOutputTokens: 2048,
		},
	}

	if systemPrompt != "" {
		request.SystemInstruction = &Content{
			Parts: []Part{{Text: systemPrompt}},
		}
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
			return "", fmt.Errorf("rate limited")
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var response GenerateContentResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}

	var result string
	for _, part := range response.Candidates[0].Content.Parts {
		result += part.Text
	}

	return result, nil
}
