FROM rust:1.84-slim-bookworm as builder

WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y protobuf-compiler pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*

# Copy source
COPY rust-gateway/ ./
COPY go-agent-service/api/proto /go-agent-service/api/proto

# Build release
RUN cargo build --release

# Runtime image
FROM debian:bookworm-slim

WORKDIR /app

# Runtime deps for TLS
RUN apt-get update && apt-get install -y libssl3 ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy binary
COPY --from=builder /app/target/release/gateway /app/gateway

EXPOSE 8080

CMD ["/app/gateway"]
