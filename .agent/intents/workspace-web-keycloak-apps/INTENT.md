# Intent: Workspace Web Keycloak + App Subscription

## Goal
Add Keycloak-based authentication to workspace-web and expose app subscription for Nucleus endpoints so users can authenticate apps per project.

## Scope
- Integrate Keycloak auth (same realm, separate client) in workspace-web UI.
- Load projects and endpoints via Nucleus GraphQL with the user token.
- Show app catalog per project and allow users to subscribe/connect apps.
- Store credentials in keystore and register app/user/project bindings in app registry.
- Update ontology/docs to reflect auth + app subscription flow.

## Out of Scope
- Implementing OAuth redirect flows for external apps (only capture credentials/keys for now).
- Changes to Nucleus metadata-api schema.

## Acceptance
- Workspace web prompts for Keycloak login and exposes user identity.
- Projects and endpoints load via Nucleus GraphQL using the user token.
- App catalog lists endpoints for selected project and shows auth modes.
- Subscribing an app stores credentials in keystore and creates app instance + user app + project app.
- Docs describe auth + app subscription flow and keystore usage.
