// Package mcp provides the MCP client for remote tool discovery/execution.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client is an HTTP client for the MCP service.
type Client struct {
	baseURL   string
	http      *http.Client
	logger    *zap.SugaredLogger
	authToken string
}

// ClientConfig holds configuration for the MCP client.
type ClientConfig struct {
	BaseURL   string
	AuthToken string
}

// NewClient creates a new MCP client.
func NewClient(baseURL string, logger *zap.SugaredLogger) *Client {
	return NewClientWithConfig(ClientConfig{BaseURL: baseURL}, logger)
}

// NewClientWithConfig creates a new MCP client with config.
func NewClientWithConfig(cfg ClientConfig, logger *zap.SugaredLogger) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:9100"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger:    logger,
		authToken: cfg.AuthToken,
	}
}

// ListTools returns available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context, userID, projectID string) ([]ToolDefinition, error) {
	endpoint := c.baseURL + "/v1/tools"
	if userID != "" || projectID != "" {
		query := make([]string, 0, 2)
		if userID != "" {
			query = append(query, "userId="+url.QueryEscape(userID))
		}
		if projectID != "" {
			query = append(query, "projectId="+url.QueryEscape(projectID))
		}
		endpoint = endpoint + "?" + strings.Join(query, "&")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		c.logger.Warnw("MCP ListTools failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp list tools failed: %s", resp.Status)
	}

	var tools []ToolDefinition
	if err := json.NewDecoder(resp.Body).Decode(&tools); err != nil {
		c.logger.Warnw("MCP ListTools decode failed", "error", err)
		return nil, err
	}
	return tools, nil
}

// ExecuteTool calls the MCP server to execute a tool action.
func (c *Client) ExecuteTool(ctx context.Context, call ToolCall) (*Result, error) {
	body, err := json.Marshal(call)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/tools/execute", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcp execute failed: %s", resp.Status)
	}

	var result Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}
