FROM rust:1.75-slim-bookworm as builder

WORKDIR /app

# Install dependencies
RUN apt-get update && apt-get install -y protobuf-compiler && rm -rf /var/lib/apt/lists/*

# Copy source
COPY . .

# Build release
RUN cargo build --release

# Runtime image
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary
COPY --from=builder /app/target/release/gateway /app/gateway

EXPOSE 8080

CMD ["/app/gateway"]
