package configs

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Services ServicesConfig
}

type ServerConfig struct {
	Port int
}

type ServicesConfig struct {
	UserServiceAddr     string
	DocumentServiceAddr string
	ConsentServiceAddr  string
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
