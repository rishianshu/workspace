FROM golang:1.24-alpine as builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache protobuf-dev make

# Copy go.mod first for caching
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Runtime image
FROM alpine:3.19

WORKDIR /app

# Copy binary
COPY --from=builder /server /app/server

EXPOSE 9000

CMD ["/app/server"]
