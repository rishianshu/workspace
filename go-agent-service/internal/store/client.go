// Package store provides gRPC clients for Nucleus Store Core services
package store

import (
	"context"
	"fmt"

	"github.com/antigravity/go-agent-service/internal/store/kvpb"
	"github.com/antigravity/go-agent-service/internal/store/vectorpb"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps gRPC clients for Store Core services (KV, Vector, Graph)
type Client struct {
	conn     *grpc.ClientConn
	logger   *zap.SugaredLogger
	addr     string
	kvClient kvpb.KVServiceClient
	vecClient vectorpb.VectorServiceClient
}

// NewClient creates a new Store Core client
func NewClient(addr string, logger *zap.SugaredLogger) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Store Core: %w", err)
	}

	logger.Infow("Connected to Store Core", "addr", addr)
	return &Client{
		conn:      conn,
		logger:    logger,
		addr:      addr,
		kvClient:  kvpb.NewKVServiceClient(conn),
		vecClient: vectorpb.NewVectorServiceClient(conn),
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ========================
// KV Operations
// ========================

// KVGet retrieves a value by key
func (c *Client) KVGet(ctx context.Context, tenantID, projectID, key string) ([]byte, error) {
	c.logger.Debugw("KV Get", "tenant", tenantID, "project", projectID, "key", key)

	resp, err := c.kvClient.Get(ctx, &kvpb.GetRequest{
		Key: &kvpb.ScopedKey{
			TenantId:  tenantID,
			ProjectId: projectID,
			Key:       key,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
}

// KVPut stores a value by key
func (c *Client) KVPut(ctx context.Context, tenantID, projectID, key string, value []byte) error {
	c.logger.Debugw("KV Put", "tenant", tenantID, "project", projectID, "key", key, "size", len(value))

	_, err := c.kvClient.Put(ctx, &kvpb.PutRequest{
		Key: &kvpb.ScopedKey{
			TenantId:  tenantID,
			ProjectId: projectID,
			Key:       key,
		},
		Value:       value,
		ContentType: "application/json",
	})
	return err
}

// KVDelete removes a value by key
func (c *Client) KVDelete(ctx context.Context, tenantID, projectID, key string) error {
	c.logger.Debugw("KV Delete", "tenant", tenantID, "project", projectID, "key", key)

	_, err := c.kvClient.Delete(ctx, &kvpb.DeleteRequest{
		Key: &kvpb.ScopedKey{
			TenantId:  tenantID,
			ProjectId: projectID,
			Key:       key,
		},
	})
	return err
}

// ========================
// Vector Operations
// ========================

// VectorSearchHit represents a search result
type VectorSearchHit struct {
	NodeID      string         `json:"nodeId"`
	ProfileID   string         `json:"profileId"`
	Score       float32        `json:"score"`
	ContentText string         `json:"contentText"`
	Metadata    map[string]any `json:"metadata"`
}

// VectorSearch performs semantic similarity search
func (c *Client) VectorSearch(ctx context.Context, tenantID, projectID string, embedding []float32, topK int) ([]VectorSearchHit, error) {
	c.logger.Debugw("Vector Search", "tenant", tenantID, "project", projectID, "topK", topK)

	resp, err := c.vecClient.Search(ctx, &vectorpb.SearchRequest{
		TenantId:  tenantID,
		ProjectId: projectID,
		TopK:      int32(topK),
		Embedding: embedding,
	})
	if err != nil {
		return nil, err
	}

	hits := make([]VectorSearchHit, 0, len(resp.Hits))
	for _, h := range resp.Hits {
		metadata := make(map[string]any)
		if h.Metadata != nil {
			metadata = h.Metadata.AsMap()
		}
		hits = append(hits, VectorSearchHit{
			NodeID:      h.NodeId,
			ProfileID:   h.ProfileId,
			Score:       h.Score,
			ContentText: h.ContentText,
			Metadata:    metadata,
		})
	}
	return hits, nil
}

// ========================
// Graph Operations
// ========================

// GraphNode represents a node in the graph
type GraphNode struct {
	NodeID     string         `json:"nodeId"`
	NodeType   string         `json:"nodeType"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

// GraphEdge represents an edge in the graph
type GraphEdge struct {
	FromID       string `json:"fromId"`
	ToID         string `json:"toId"`
	Relationship string `json:"relationship"`
}

// GraphQuery performs graph traversal (requires graph proto - stub for now)
func (c *Client) GraphQuery(ctx context.Context, tenantID, projectID, nodeID string, depth int) ([]GraphNode, []GraphEdge, error) {
	c.logger.Debugw("Graph Query", "tenant", tenantID, "project", projectID, "nodeId", nodeID, "depth", depth)

	// Graph proto not yet available - return empty results
	return []GraphNode{}, []GraphEdge{}, nil
}
