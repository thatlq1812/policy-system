package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server   ServerConfig
	Services ServicesConfig
	JWT      JWTConfig

	// Timeouts
	GRPCDialTimeout time.Duration
	GRPCcallTimeout time.Duration

	// CORS
	AllowedOrigins     []string
	AllowedCredentials bool

	// Logging
	LogLevel string
}

type ServerConfig struct {
	Port int
}

// GetAddr returns the server address in format ":port"
func (s ServerConfig) GetAddr() string {
	return fmt.Sprintf(":%d", s.Port)
}

type ServicesConfig struct {
	UserServiceAddr     string
	DocumentServiceAddr string
	ConsentServiceAddr  string
}

type JWTConfig struct {
	Secret     string
	Expiration int // hours
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnvAsInt("SERVER_PORT", 8080),
		},
		Services: ServicesConfig{
			UserServiceAddr:     getEnv("USER_SERVICE_ADDR", "localhost:50052"),
			DocumentServiceAddr: getEnv("DOCUMENT_SERVICE_ADDR", "localhost:50051"),
			ConsentServiceAddr:  getEnv("CONSENT_SERVICE_ADDR", "localhost:50053"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
			Expiration: getEnvAsInt("JWT_EXPIRATION", 24), // 24 hours
		},

		// Timeouts
		GRPCDialTimeout: getEnvAsDuration("GRPC_DIAL_TIMEOUT", 5*time.Second),
		GRPCcallTimeout: getEnvAsDuration("GRPC_CALL_TIMEOUT", 10*time.Second),

		// CORS
		AllowedOrigins:     getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		AllowedCredentials: getEnvAsBool("ALLOW_CREDENTIALS", true),

		// Logging
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Split by comma and trim spaces
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, item := range parts {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
