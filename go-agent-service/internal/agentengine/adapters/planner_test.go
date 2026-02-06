package adapters

import (
	"context"
	"reflect"
	"testing"

	"github.com/antigravity/go-agent-service/internal/agentengine"
)

func TestHeuristicPlannerDeterminism(t *testing.T) {
	planner := NewHeuristicPlanner()
	input := agentengine.PlanInput{
		Request: agentengine.Request{
			Query:     "tool:app/jira.search",
			UserID:    "user-1",
			ProjectID: "project-1",
		},
		Tools: []agentengine.ToolDef{
			{
				Name:        "app/jira",
				Description: "Jira tool",
				Actions: []agentengine.ToolAction{
					{Name: "search", Description: "Search issues"},
				},
			},
		},
	}

	plan1, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plan2, err := planner.Plan(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(plan1, plan2) {
		t.Fatalf("planner output is not deterministic: %+v vs %+v", plan1, plan2)
	}
}
