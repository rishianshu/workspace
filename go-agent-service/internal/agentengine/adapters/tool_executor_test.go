package adapters

import (
	"context"
	"reflect"
	"testing"

	"github.com/antigravity/go-agent-service/internal/agentengine"
	"github.com/antigravity/go-agent-service/internal/tools"
)

type stubExecutor struct {
	gotName   string
	gotAction string
	gotParams map[string]any
	result    *tools.Result
}

func (s *stubExecutor) Execute(ctx context.Context, name, action string, params map[string]any) (*tools.Result, error) {
	_ = ctx
	s.gotName = name
	s.gotAction = action
	s.gotParams = params
	return s.result, nil
}

func TestRegistryExecutorExecute(t *testing.T) {
	stub := &stubExecutor{
		result: &tools.Result{
			Success: true,
			Data:    map[string]any{"ok": true},
			Message: "done",
		},
	}
	executor := NewRegistryExecutor(stub)

	call := agentengine.ToolCall{
		Name:   "app/jira",
		Action: "search",
		Args:   map[string]any{"query": "status = Open"},
	}

	result, err := executor.Execute(context.Background(), call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stub.gotName != call.Name || stub.gotAction != call.Action {
		t.Fatalf("expected execute called with %s/%s, got %s/%s", call.Name, call.Action, stub.gotName, stub.gotAction)
	}
	if !reflect.DeepEqual(stub.gotParams, call.Args) {
		t.Fatalf("expected params %+v, got %+v", call.Args, stub.gotParams)
	}
	if result == nil || !result.Success {
		t.Fatalf("unexpected result: %+v", result)
	}
}
