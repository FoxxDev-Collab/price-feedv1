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

	// SMTP Email
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFromAddr string
	SMTPFromName string
	SMTPEnabled  bool

	// S3/Garage Storage
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3UseSSL    bool
	S3Region    string
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
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getIntEnv("SMTP_PORT", 587),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPassword:     getEnv("SMTP_PASSWORD", ""),
		SMTPFromAddr:     getEnv("SMTP_FROM_ADDR", "noreply@pricefeed.app"),
		SMTPFromName:     getEnv("SMTP_FROM_NAME", "PriceFeed"),
		SMTPEnabled:      getBoolEnv("SMTP_ENABLED", false),
		S3Endpoint:       getEnv("S3_ENDPOINT", "localhost:3900"),
		S3AccessKey:      getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getEnv("S3_SECRET_KEY", ""),
		S3Bucket:         getEnv("S3_BUCKET", "receipts"),
		S3UseSSL:         getBoolEnv("S3_USE_SSL", false),
		S3Region:         getEnv("S3_REGION", "garage"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
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
