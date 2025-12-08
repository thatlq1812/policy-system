package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	GRPCPort    string
	DatabaseURL string
	DBMaxConn   int
}

func Load() (*Config, error) {
	// Load .env file (optional - for local development)
	_ = godotenv.Load()

	cfg := &Config{
		GRPCPort:    getEnv("GRPC_PORT", "50053"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBMaxConn:   getEnvAsInt("DB_MAX_CONN", 10),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
