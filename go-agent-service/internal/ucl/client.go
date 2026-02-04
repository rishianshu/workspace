// Package ucl provides the UCL Gateway client using Nucleus UCLService proto
package ucl

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/antigravity/go-agent-service/internal/ucl/gatewaypb"
	"github.com/antigravity/go-agent-service/internal/ucl/uclpb"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"
)

// Client wraps the Nucleus UCL gRPC client
type Client struct {
	conn      *grpc.ClientConn
	ucl       uclpb.UCLServiceClient
	gateway   gatewaypb.GatewayServiceClient
	logger    *zap.SugaredLogger
	endpoints []string
}

// NewClient creates a new UCL client
func NewClient(address string, logger *zap.SugaredLogger) (*Client, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to UCL: %w", err)
	}

	return &Client{
		conn:    conn,
		ucl:     uclpb.NewUCLServiceClient(conn),
		gateway: gatewaypb.NewGatewayServiceClient(conn),
		logger:  logger,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// EndpointDescription represents an endpoint template for compatibility
type EndpointDescription struct {
	Id          string
	TemplateId  string
	DisplayName string
	Family      string
	Description string
}

// ActionSchema represents an action capability for compatibility
type ActionSchema struct {
	Name            string
	Description     string
	InputSchemaJSON string
}

// DatasetItem represents a dataset from the Gateway service.
type DatasetItem struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Kind               string `json:"kind"`
	SupportsIncremental bool   `json:"supportsIncremental"`
	CdmModelID         string `json:"cdmModelId"`
	IngestionStrategy  string `json:"ingestionStrategy"`
}

// FieldDefinition represents dataset schema fields.
type FieldDefinition struct {
	Name      string `json:"name"`
	DataType  string `json:"dataType"`
	Nullable  bool   `json:"nullable"`
	Precision int32  `json:"precision"`
	Scale     int32  `json:"scale"`
	Comment   string `json:"comment"`
}

// Constraint represents schema constraints.
type Constraint struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Fields []string `json:"fields"`
}

// DatasetStatistics represents dataset stats.
type DatasetStatistics struct {
	RowCount   int64 `json:"rowCount"`
	SizeBytes  int64 `json:"sizeBytes"`
	Partitions int32 `json:"partitions"`
}

// Schema represents a dataset schema response.
type Schema struct {
	Fields      []FieldDefinition `json:"fields"`
	Constraints []Constraint      `json:"constraints"`
	Statistics  DatasetStatistics `json:"statistics"`
}

// ListEndpoints returns all available endpoint templates as endpoints
func (c *Client) ListEndpoints(ctx context.Context) ([]*EndpointDescription, error) {
	resp, err := c.ucl.ListEndpointTemplates(ctx, &uclpb.ListTemplatesRequest{})
	if err != nil {
		return nil, err
	}

	// Convert templates to endpoint descriptions
	endpoints := make([]*EndpointDescription, 0, len(resp.Templates))
	for _, t := range resp.Templates {
		endpoints = append(endpoints, &EndpointDescription{
			Id:          t.Id,
			TemplateId:  t.Id,
			DisplayName: t.DisplayName,
			Family:      t.Family,
			Description: t.Description,
		})
	}
	return endpoints, nil
}

// ListActions returns actions for a template using GatewayService.ListActions API and UCL capabilities
func (c *Client) ListActions(ctx context.Context, templateID string) ([]*ActionSchema, error) {
	c.logger.Debugw("ListActions called", "templateID", templateID)

	var actions []*ActionSchema

	// 1. Get Write Actions (GatewayService)
	gwResp, err := c.gateway.ListActions(ctx, &gatewaypb.ListActionsRequest{
		EndpointTemplateId: templateID,
	})
	if err == nil && len(gwResp.Actions) > 0 {
		c.logger.Infow("Got actions from GatewayService", "templateID", templateID, "count", len(gwResp.Actions))
		for _, a := range gwResp.Actions {
			actions = append(actions, &ActionSchema{
				Name:            a.Name,
				Description:     a.Description,
				InputSchemaJSON: a.InputSchemaJson,
			})
		}
	} else if err != nil {
		c.logger.Debugw("GatewayService.ListActions failed or empty", "error", err)
	}

	// 2. Get Read Capabilities (UCLService)
	resp, err := c.ucl.ListEndpointTemplates(ctx, &uclpb.ListTemplatesRequest{
		Family: "", // Don't filter by family
	})
	if err == nil {
		// Find the template and extract capabilities
		for _, t := range resp.Templates {
			if t.Id == templateID {
				for _, cap := range t.Capabilities {
					actions = append(actions, &ActionSchema{
						Name:        cap.Key,
						Description: cap.Description,
						// Read capabilities don't currently expose schemas in ListEndpointTemplates
						InputSchemaJSON: "", 
					})
				}
				break
			}
		}
	} else {
		c.logger.Warnw("UCL ListEndpointTemplates failed", "error", err)
	}

	return actions, nil
}

