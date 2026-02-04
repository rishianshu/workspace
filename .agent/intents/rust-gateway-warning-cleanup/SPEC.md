# Spec: Rust Gateway Warning Cleanup

## Changes
- `rust-gateway/build.rs`: use `compile_protos()`.
- `rust-gateway/src/routes/stream.rs`: remove unused import.
- `rust-gateway/src/proxy/grpc_client.rs`: drop unused method and error variant.
- `rust-gateway/src/config.rs`: remove unused fields.
- `rust-gateway/src/error.rs`: remove unused error variant.
