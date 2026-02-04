# Spec: Agent + Gateway Risk Fixes

## Interfaces
- rust-gateway chat uses AGENT_SERVICE_URL with default http://localhost:9001.
- go-agent-service HTTP mux includes /action.
- ChatHTTPRequest accepts attachedFiles.
- Workflow handlers check engine availability.