// ListDatasets returns datasets for an endpoint.
func (c *Client) ListDatasets(ctx context.Context, endpointID string) ([]DatasetItem, error) {
	resp, err := c.gateway.ListDatasets(ctx, &gatewaypb.ListDatasetsRequest{
		EndpointId: endpointID,
	})
	if err != nil {
		return nil, err
	}

	datasets := make([]DatasetItem, 0, len(resp.Datasets))
	for _, d := range resp.Datasets {
		datasets = append(datasets, DatasetItem{
			ID:                 d.Id,
			Name:               d.Name,
			Kind:               d.Kind,
			SupportsIncremental: d.SupportsIncremental,
			CdmModelID:         d.CdmModelId,
			IngestionStrategy:  d.IngestionStrategy,
		})
	}
	return datasets, nil
}

// GetSchema returns schema for a dataset.
func (c *Client) GetSchema(ctx context.Context, endpointID, datasetID string) (*Schema, error) {
	resp, err := c.gateway.GetSchema(ctx, &gatewaypb.GetSchemaRequest{
		EndpointId: endpointID,
		DatasetId:  datasetID,
	})
	if err != nil {
		return nil, err
	}

	fields := make([]FieldDefinition, 0, len(resp.Fields))
	for _, f := range resp.Fields {
		fields = append(fields, FieldDefinition{
			Name:      f.Name,
			DataType:  f.DataType,
			Nullable:  f.Nullable,
			Precision: f.Precision,
			Scale:     f.Scale,
			Comment:   f.Comment,
		})
	}

	constraints := make([]Constraint, 0, len(resp.Constraints))
	for _, cst := range resp.Constraints {
		constraints = append(constraints, Constraint{
			Name:   cst.Name,
			Type:   cst.Type,
			Fields: cst.Fields,
		})
	}

	stats := DatasetStatistics{}
	if resp.Statistics != nil {
		stats = DatasetStatistics{
			RowCount:   resp.Statistics.RowCount,
			SizeBytes:  resp.Statistics.SizeBytes,
			Partitions: resp.Statistics.Partitions,
		}
	}

	return &Schema{
		Fields:      fields,
		Constraints: constraints,
		Statistics:  stats,
	}, nil
}

