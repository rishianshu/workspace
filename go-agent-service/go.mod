module github.com/antigravity/go-agent-service

go 1.24.4

toolchain go1.24.11

require (
	// Utilities
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9

	// Logging
	go.uber.org/zap v1.26.0

	// gRPC
	google.golang.org/grpc v1.76.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect

	// Observability
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
