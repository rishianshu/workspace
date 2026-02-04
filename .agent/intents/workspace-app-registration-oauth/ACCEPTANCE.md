1. Endpoint discovery pulls only from Nucleus and returns `templateId` + config fields needed for registration.
2. Registration prefers OAuth when template auth descriptors indicate support.
3. App registry writes are idempotent by `(templateId, instanceKey)` and `(userId, appInstanceId)`.
4. Project linking binds a `userAppId` to a `projectId` with a concrete Nucleus `endpointId`.
5. MCP `/v1/tools` returns `app/{appId}` tools scoped to `userId` + `projectId`.
6. Credentials are stored only in keystore; Workspace DB stores opaque `credential_ref`.
7. Tool execution fails with `auth_required` (or equivalent) if no credential ref exists.
