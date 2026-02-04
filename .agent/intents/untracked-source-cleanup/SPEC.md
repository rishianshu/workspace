# Spec: Untracked Source Cleanup

## Files to Commit
- go-agent-service/internal/agent/llm_router.go
- go-agent-service/internal/agent/openai_client.go
- go-agent-service/internal/endpoints/*
- go-agent-service/internal/workflow/temporal_client.go
- go-agent-service/internal/workflow/temporal_workflow.go
- go-agent-service/internal/server/agent.pb.go
- go-agent-service/internal/server/agent_grpc.pb.go
- go-agent-service/internal/server/context_keys.go
- go-agent-service/api/proto/agent.pb.go
- go-agent-service/api/proto/agent_grpc.pb.go
- start-dev.sh

## Files to Ignore
- go-agent-service/agent-service
