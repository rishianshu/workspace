package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestListToolsUsesAuthHeader(t *testing.T) {
	logger := zap.NewNop().Sugar()
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]ToolDefinition{{Name: "tool", Description: "t"}})
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{BaseURL: srv.URL, AuthToken: "mcp-token"}, logger)
	tools, err := client.ListTools(context.Background(), "user", "project")
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if gotAuth != "Bearer mcp-token" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
}

func TestExecuteToolUsesAuthHeader(t *testing.T) {
	logger := zap.NewNop().Sugar()
	var gotAuth string
	var gotCall ToolCall

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotCall); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Result{Success: true, Message: "ok"})
	}))
	defer srv.Close()

	client := NewClientWithConfig(ClientConfig{BaseURL: srv.URL, AuthToken: "mcp-token"}, logger)
	res, err := client.ExecuteTool(context.Background(), ToolCall{Name: "tool", Action: "act"})
	if err != nil {
		t.Fatalf("ExecuteTool error: %v", err)
	}
	if gotAuth != "Bearer mcp-token" {
		t.Fatalf("expected bearer auth, got %q", gotAuth)
	}
	if gotCall.Name != "tool" || gotCall.Action != "act" {
		t.Fatalf("unexpected tool call payload: %+v", gotCall)
	}
	if !res.Success {
		t.Fatalf("expected success")
	}
}
