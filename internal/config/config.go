package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Server
	Port           string
	AllowedOrigins string

	// Database
	DatabaseURL string

	// JWT
	JWTSecret        string
	JWTExpiry        time.Duration
	RefreshJWTExpiry time.Duration

	// Admin
	AdminEmail    string
	AdminPassword string

	// Environment
	Environment string

	// Google Maps
	GoogleMapsAPIKey string
}

func Load() *Config {
	return &Config{
		Port:             getEnv("PORT", "8080"),
		AllowedOrigins:   getEnv("ALLOWED_ORIGINS", "*"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pricefeed?sslmode=disable"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production-please"),
		JWTExpiry:        getDurationEnv("JWT_EXPIRY_HOURS", 24) * time.Hour,
		RefreshJWTExpiry: getDurationEnv("REFRESH_JWT_EXPIRY_DAYS", 7) * 24 * time.Hour,
		AdminEmail:       getEnv("ADMIN_EMAIL", "admin@pricefeed.local"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		Environment:      getEnv("ENVIRONMENT", "development"),
		GoogleMapsAPIKey: getEnv("GOOGLE_API_KEY_MAPS", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return time.Duration(intVal)
		}
	}
	return time.Duration(defaultValue)
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
