- title: Workspace App Registration + OAuth-first Auth Flow
- slug: workspace-app-registration-oauth
- type: feature
- context:
  - Workspace app registry (app_instances, user_apps, project_apps)
  - Nucleus metadata-api (endpoints + templates + auth descriptors)
  - UCL gateway (action schemas, execution)
  - Keystore (credential storage + token exchange)
- why_now: MCP now requires app-bound tools; registration is the missing path to create app bindings, credentials, and endpoint resolution with user-level OAuth.
- scope_in:
  - Pull registered endpoints from Nucleus for a project with full config + templateId.
  - Fetch endpoint templates / auth descriptors to understand supported auth methods.
  - Registration flow that prefers OAuth when supported; supports additional config fields for endpoints.
  - Create/maintain app registry records (app_instance + user_app + project_app).
  - Store credential references in keystore (never expose raw secrets to agents).
  - Return deterministic appId handles and bind tools to appId in MCP.
- scope_out:
  - Building provider-specific OAuth UIs (Slack/Jira/GitHub screens).
  - Implementing org-level credential migration.
  - Multi-tenant SSO/SCIM integrations.
- acceptance:
  1. Registration uses Nucleus endpoints as the sole source of endpoint config + templateId.
  2. OAuth is the default path when endpoint templates declare OAuth support.
  3. App registry records are created/updated deterministically with idempotent keys.
  4. MCP tool list surfaces only app/{appId} tools for a user/project.
  5. No raw credentials are ever stored in Workspace DB or returned to agents.
- constraints:
  - Must not invent endpoint configs outside Nucleus.
  - Must honor template auth descriptors and capabilities from Nucleus.
  - Must preserve user-level audit trail for tool executions.
- non_negotiables:
  - OAuth-first when supported.
  - No special casing for individual apps.
- dependencies:
  - Nucleus GraphQL exposes endpoint config + templateId explicitly.
  - Nucleus exposes template auth descriptors in a stable schema.
- status: draft
