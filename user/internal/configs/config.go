package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort      string
	DatabaseURL     string
	DatabaseMaxConn int
	JWTSecret       string
	JWTExpiryHours  int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:      getEnv("GRPC_PORT", "50052"),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
		DatabaseMaxConn: getEnvAsInt("DB_MAX_CONN", 10),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		JWTExpiryHours:  getEnvAsInt("JWT_EXPIRY_HOURS", 24),
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
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
