# keystore

Purpose
- KMS-backed credential storage and retrieval for tools and connectors.
 - Stores user-scoped credentials separate from endpoint definitions.

Entry Point
- `go-agent-service/cmd/keystore/main.go`

Port
- 9200 (default `KEYSTORE_PORT`)

Expected Responsibilities
- Encrypted secret storage with scoped access.
- Token lifecycle (expiry, refresh metadata).
- Auditable access logs.
- User-scoped credentials (separate from org-level Nucleus credentials).
