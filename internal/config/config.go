package config

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"strings"
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

// generateSecureSecret generates a cryptographically secure random secret
func generateSecureSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Warning: Failed to generate secure secret, using fallback")
		return "insecure-fallback-change-me-" + time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(bytes)
}

// isWeakSecret checks if a JWT secret is too weak for use
func isWeakSecret(secret string) bool {
	weakSecrets := []string{
		"change-me-in-production-please",
		"change-me",
		"secret",
		"password",
		"jwt-secret",
		"your-secret-key",
	}
	lowerSecret := strings.ToLower(secret)
	for _, weak := range weakSecrets {
		if lowerSecret == weak || strings.Contains(lowerSecret, weak) {
			return true
		}
	}
	// Also check if it's too short (less than 32 chars)
	return len(secret) < 32
}

func Load() *Config {
	env := getEnv("ENVIRONMENT", "development")
	jwtSecret := getEnv("JWT_SECRET", "")

	// Handle JWT secret based on environment
	if jwtSecret == "" || isWeakSecret(jwtSecret) {
		if env == "production" {
			log.Fatal("FATAL: JWT_SECRET must be set to a strong value (32+ characters) in production")
		}
		// In development, generate a random secret if not set
		if jwtSecret == "" {
			jwtSecret = generateSecureSecret(32)
			log.Printf("Warning: JWT_SECRET not set, generated random secret for development")
		} else {
			log.Printf("Warning: JWT_SECRET appears weak. Use a 32+ character random string in production")
		}
	}

	// Warn about CORS wildcard in non-development
	allowedOrigins := getEnv("ALLOWED_ORIGINS", "*")
	if allowedOrigins == "*" && env != "development" {
		log.Printf("Warning: ALLOWED_ORIGINS is set to '*' which allows all origins. Restrict this in production")
	}

	// Warn about SSL mode in database connection
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pricefeed?sslmode=disable")
	if strings.Contains(dbURL, "sslmode=disable") && env == "production" {
		log.Printf("Warning: Database SSL is disabled. Enable SSL for production: sslmode=require")
	}

	return &Config{
		Port:             getEnv("PORT", "8080"),
		AllowedOrigins:   allowedOrigins,
		DatabaseURL:      dbURL,
		JWTSecret:        jwtSecret,
		JWTExpiry:        getDurationEnv("JWT_EXPIRY_HOURS", 24) * time.Hour,
		RefreshJWTExpiry: getDurationEnv("REFRESH_JWT_EXPIRY_DAYS", 7) * 24 * time.Hour,
		AdminEmail:       getEnv("ADMIN_EMAIL", "admin@pricefeed.local"),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		Environment:      env,
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
