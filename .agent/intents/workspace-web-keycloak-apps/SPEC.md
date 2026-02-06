# Spec: Workspace Web Keycloak + App Subscription

## Auth
- Add Keycloak client config via env:
  - `NEXT_PUBLIC_KEYCLOAK_URL`
  - `NEXT_PUBLIC_KEYCLOAK_REALM`
  - `NEXT_PUBLIC_KEYCLOAK_CLIENT_ID`
- Create an `AuthProvider` to initialize Keycloak, expose `user`, `token`, `login`, `logout`.
- Gate UI when Keycloak is configured but user is anonymous.
- Forward `Authorization: Bearer <token>` through `/api/nucleus/graphql` proxy.

## Nucleus GraphQL
- Fetch `metadataProjects` and `metadataEndpoints(projectId)` via `/api/nucleus/graphql` with bearer token.
- Fetch `endpointTemplates` to resolve auth modes and display titles.

## App Subscription
- App catalog shown per selected project.
- For each endpoint:
  - Resolve `templateId` from endpoint `config.templateId`/`config.template_id`.
  - Display template title + auth modes.
- Subscribe flow:
  1) Upsert app instance: `POST /api/apps/instances` (templateId, instanceKey=endpointId, displayName, config).
  2) Store credentials in keystore: `POST /api/keystore/credentials` → key token.
  3) Upsert user app: `POST /api/apps/users` (userId, appInstanceId, credentialRef=keyToken).
  4) Upsert project app: `POST /api/apps/projects` (projectId, userAppId, endpointId).
- If user app already exists for the endpoint, allow linking without re-entering credentials.
- Credential form is generated from template auth `modes.requiredFields`.
  - Map fields to keystore `Credentials` (access_token, refresh_token, api_key, username, password; extras for others).

## Docs
- Update `docs/ontology/INDEX.md` with auth + app subscription workflow.
- Update `.env.example` + `docker-compose.yml` for Keycloak + keystore envs.

## Testing
- Run `scripts/record-unit-test.sh` and `scripts/record-integration-test.sh` per workflow.
