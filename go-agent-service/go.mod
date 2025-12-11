module github.com/antigravity/go-agent-service

go 1.22

require (
	// Google ADK
	google.golang.org/adk v0.1.0
	
	// gRPC
	google.golang.org/grpc v1.62.0
	google.golang.org/protobuf v1.32.0
	
	// Temporal
	go.temporal.io/sdk v1.26.0
	
	// Database
	github.com/jackc/pgx/v5 v5.5.0
	github.com/pgvector/pgvector-go v0.1.1
	
	// GraphQL client
	github.com/hasura/go-graphql-client v0.12.0
	
	// Configuration
	github.com/spf13/viper v1.18.0
	
	// Logging
	go.uber.org/zap v1.26.0
	
	// Observability
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	
	// Utilities
	github.com/google/uuid v1.6.0
)
