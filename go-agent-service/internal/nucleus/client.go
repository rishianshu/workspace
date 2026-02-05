// Package nucleus provides the Nucleus GraphQL client
package nucleus

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Client is the Nucleus GraphQL client
type Client struct {
	url                  string
	username             string
	password             string
	tenantID             string
	bearerToken          string
	keycloakURL          string
	keycloakRealm        string
	keycloakClientID     string
	keycloakClientSecret string
	keycloakUsername     string
	keycloakPassword     string
	tokenExpiry          time.Time
	tokenMu              sync.Mutex
	httpClient           *http.Client
	logger               *zap.SugaredLogger
}

// ClientConfig holds configuration for the Nucleus client
type ClientConfig struct {
	APIURL               string
	Username             string
	Password             string
	TenantID             string
	BearerToken          string
	KeycloakURL          string
	KeycloakRealm        string
	KeycloakClientID     string
	KeycloakClientSecret string
	KeycloakUsername     string
	KeycloakPassword     string
}

// NewClient creates a new Nucleus client
func NewClient(url string, logger *zap.SugaredLogger) *Client {
	return &Client{
		url:      url,
		tenantID: "default",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// NewClientWithConfig creates a new Nucleus client with full config
func NewClientWithConfig(cfg ClientConfig, logger *zap.SugaredLogger) *Client {
	return &Client{
		url:                  cfg.APIURL,
		username:             cfg.Username,
		password:             cfg.Password,
		tenantID:             cfg.TenantID,
		bearerToken:          cfg.BearerToken,
		keycloakURL:          cfg.KeycloakURL,
		keycloakRealm:        cfg.KeycloakRealm,
		keycloakClientID:     cfg.KeycloakClientID,
		keycloakClientSecret: cfg.KeycloakClientSecret,
		keycloakUsername:     cfg.KeycloakUsername,
		keycloakPassword:     cfg.KeycloakPassword,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Node represents a knowledge graph node
type Node struct {
	ID          string         `json:"id"`
	DisplayName string         `json:"displayName"`
	EntityType  string         `json:"entityType"`
	Properties  map[string]any `json:"properties"`
}

// Edge represents a knowledge graph edge
type Edge struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Relationship string `json:"relationship"`
}

// Project represents a Nucleus project
type Project struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// MetadataEndpoint represents a replicated endpoint from Nucleus
type MetadataEndpoint struct {
	ID                 string         `json:"id"`
	SourceID           string         `json:"sourceId"`
	ProjectID          string         `json:"projectId"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	Verb               string         `json:"verb,omitempty"`
	URL                string         `json:"url,omitempty"`
	AuthPolicy         string         `json:"authPolicy,omitempty"`
	Domain             string         `json:"domain,omitempty"`
	Labels             []string       `json:"labels,omitempty"`
	Config             map[string]any `json:"config,omitempty"`
	DetectedVersion    string         `json:"detectedVersion,omitempty"`
	VersionHint        string         `json:"versionHint,omitempty"`
	Capabilities       []string       `json:"capabilities,omitempty"`
	DelegatedConnected bool           `json:"delegatedConnected,omitempty"`
	TemplateID         string         `json:"templateId,omitempty"` // Derived from config
}

// BrainSearchResult holds brain search response
type BrainSearchResult struct {
	Hits       []BrainSearchHit `json:"hits"`
	Episodes   []BrainEpisode   `json:"episodes"`
	GraphNodes []BrainGraphNode `json:"graphNodes"`
	Passages   []BrainPassage   `json:"passages"`
	PromptPack BrainPromptPack  `json:"promptPack"`
}

type BrainSearchHit struct {
	NodeID    string  `json:"nodeId"`
	NodeType  string  `json:"nodeType"`
	ProfileID string  `json:"profileId"`
	Score     float64 `json:"score"`
	Title     string  `json:"title"`
	URL       string  `json:"url"`
}

type BrainEpisode struct {
	ClusterNodeID string   `json:"clusterNodeId"`
	ClusterKind   string   `json:"clusterKind"`
	Score         float64  `json:"score"`
	MemberNodeIDs []string `json:"memberNodeIds"`
}

type BrainGraphNode struct {
	NodeID     string         `json:"nodeId"`
	NodeType   string         `json:"nodeType"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

type BrainPassage struct {
	SourceNodeID string `json:"sourceNodeId"`
	SourceKind   string `json:"sourceKind"`
	Text         string `json:"text"`
	URL          string `json:"url"`
}

type BrainPromptPack struct {
	ContextMarkdown string                   `json:"contextMarkdown"`
	Citations       []map[string]interface{} `json:"citations"`
}

// GraphQL request/response types
type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors,omitempty"`
}

type graphqlError struct {
	Message string `json:"message"`
}

func extractTemplateID(config map[string]any) string {
	if config == nil {
		return ""
	}
	if v, ok := config["templateId"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := config["template_id"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// execute sends a GraphQL query to Nucleus
func (c *Client) execute(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use direct GraphQL URL - ensure /graphql path
	url := c.url
	if !strings.HasSuffix(url, "/graphql") && !strings.HasSuffix(url, "/graphql/") {
		url = strings.TrimSuffix(url, "/") + "/graphql"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Add bearer token if configured or can be fetched
	if token := c.getBearerToken(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else if c.username != "" && c.password != "" {
		// Fallback to basic auth if configured
		auth := base64.StdEncoding.EncodeToString([]byte(c.username + ":" + c.password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warnw("Nucleus request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", gqlResp.Errors[0].Message)
	}

	return gqlResp.Data, nil
}

type keycloakTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (c *Client) getBearerToken(ctx context.Context) string {
	if c.bearerToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.bearerToken
	}
	if c.bearerToken != "" && c.tokenExpiry.IsZero() {
		return c.bearerToken
	}

	if c.keycloakURL == "" || c.keycloakClientID == "" || c.keycloakUsername == "" || c.keycloakPassword == "" {
		return ""
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.bearerToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.bearerToken
	}

	token, expires, err := c.fetchKeycloakToken(ctx)
	if err != nil {
		c.logger.Warnw("Failed to fetch Keycloak token", "error", err)
		return ""
	}
	c.bearerToken = token
	expirySeconds := expires
	if expires > 60 {
		expirySeconds = expires - 60
	}
	if expirySeconds <= 0 {
		expirySeconds = 1
	}
	c.tokenExpiry = time.Now().Add(time.Duration(expirySeconds) * time.Second)
	return c.bearerToken
}

func (c *Client) fetchKeycloakToken(ctx context.Context) (string, int64, error) {
	tokenURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", strings.TrimSuffix(c.keycloakURL, "/"), c.keycloakRealm)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", c.keycloakClientID)
	form.Set("username", c.keycloakUsername)
	form.Set("password", c.keycloakPassword)
	if c.keycloakClientSecret != "" {
		form.Set("client_secret", c.keycloakClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, fmt.Errorf("keycloak token error: %s", string(body))
	}

	var tokenResp keycloakTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", 0, err
	}
	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}

// ========================
// Project Operations
// ========================

// ListProjects returns all projects from Nucleus
func (c *Client) ListProjects(ctx context.Context) ([]Project, error) {
	c.logger.Debug("Fetching projects from Nucleus")

	query := `
		query ListProjects {
			metadataProjects {
				id
				slug
				displayName
				description
			}
		}
	`

	data, err := c.execute(ctx, query, nil)
	if err != nil {
		c.logger.Warnw("Failed to fetch projects", "error", err)
		return nil, err
	}

	var result struct {
		MetadataProjects []Project `json:"metadataProjects"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse projects: %w", err)
	}

	return result.MetadataProjects, nil
}

// GetProject returns a single project by ID
func (c *Client) GetProject(ctx context.Context, id string) (*Project, error) {
	query := `
		query GetProject($id: ID!) {
			metadataProject(id: $id) {
				id
				slug
				displayName
				description
			}
		}
	`

	data, err := c.execute(ctx, query, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}

	var result struct {
		MetadataProject *Project `json:"metadataProject"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse project: %w", err)
	}

	return result.MetadataProject, nil
}

// ========================
// Brain Search
// ========================

// BrainSearch performs semantic search with RAG context using Nucleus search API
func (c *Client) BrainSearch(ctx context.Context, queryText string, projectID string, options *BrainSearchOptions) (*BrainSearchResult, error) {
	c.logger.Infow("Brain search", "query", queryText, "project", projectID)

	// Use Nucleus search query with filters for project support
	query := `
		query Search($query: String!, $filters: SearchFilters, $limit: Int) {
			search(query: $query, filters: $filters, limit: $limit) {
				node {
					id
					entityType
					displayName
					sourceSystem
					canonicalPath
					properties
				}
				score
				highlights
			}
		}
	`

	limit := 10
	if options != nil && options.TopK > 0 {
		limit = options.TopK
	}

	variables := map[string]any{
		"query": queryText,
		"limit": limit,
	}

	// Add project filter if provided
	if projectID != "" {
		variables["filters"] = map[string]any{
			"projects": []string{projectID},
		}
	}

	data, err := c.execute(ctx, query, variables)
	if err != nil {
		c.logger.Errorw("Brain search failed", "error", err)
		return nil, fmt.Errorf("brain search failed: %w", err)
	}

	// Parse the Nucleus search response
	var searchResult struct {
		Search []struct {
			Node struct {
				ID            string         `json:"id"`
				EntityType    string         `json:"entityType"`
				DisplayName   string         `json:"displayName"`
				SourceSystem  string         `json:"sourceSystem"`
				CanonicalPath string         `json:"canonicalPath"`
				Properties    map[string]any `json:"properties"`
			} `json:"node"`
			Score      float64  `json:"score"`
			Highlights []string `json:"highlights"`
		} `json:"search"`
	}
	if err := json.Unmarshal(data, &searchResult); err != nil {
		c.logger.Errorw("Failed to parse search result", "error", err, "data", string(data))
		return nil, fmt.Errorf("failed to parse search result: %w", err)
	}

	c.logger.Infow("Brain search results", "count", len(searchResult.Search))

	// Convert to BrainSearchResult format
	hits := make([]BrainSearchHit, 0, len(searchResult.Search))
	for _, s := range searchResult.Search {
		hits = append(hits, BrainSearchHit{
			NodeID:   s.Node.ID,
			NodeType: s.Node.EntityType,
			Score:    s.Score,
			Title:    s.Node.DisplayName,
			URL:      s.Node.CanonicalPath,
		})
	}

	// Build context markdown from hits
	context := fmt.Sprintf("# Search Results\nQuery: %s\n\n", queryText)
	for i, hit := range hits {
		context += fmt.Sprintf("%d. **%s** (%s) - score: %.2f\n", i+1, hit.Title, hit.NodeType, hit.Score)
	}

	return &BrainSearchResult{
		Hits: hits,
		PromptPack: BrainPromptPack{
			ContextMarkdown: context,
		},
	}, nil
}

// BrainSearchOptions configures brain search behavior
type BrainSearchOptions struct {
	TopK            int  `json:"topK,omitempty"`
	MaxEpisodes     int  `json:"maxEpisodes,omitempty"`
	ExpandDepth     int  `json:"expandDepth,omitempty"`
	IncludeEpisodes bool `json:"includeEpisodes,omitempty"`
}

// ========================
// Endpoint Operations
// ========================

// ListEndpoints returns endpoints from Nucleus for a project
func (c *Client) ListEndpoints(ctx context.Context, projectID string) ([]MetadataEndpoint, error) {
	c.logger.Debugw("Fetching endpoints", "project", projectID)

	query := `
		query ListEndpoints($projectId: ID) {
			metadataEndpoints(projectId: $projectId, includeDeleted: false) {
				id
				sourceId
				projectId
				name
				description
				verb
				url
				authPolicy
				capabilities
				domain
				labels
				config
				detectedVersion
				versionHint
				delegatedConnected
			}
		}
	`

	variables := map[string]any{}
	if projectID != "" {
		variables["projectId"] = projectID
	}

	data, err := c.execute(ctx, query, variables)
	if err != nil {
		c.logger.Warnw("Failed to fetch endpoints", "error", err)
		return nil, err
	}

	var result struct {
		MetadataEndpoints []MetadataEndpoint `json:"metadataEndpoints"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse endpoints: %w", err)
	}

	for _, ep := range result.MetadataEndpoints {
		ep.TemplateID = extractTemplateID(ep.Config)
	}

	return result.MetadataEndpoints, nil
}

// GetEndpoint returns a single endpoint by ID from Nucleus.
func (c *Client) GetEndpoint(ctx context.Context, endpointID string) (*MetadataEndpoint, error) {
	if endpointID == "" {
		return nil, fmt.Errorf("endpoint id is required")
	}

	query := `
		query GetEndpoint($id: ID!) {
			metadataEndpoint(id: $id) {
				id
				sourceId
				projectId
				name
				description
				verb
				url
				authPolicy
				capabilities
				domain
				labels
				config
				detectedVersion
				versionHint
				delegatedConnected
			}
		}
	`

	variables := map[string]any{"id": endpointID}
	data, err := c.execute(ctx, query, variables)
	if err != nil {
		return nil, err
	}

	var result struct {
		MetadataEndpoint *MetadataEndpoint `json:"metadataEndpoint"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse endpoint: %w", err)
	}
	if result.MetadataEndpoint == nil {
		return nil, fmt.Errorf("endpoint not found")
	}
	result.MetadataEndpoint.TemplateID = extractTemplateID(result.MetadataEndpoint.Config)
	return result.MetadataEndpoint, nil
}

// ========================
// Legacy Methods (preserved)
// ========================

// QueryNodes retrieves nodes by IDs
func (c *Client) QueryNodes(ctx context.Context, ids []string) ([]Node, error) {
	c.logger.Debugw("Querying Nucleus for nodes", "ids", ids)

	if len(ids) == 0 {
		return []Node{}, nil
	}

	query := `
		query GetNodes($ids: [String!]!) {
			nodes(ids: $ids) {
				id
				displayName
				entityType
				properties
			}
		}
	`

	data, err := c.execute(ctx, query, map[string]any{"ids": ids})
	if err != nil {
		c.logger.Warnw("Using stub data for nodes", "error", err)
		return c.stubNodes(ids), nil
	}

	var result struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse nodes: %w", err)
	}

	return result.Nodes, nil
}

// SearchNodes searches for nodes matching a query
func (c *Client) SearchNodes(ctx context.Context, queryText string, limit int) ([]Node, error) {
	c.logger.Debugw("Searching Nucleus", "query", queryText, "limit", limit)

	query := `
		query SearchNodes($query: String!, $limit: Int) {
			searchNodes(query: $query, limit: $limit) {
				id
				displayName
				entityType
				properties
			}
		}
	`

	data, err := c.execute(ctx, query, map[string]any{
		"query": queryText,
		"limit": limit,
	})
	if err != nil {
		c.logger.Warnw("Using stub data for search", "error", err)
		return c.stubSearch(queryText), nil
	}

	var result struct {
		SearchNodes []Node `json:"searchNodes"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return result.SearchNodes, nil
}

// GetRelatedNodes gets nodes related to a given node
func (c *Client) GetRelatedNodes(ctx context.Context, nodeID string) ([]Node, []Edge, error) {
	c.logger.Debugw("Getting related nodes", "node_id", nodeID)

	query := `
		query GetRelated($nodeId: String!) {
			node(id: $nodeId) {
				id
				displayName
				entityType
				related {
					node {
						id
						displayName
						entityType
						properties
					}
					edge {
						relationship
					}
				}
			}
		}
	`

	data, err := c.execute(ctx, query, map[string]any{"nodeId": nodeID})
	if err != nil {
		c.logger.Warnw("Using stub data for related nodes", "error", err)
		return c.stubRelated(nodeID)
	}

	var result struct {
		Node struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			EntityType  string `json:"entityType"`
			Related     []struct {
				Node struct {
					ID          string         `json:"id"`
					DisplayName string         `json:"displayName"`
					EntityType  string         `json:"entityType"`
					Properties  map[string]any `json:"properties"`
				} `json:"node"`
				Edge struct {
					Relationship string `json:"relationship"`
				} `json:"edge"`
			} `json:"related"`
		} `json:"node"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse related nodes: %w", err)
	}

	nodes := make([]Node, 0, len(result.Node.Related))
	edges := make([]Edge, 0, len(result.Node.Related))
	for _, rel := range result.Node.Related {
		nodes = append(nodes, Node{
			ID:          rel.Node.ID,
			DisplayName: rel.Node.DisplayName,
			EntityType:  rel.Node.EntityType,
			Properties:  rel.Node.Properties,
		})
		edges = append(edges, Edge{
			From:         nodeID,
			To:           rel.Node.ID,
			Relationship: rel.Edge.Relationship,
		})
	}

	return nodes, edges, nil
}

// ========================
// Stub data fallbacks
// ========================

func (c *Client) stubNodes(ids []string) []Node {
	nodes := make([]Node, 0, len(ids))
	for _, id := range ids {
		nodes = append(nodes, Node{
			ID:          id,
			DisplayName: "Node " + id,
			EntityType:  "entity",
			Properties:  map[string]any{"source": "stub"},
		})
	}
	return nodes
}

func (c *Client) stubSearch(query string) []Node {
	return []Node{
		{
			ID:          "MOBILE-1234",
			DisplayName: "Mobile Login Bug Investigation",
			EntityType:  "ticket",
			Properties: map[string]any{
				"status":   "In Progress",
				"priority": "High",
				"assignee": "developer@example.com",
			},
		},
		{
			ID:          "PR-4423",
			DisplayName: "Fix authentication token validation",
			EntityType:  "pr",
			Properties: map[string]any{
				"status":    "open",
				"author":    "developer",
				"additions": 45,
				"deletions": 12,
			},
		},
	}
}

func (c *Client) stubRelated(nodeID string) ([]Node, []Edge, error) {
	nodes := []Node{
		{ID: "auth.ts", DisplayName: "auth.ts", EntityType: "file"},
		{ID: "login.ts", DisplayName: "login.ts", EntityType: "file"},
	}
	edges := []Edge{
		{From: nodeID, To: "auth.ts", Relationship: "AFFECTS"},
		{From: nodeID, To: "login.ts", Relationship: "AFFECTS"},
	}
	return nodes, edges, nil
}

func (c *Client) stubBrainSearch(query string) *BrainSearchResult {
	return &BrainSearchResult{
		Hits: []BrainSearchHit{
			{NodeID: "MOBILE-1234", NodeType: "ticket", Score: 0.85, Title: "Mobile Login Bug"},
			{NodeID: "PR-4423", NodeType: "pr", Score: 0.72, Title: "Fix auth token validation"},
		},
		Episodes: []BrainEpisode{
			{ClusterNodeID: "cluster-auth", ClusterKind: "authentication", Score: 0.9, MemberNodeIDs: []string{"MOBILE-1234", "PR-4423"}},
		},
		Passages: []BrainPassage{
			{SourceNodeID: "MOBILE-1234", SourceKind: "work", Text: "Users experiencing 401 errors on mobile OAuth flow."},
		},
		PromptPack: BrainPromptPack{
			ContextMarkdown: "# Brain Search Context\nQuery: " + query + "\n\nHits:\n1. Mobile Login Bug (ticket) score=0.850\n2. Fix auth token validation (pr) score=0.720",
			Citations: []map[string]interface{}{
				{"nodeId": "MOBILE-1234", "title": "Mobile Login Bug"},
			},
		},
	}
}
