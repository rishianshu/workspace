// Package memory provides embedding generation using Gemini
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// GeminiEmbedder generates embeddings using Gemini API
type GeminiEmbedder struct {
	apiKey string
	model  string
	client *http.Client
}

// NewGeminiEmbedder creates a new Gemini embedding service
func NewGeminiEmbedder(apiKey string) *GeminiEmbedder {
	return &GeminiEmbedder{
		apiKey: apiKey,
		model:  "text-embedding-004", // 768 dimensions
		client: &http.Client{},
	}
}

type embeddingRequest struct {
	Model   string `json:"model"`
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
}

type embeddingResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Embed generates an embedding for the given text
func (e *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, nil
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s",
		e.model,
		e.apiKey,
	)

	reqBody := embeddingRequest{
		Model: fmt.Sprintf("models/%s", e.model),
	}
	reqBody.Content.Parts = []struct {
		Text string `json:"text"`
	}{{Text: text}}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embedResp embeddingResponse
	if err := json.Unmarshal(body, &embedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if embedResp.Error != nil {
		return nil, fmt.Errorf("embedding API error: %s", embedResp.Error.Message)
	}

	return embedResp.Embedding.Values, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *GeminiEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	
	for i, text := range texts {
		embedding, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		results[i] = embedding
	}
	
	return results, nil
}
