package common

import (
	"os"
	"strconv"
)

type Config struct {
	APIHost         string
	APIPort         int
	APIBaseURL      string
	Environment     string
	DBDSN           string
	RedisAddr       string
	NATSURL         string
	StorageDriver   string
	AutoMigrate     bool
	AuthMode        string
	AuthTokenTTL    int
	AuthTokenSecret string
}

func LoadConfig() Config {
	return Config{
		APIHost:         envOrDefault("CCP_API_HOST", "0.0.0.0"),
		APIPort:         envIntOrDefault("CCP_API_PORT", 8080),
		APIBaseURL:      envOrDefault("CCP_API_BASE_URL", "http://localhost:8080"),
		Environment:     envOrDefault("CCP_ENV", "development"),
		DBDSN:           envOrDefault("CCP_DB_DSN", "postgres://postgres:postgres@localhost:5432/change_control_plane?sslmode=disable"),
		RedisAddr:       envOrDefault("CCP_REDIS_ADDR", "localhost:6379"),
		NATSURL:         envOrDefault("CCP_NATS_URL", "nats://localhost:4222"),
		StorageDriver:   envOrDefault("CCP_STORAGE_DRIVER", "postgres"),
		AutoMigrate:     envBoolOrDefault("CCP_AUTO_MIGRATE", true),
		AuthMode:        envOrDefault("CCP_AUTH_MODE", "dev"),
		AuthTokenTTL:    envIntOrDefault("CCP_AUTH_TOKEN_TTL_MINUTES", 480),
		AuthTokenSecret: envOrDefault("CCP_AUTH_TOKEN_SECRET", "change-control-plane-dev-secret"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}
