// Package mcp provides HTTP handlers for the MCP service.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Service wraps tool discovery/execution over HTTP.
type Service struct {
	uclServer   *Server
	nucleusTool Tool
	logger      *zap.SugaredLogger
}

// Tool is the minimal interface for an extra MCP tool.
type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, params map[string]any) (*Result, error)
}

// NewService creates a new MCP HTTP service.
func NewService(uclServer *Server, nucleusTool Tool, logger *zap.SugaredLogger) *Service {
	return &Service{
		uclServer:   uclServer,
		nucleusTool: nucleusTool,
		logger:      logger,
	}
}

// Handler returns an http.Handler with MCP routes.
func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tools", s.handleListTools)
	mux.HandleFunc("/v1/tools/execute", s.handleExecuteTool)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("OK"))
	})
	return mux
}

func (s *Service) handleListTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	userID := query.Get("userId")
	projectID := query.Get("projectId")
	toolsList, err := s.uclServer.ListTools(r.Context(), userID, projectID)
	if err != nil {
		s.logger.Warnw("Failed to list UCL tools", "error", err)
		if strings.Contains(err.Error(), "userId and projectId required") {
			http.Error(w, "userId and projectId required", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "app registry unavailable") {
			http.Error(w, "app registry unavailable", http.StatusServiceUnavailable)
		} else {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		return
	}

	if s.nucleusTool != nil {
		toolsList = append(toolsList, s.nucleusTool.Definition())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(toolsList)
}

func (s *Service) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ToolCall
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var result *Result
	var err error

	if req.Name == "nucleus_search" && s.nucleusTool != nil {
		params := req.Params
		if params == nil {
			params = map[string]any{}
		}
		params["action"] = req.Action
		result, err = s.nucleusTool.Execute(r.Context(), params)
	} else {
		result, err = s.uclServer.ExecuteTool(r.Context(), req)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

 
