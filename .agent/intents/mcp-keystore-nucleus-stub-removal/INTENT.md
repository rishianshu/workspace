# Intent: MCP + Keystore Services and Nucleus Stub Removal

## Goal
Introduce standalone MCP and Keystore services, wire them into the dev stack, and remove the local nucleus-stub in favor of external Nucleus APIs.

## Scope
- Add MCP server and keystore service containers and env wiring.
- Remove nucleus-stub from the repo and compose.
- Document the service layout and required envs.

## Out of Scope
- Production deployment hardening.
- OAuth/registration flows or endpoint metadata enhancements.
- CI/CD enforcement changes.

## Acceptance
- MCP and Keystore services exist and are wired in docker-compose.
- nucleus-stub is removed from repo and compose references.
- README/docs reflect the new service layout and required envs.
