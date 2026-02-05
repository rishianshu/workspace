// Package config handles application configuration
package config

import (
	"os"
	"strconv"
)

// NucleusConfig holds Nucleus platform connection settings
type NucleusConfig struct {
	APIURL               string // GraphQL API URL (metadata-api)
	UCLURL               string // UCL Core gRPC URL
	Username             string // Basic auth username
	Password             string // Basic auth password
	TenantID             string // Default tenant ID
	BearerToken          string // Optional Bearer token for GraphQL auth
	KeycloakURL          string // Keycloak base URL (e.g., http://localhost:8081)
	KeycloakRealm        string // Keycloak realm (default: nucleus)
	KeycloakClientID     string // Keycloak client ID for password grant
	KeycloakClientSecret string // Optional client secret for password grant
	KeycloakUsername     string // Optional Keycloak username override
	KeycloakPassword     string // Optional Keycloak password override
}

// KeyStoreConfig holds Key Store service settings
type KeyStoreConfig struct {
	DatabaseURL string
}

// Config holds all configuration values
type Config struct {
	GRPCPort      int
	NucleusURL    string // Legacy - use Nucleus.APIURL
	UCLGatewayURL string // Legacy - use Nucleus.UCLURL
	MCPServerURL  string
	MCPAuthToken  string
	GeminiAPIKey  string
	OpenAIAPIKey  string
	PostgresURL   string
	TemporalHost  string

	// Nucleus platform config
	Nucleus  NucleusConfig
	KeyStore KeyStoreConfig
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("GRPC_PORT", "9000"))

	return &Config{
		GRPCPort:      port,
		NucleusURL:    getEnv("NUCLEUS_URL", "http://localhost:4000"),
		UCLGatewayURL: getEnv("UCL_GATEWAY_URL", "localhost:50051"),
		MCPServerURL:  getEnv("MCP_SERVER_URL", "http://localhost:9100"),
		MCPAuthToken:  getEnv("MCP_BEARER_TOKEN", ""),
		GeminiAPIKey:  getEnv("GEMINI_API_KEY", ""),
		OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
		PostgresURL:   getEnv("POSTGRES_URL", "postgres://localhost:5432/agent"),
		TemporalHost:  getEnv("TEMPORAL_HOST", "localhost:7233"),

		Nucleus: NucleusConfig{
			APIURL:               getEnv("NUCLEUS_API_URL", "http://localhost:4000/graphql"),
			UCLURL:               getEnv("NUCLEUS_UCL_URL", "localhost:50051"),
			Username:             getEnv("NUCLEUS_USERNAME", "dev-admin"),
			Password:             getEnv("NUCLEUS_PASSWORD", "password"),
			TenantID:             getEnv("NUCLEUS_TENANT_ID", "default"),
			BearerToken:          getEnv("NUCLEUS_BEARER_TOKEN", ""),
			KeycloakURL:          getEnv("KEYCLOAK_URL", "http://localhost:8081"),
			KeycloakRealm:        getEnv("KEYCLOAK_REALM", "nucleus"),
			KeycloakClientID:     getEnv("KEYCLOAK_CLIENT_ID", ""),
			KeycloakClientSecret: getEnv("KEYCLOAK_CLIENT_SECRET", ""),
			KeycloakUsername:     getEnv("KEYCLOAK_USERNAME", getEnv("NUCLEUS_USERNAME", "dev-admin")),
			KeycloakPassword:     getEnv("KEYCLOAK_PASSWORD", getEnv("NUCLEUS_PASSWORD", "password")),
		},
		KeyStore: KeyStoreConfig{
			DatabaseURL: getEnv("KEYSTORE_DATABASE_URL", getEnv("POSTGRES_URL", "postgres://localhost:5432/agent")),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
