// Package config handles application configuration
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values
type Config struct {
	GRPCPort    int
	NucleusURL  string
	GeminiAPIKey string
	PostgresURL string
	TemporalHost string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("GRPC_PORT", "9000"))
	
	return &Config{
		GRPCPort:     port,
		NucleusURL:   getEnv("NUCLEUS_URL", "http://localhost:4000"),
		GeminiAPIKey: getEnv("GEMINI_API_KEY", ""),
		PostgresURL:  getEnv("POSTGRES_URL", "postgres://localhost:5432/agent"),
		TemporalHost: getEnv("TEMPORAL_HOST", "localhost:7233"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
