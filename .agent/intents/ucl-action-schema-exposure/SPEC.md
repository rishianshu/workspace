# SPEC — UCL Action Schema Exposure

## Problem
Currently, `UCL` connectors define actions with input fields, but this rich schema information is lost when the Agent discovers actions via `GatewayService`. The Agent only sees action names or basic capabilities. For an LLM to call these actions (Function Calling), it needs a precise JSON Schema defining required fields, types, and descriptions.

## Interfaces / Contracts

### A) Schema Registry (`internal/endpoint`)
- **Mechanism**: A thread-safe registry to store `ActionSchema` structs keyed by action name.
- **Functions**:
  - `RegisterActionSchema(actionName string, schema ActionSchema)`
  - `GetRegisteredActionSchema(actionName string) *ActionSchema`
- **Usage**: Connectors register their schemas in their `init()` or constructor.

### B) Gateway Service Enhancement (`internal/gateway`)
- **Protobuf Update**: Add `string input_schema_json` to the `ActionDTO` or equivalent in `gateway.proto` (if not already present, we reusing `ActionSchema` struct which might need a JSON field).
- **Serialization Logic**:
  - Implement `actionSchemaToJSONSchema(schema ActionSchema) string`.
  - Maps internal `DataType` (String, Integer, Boolean) to JSON Schema types.
  - Generates `required` list and `properties` map.
- **ListActions Flow**:
  1. Retrieve registered actions.
  2. For each action, look up its `ActionSchema`.
  3. Serialize to JSON Schema.
  4. Populate response field.

### C) Logic & Data Flow
```mermaid
graph LR
    Connector[Connector (Jira/GitHub)] -->|RegisterSchema| Registry[Schema Registry]
    Agent -->|ListActions| Gateway
    Gateway -->|Lookup| Registry
    Gateway -->|Serialize| JSON[JSON Schema]
    Gateway -->|Return| Agent
```

## Data & State
- **Stateless**: Schemas are static code definitions registered at memory startup. No DB changes.

## Constraints
- **Performance**: Serialization happens on `ListActions`. Since tool counts are low (<100), dynamic serialization is acceptable.
- **Complexity**: Keep JSON schema generation simple (flat properties, basic types) for v1.

## Acceptance Mapping
- **AC1** (Gateway API): `grpcurl` output shows `input_schema_json`.
- **AC2** (Agent Tools): MCP `/tools` endpoint returns valid tool definitions with schemas.
