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
	ProviderLocal    Provider = "local"    // Stub for testing
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
		// Local stub (for testing without API calls)
		{Provider: ProviderLocal, Model: "stub", DisplayName: "Local Stub (Testing)", Tier: "free", MaxTokens: 4096},
		
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
		
	case ProviderLocal:
		// Local stub - no API key needed
		return &localClient{}, nil
		
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

// ========== Local Stub Client (Testing) ==========

type localClient struct{}

func (c *localClient) Provider() Provider { return ProviderLocal }
func (c *localClient) Model() string      { return "stub" }

func (c *localClient) Generate(ctx context.Context, prompt string, systemPrompt string) (string, error) {
	return c.matchKeywords(prompt), nil
}

func (c *localClient) Chat(ctx context.Context, messages []ChatMessage, systemPrompt string) (string, error) {
	// Take the last user message
	var lastMsg string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			lastMsg = messages[i].Content
			break
		}
	}
	return c.matchKeywords(lastMsg), nil
}

func (c *localClient) matchKeywords(prompt string) string {
	lower := toLowerLocal(prompt)
	
	// Bug/Error keywords
	if containsLocal(lower, "bug") || containsLocal(lower, "error") || containsLocal(lower, "fix") || containsLocal(lower, "login") {
		return "[LOCAL STUB] 🐛 Detected BUG/FIX request. I found an issue in the authentication flow. The session token validation is failing due to an incorrect expiry check. Here's my proposed fix:\n\n```typescript\n// Fix: Correct token expiry validation\nfunction validateToken(token: string): boolean {\n  const decoded = jwt.decode(token);\n  const now = Math.floor(Date.now() / 1000);\n  return decoded.exp > now; // Fixed: was using < instead of >\n}\n```\n\n**Citations:** [MOBILE-1234], [auth.ts:45]"
	}
	
	// PR/Review keywords
	if containsLocal(lower, "pr") || containsLocal(lower, "review") || containsLocal(lower, "pull request") {
		return "[LOCAL STUB] 📝 Detected PR REVIEW request. I've reviewed the pull request and found 2 potential issues:\n\n1. **Missing null check** on line 23 - Add `if (!user) return null;`\n2. **Potential performance issue** with nested loops - Consider using `Map` instead\n\n**Overall:** Approved with minor changes requested.\n\n**Citations:** [PR-4423]"
	}
	
	// Documentation keywords
	if containsLocal(lower, "doc") || containsLocal(lower, "api") || containsLocal(lower, "spec") {
		return "[LOCAL STUB] 📚 Detected DOCUMENTATION request. Here's the generated API documentation:\n\n```yaml\nopenapi: 3.0.0\ninfo:\n  title: Auth API\n  version: 1.0.0\npaths:\n  /login:\n    post:\n      summary: User login\n```\n\n**Citations:** [auth.ts]"
	}
	
	// Workflow keywords
	if containsLocal(lower, "workflow") || containsLocal(lower, "automate") || containsLocal(lower, "schedule") {
		return "[LOCAL STUB] ⚙️ Detected WORKFLOW request. I've synthesized a workflow:\n\n```yaml\nname: Daily Bug Scanner\ntrigger:\n  schedule: \"0 9 * * *\"  # Every day at 9 AM\nsteps:\n  - id: scan\n    action: ucl.jira.search\n    params:\n      query: \"priority = Critical\"\n  - id: notify\n    action: ucl.slack.post\n    params:\n      channel: \"#dev-alerts\"\n```"
	}
	
	// Ticket/Jira keywords
	if containsLocal(lower, "ticket") || containsLocal(lower, "jira") || containsLocal(lower, "assignee") {
		return "[LOCAL STUB] 🎫 Detected TICKET request. Found 3 critical tickets:\n\n1. **MOBILE-1234** - Login button unresponsive (Critical)\n2. **MOBILE-1235** - Session timeout issues (High)\n3. **MOBILE-1236** - User avatar not loading (Medium)\n\nWould you like me to assign or update any of these?"
	}
	
	// Alert/PagerDuty keywords
	if containsLocal(lower, "alert") || containsLocal(lower, "pagerduty") || containsLocal(lower, "incident") {
		return "[LOCAL STUB] 🚨 Detected ALERT request. Current active incidents:\n\n1. **INC-001** - API latency spike (Acknowledged)\n2. **INC-002** - Database connection pool exhausted (Triggered)\n\nWould you like me to acknowledge or resolve any alerts?"
	}
	
	// GitHub keywords  
	if containsLocal(lower, "github") || containsLocal(lower, "merge") || containsLocal(lower, "branch") {
		return "[LOCAL STUB] 🔱 Detected GITHUB request. Repository status:\n\n- **main**: 234 commits, last updated 2h ago\n- **feature/auth-fix**: 3 commits ahead, ready for merge\n- **Open PRs**: 5 pending review\n\nWould you like me to merge or create a PR?"
	}
	
	// Hello/greeting
	if containsLocal(lower, "hello") || containsLocal(lower, "hi") || containsLocal(lower, "hey") {
		return "[LOCAL STUB] 👋 Hello! I'm the Antigravity Agent running in **local stub mode**. This mode provides deterministic responses for testing.\n\nTry asking about:\n- 🐛 Bug fixes\n- 📝 PR reviews\n- 📚 Documentation\n- ⚙️ Workflows\n- 🎫 Tickets\n- 🚨 Alerts"
	}
	
	// Default response
	return fmt.Sprintf("[LOCAL STUB] 🤖 Received query: \"%s\"\n\nI'm running in local stub mode for deterministic testing. Keywords I understand:\n- bug, error, fix, login\n- pr, review, pull request\n- doc, api, spec\n- workflow, automate, schedule\n- ticket, jira\n- alert, pagerduty\n- github, merge\n- hello, hi", prompt)
}

func toLowerLocal(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsLocal(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

