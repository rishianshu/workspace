
- title: UCL Action Schema Exposure
- slug: ucl-action-schema-exposure
- type: feature
- context:
  - platform/ucl-core (GatewayService, Endpoint Registry)
  - go-agent-service (MCP Server, Tool Registry)
- why_now: The Agent currently discovers actions but lacks their input schemas (parameters, types). To enable reliable LLM function calling, we must expose the full `ActionSchema` (serialized as JSON Schema) via the Gateway API.
- scope_in:
  - **Schema Registry**: Central registry in `ucl-core` to store `ActionSchema` definitions.
  - **Gateway Service**: `ListActions` to return `input_schema_json` derived from `ActionSchema`.
  - **Connectors**: Register action schemas for `http.jira`, `http.github`, `http.confluence`, `object.minio`, `jdbc.*`.
  - **Agent Service**: Update `GatewayServiceClient` to consume the enhanced `ListActions` response.
- scope_out:
  - New action types or capabilities (only exposing existing ones).
  - UI changes in Workspace Web (purely backend/agent enabling).
- acceptance:
  1. `grpcurl` on `GatewayService.ListActions` returns `input_schema_json` for all registered connectors.
  2. Agent's `/tools` endpoint (MCP) returns full JSON schema for tools like `jira.create_issue`.
  3. LLM can successfully generate valid arguments for these tools.
  4. No regression in existing capability-based discovery.
- constraints:
  - JSON Schema draft-07 compatibility.
  - No breaking changes to existing `ListActions` protobuf signature (additive fields).
- non_negotiables:
  - Schemas must be statically registered at startup.
- refs:
  - `requirement_ucl_schema.md` (original draft)
- status: ready