// ReadData streams records for a dataset.
func (c *Client) ReadData(ctx context.Context, endpointID, datasetID string, filter map[string]any, limit int64) ([]map[string]any, error) {
	var pbFilter *structpb.Struct
	if filter != nil {
		f, err := structpb.NewStruct(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal filter: %w", err)
		}
		pbFilter = f
	}

	stream, err := c.gateway.ReadData(ctx, &gatewaypb.ReadDataRequest{
		EndpointId: endpointID,
		DatasetId:  datasetID,
		Filter:     pbFilter,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	records := make([]map[string]any, 0)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if resp.Record != nil {
			records = append(records, resp.Record.AsMap())
		}
	}

	return records, nil
}

// ExecuteActionResponse wraps action execution result
type ExecuteActionResponse struct {
	ExecutionId string
	Success     bool
	Message     string
	Result      map[string]any
}

// ExecuteAction runs an operation via GatewayService (Write) or UCLService (Read/Long-running)
func (c *Client) ExecuteAction(ctx context.Context, endpointID, actionName string, params map[string]any, async bool) (*ExecuteActionResponse, error) {
	// 1. Check for UCL Read/Long-running capabilities
	kind := resolveOperationKind(actionName)
	if kind != uclpb.OperationKind_OPERATION_KIND_UNSPECIFIED {
		// Convert params to string map
		strParams := make(map[string]string)
		for k, v := range params {
			strParams[k] = fmt.Sprintf("%v", v)
		}

		resp, err := c.ucl.StartOperation(ctx, &uclpb.StartOperationRequest{
			TemplateId: endpointID,
			EndpointId: endpointID,
			Kind:       kind,
			Parameters: strParams,
		})
		if err != nil {
			return nil, err
		}

		return &ExecuteActionResponse{
			ExecutionId: resp.OperationId,
			Success:     true,
			Message:     fmt.Sprintf("Started operation %s", resp.OperationId),
			Result:      map[string]any{"operation_id": resp.OperationId},
		}, nil
	}

	// 2. Default to GatewayService (Write Plane / Sync Actions)
	// Convert params to structpb
	pbParams, err := structpb.NewStruct(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params to structpb: %w", err)
	}

	resp, err := c.gateway.ExecuteAction(ctx, &gatewaypb.ExecuteActionRequest{
		EndpointId: endpointID, // Using endpointID (likely template ID if no instance)
		ActionName: actionName,
		Parameters: pbParams,
		// Mode: default (SYNC)
	})
	if err != nil {
		return nil, err
	}

	resultMap := make(map[string]any)
	if resp.Result != nil {
		resultMap = resp.Result.AsMap()
	}

	return &ExecuteActionResponse{
		ExecutionId: resp.ExecutionId,
		Success:     true, // Gateway execution successful if no error
		Message:     "Executed successfully",
		Result:      resultMap,
	}, nil
}

func resolveOperationKind(actionName string) uclpb.OperationKind {
	switch actionName {
	case "metadata", "metadata.run":
		return uclpb.OperationKind_METADATA_RUN
	case "preview", "preview.run":
		return uclpb.OperationKind_PREVIEW_RUN
	case "ingestion", "ingestion.run":
		return uclpb.OperationKind_INGESTION_RUN
	default:
		return uclpb.OperationKind_OPERATION_KIND_UNSPECIFIED
	}
}

// ========================
// Stub Registry (fallback)
// ========================

// StubToolRegistry provides fallback tool definitions
type StubToolRegistry struct {
	logger *zap.SugaredLogger
	tools  map[string]StubTool
	mu     sync.RWMutex
}

// StubTool represents a stub tool definition
type StubTool struct {
	TemplateID  string
	DisplayName string
	Actions     []string
}

// NewStubToolRegistry creates a fallback registry
func NewStubToolRegistry(logger *zap.SugaredLogger) *StubToolRegistry {
	return &StubToolRegistry{
		logger: logger,
		tools: map[string]StubTool{
			"http.jira": {
				TemplateID:  "http.jira",
				DisplayName: "Jira",
				Actions:     []string{"search_issues", "get_issue", "create_issue", "add_comment", "update_status", "assign_issue"},
			},
			"http.github": {
				TemplateID:  "http.github",
				DisplayName: "GitHub",
				Actions:     []string{"get_pr", "list_prs", "approve_pr", "request_changes", "get_file", "list_commits"},
			},
			"http.slack": {
				TemplateID:  "http.slack",
				DisplayName: "Slack",
				Actions:     []string{"send_message", "list_channels", "get_thread", "add_reaction"},
			},
			"http.pagerduty": {
				TemplateID:  "http.pagerduty",
				DisplayName: "PagerDuty",
				Actions:     []string{"list_incidents", "acknowledge", "resolve", "escalate"},
			},
			"http.confluence": {
				TemplateID:  "http.confluence",
				DisplayName: "Confluence",
				Actions:     []string{"search_pages", "get_page", "create_page", "update_page"},
			},
		},
	}
}

// ListTools returns stub tool definitions
func (r *StubToolRegistry) ListTools() []StubTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]StubTool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ExecuteStub simulates tool execution
func (r *StubToolRegistry) ExecuteStub(toolName, action string, params map[string]any) map[string]any {
	r.logger.Infow("Executing stub tool", "tool", toolName, "action", action)

	return map[string]any{
		"tool":       toolName,
		"action":     action,
		"executed":   true,
		"stub":       true,
		"timestamp":  time.Now().Unix(),
	}
}

// GetActions returns actions for a tool (for compatibility)
func (r *StubToolRegistry) GetActions() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	total := 0
	for _, t := range r.tools {
		total += len(t.Actions)
	}
	return total
}

// FormatForLLM returns a formatted string for LLM consumption
func (r *StubToolRegistry) FormatForLLM() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var result string
	for _, t := range r.tools {
		result += fmt.Sprintf("- %s: %v\n", t.TemplateID, t.Actions)
	}
	return result
}

// Execute implements UCLExecutor interface for workflow activities
func (r *StubToolRegistry) Execute(ctx context.Context, endpointID, actionName string, params map[string]any) (map[string]any, error) {
	r.logger.Infow("Executing UCL tool", "endpoint", endpointID, "action", actionName)
	
	result := r.ExecuteStub(endpointID, actionName, params)
	return result, nil
}
