# Workspace App Registration + OAuth-first Auth Flow (Spec)

## Overview
Workspace must register user-level app bindings by reading endpoint configs from Nucleus, validating auth capabilities via template descriptors, and storing credentials via keystore. The registry drives MCP tool discovery and execution.

## Primary Flows

### 1) Discover endpoints + templates
- Input: `projectId`
- Calls:
  - Nucleus `metadataEndpoints(projectId)` → endpoint instances with `config` + `templateId`.
  - Nucleus `endpointTemplates(family, refresh)` (or equivalent) → template auth descriptors + fields.
- Output: list of endpoints + supported auth methods.

### 2) Register app instance
- Choose endpoint instance from Nucleus and derive:
  - `templateId`
  - `instanceKey` (stable identifier from endpoint config, e.g. `url/tenant/host`)
- Create/Upsert `app_instances` with:
  - `template_id`, `instance_key`, `config` (non-secret)
  - display name from endpoint name

### 3) Register user app
- Input: `userId`, `appInstanceId`
- If OAuth supported:
  - Start OAuth flow → receive token → store in keystore → return `credential_ref`
- Upsert `user_apps` with `credential_ref`.

### 4) Link to project
- Input: `projectId`, `userAppId`, `endpointId`
- Upsert `project_apps` binding; optionally mark `is_default`.

### 5) MCP tool discovery
- MCP `/v1/tools?userId=&projectId=` resolves:
  - `project_apps` → `user_apps` → `app_instances` → Nucleus endpoint
  - returns `app/{appId}` tool names with UCL action schemas + read APIs

## Data Model
- `app_instances(template_id, instance_key, config, display_name)`
- `user_apps(user_id, app_instance_id, credential_ref)`
- `project_apps(project_id, user_app_id, endpoint_id, alias, is_default)`

## Auth Resolution
- Use template auth descriptor from Nucleus (preferred OAuth).
- If OAuth unsupported, require explicit credential path (API key/basic) but still use keystore.

## Errors / Contracts
- `E_APP_REGISTRY_UNAVAILABLE`: registry storage missing/unreachable.
- `E_ENDPOINT_CONFIG_MISSING`: endpoint config lacks `templateId` or required fields.
- `E_AUTH_UNSUPPORTED`: no compatible auth method found.
- `E_AUTH_REQUIRED`: tool execution requested without valid credential.

## Open Questions
- Exact schema for Nucleus endpoint auth descriptors (fields, OAuth metadata).
- Canonical `instance_key` derivation from endpoint config per template family.
- Whether Nucleus exposes template `auth` in GraphQL or via separate endpoint.

## Out-of-scope Implementation Notes
- UI flows for OAuth provider selection.
- Tenant/org-level credential sharing.

