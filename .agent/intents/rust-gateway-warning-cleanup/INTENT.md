# Intent: Rust Gateway Warning Cleanup

## Goal
Remove build warnings in rust-gateway (deprecated tonic_build API and unused imports/fields).

## Scope
- Replace tonic_build::compile with compile_protos.
- Remove unused imports, methods, and enum variants.
- Remove unused config fields.

## Out of Scope
- Functional changes to routing behavior.
- Refactors beyond warning cleanup.

## Acceptance
- Build warnings eliminated for rust-gateway.
- No behavior change intended.
