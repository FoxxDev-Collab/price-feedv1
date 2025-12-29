package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/services"
)

// SettingsHandler handles settings-related API endpoints
type SettingsHandler struct {
	db            *database.DB
	cfg           *config.Config
	emailService  *services.EmailService
	encryptionKey []byte
}

// NewSettingsHandler creates a new SettingsHandler instance
func NewSettingsHandler(db *database.DB, cfg *config.Config, emailService *services.EmailService) *SettingsHandler {
	return &SettingsHandler{
		db:            db,
		cfg:           cfg,
		emailService:  emailService,
		encryptionKey: DeriveEncryptionKey(cfg.JWTSecret),
	}
}

// GetSettingsByCategory returns all settings for a given category
func (h *SettingsHandler) GetSettingsByCategory(c *fiber.Ctx) error {
	category := c.Params("category")
	if category == "" {
		return Error(c, fiber.StatusBadRequest, "category is required")
	}

	settings, err := h.db.GetSettingsByCategoryAsMap(c.Context(), category, h.encryptionKey, false)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get settings: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    settings,
	})
}

// GetAllSettings returns all settings grouped by category
func (h *SettingsHandler) GetAllSettings(c *fiber.Ctx) error {
	settings, err := h.db.GetAllSettings(c.Context(), h.encryptionKey)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get settings: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    settings,
	})
}

// UpdateSettingsRequest is the request body for updating settings
type UpdateSettingsRequest struct {
	Settings map[string]interface{} `json:"settings"`
}

// UpdateSettings updates multiple settings at once
func (h *SettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	category := c.Params("category")
	if category == "" {
		return Error(c, fiber.StatusBadRequest, "category is required")
	}

	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Convert values to strings for storage
	settingsMap := make(map[string]string)
	for key, value := range req.Settings {
		switch v := value.(type) {
		case string:
			settingsMap[key] = v
		case float64:
			settingsMap[key] = strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			settingsMap[key] = strconv.FormatBool(v)
		case int:
			settingsMap[key] = strconv.Itoa(v)
		default:
			settingsMap[key] = fmt.Sprintf("%v", v)
		}
	}

	if err := h.db.SetSettings(c.Context(), settingsMap, h.encryptionKey); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to update settings: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Settings updated successfully",
	})
}

// GetEmailConfig returns the current email configuration
func (h *SettingsHandler) GetEmailConfig(c *fiber.Ctx) error {
	config := h.emailService.GetConfig()
	return c.JSON(fiber.Map{
		"success": true,
		"data":    config,
	})
}

// TestEmailRequest is the request body for sending a test email
type TestEmailRequest struct {
	ToEmail string `json:"to_email"`
}

// SendTestEmail sends a test email to verify SMTP configuration
func (h *SettingsHandler) SendTestEmail(c *fiber.Ctx) error {
	var req TestEmailRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// If no email provided, use the admin email
	toEmail := req.ToEmail
	if toEmail == "" {
		toEmail = h.cfg.AdminEmail
	}

	// Check if SMTP is configured
	if !h.emailService.IsConfigured() {
		return Error(c, fiber.StatusBadRequest, "SMTP is not configured. Please configure SMTP settings first.")
	}

	// Send test email
	if err := h.emailService.SendTestEmail(toEmail); err != nil {
		return Error(c, fiber.StatusInternalServerError, "Failed to send test email: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Test email sent successfully to " + toEmail,
	})
}

// GetEmailStatus returns whether email service is configured and ready
func (h *SettingsHandler) GetEmailStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"configured": h.emailService.IsConfigured(),
		},
	})
}

// UpdateEmailSettingsRequest is the request body for updating email settings
type UpdateEmailSettingsRequest struct {
	Enabled  bool   `json:"smtp_enabled"`
	Host     string `json:"smtp_host"`
	Port     int    `json:"smtp_port"`
	User     string `json:"smtp_user"`
	Password string `json:"smtp_password"`
	FromAddr string `json:"smtp_from_addr"`
	FromName string `json:"smtp_from_name"`
}

// UpdateEmailSettings updates SMTP configuration
func (h *SettingsHandler) UpdateEmailSettings(c *fiber.Ctx) error {
	var req UpdateEmailSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Build settings map
	settings := map[string]string{
		"smtp_enabled":   strconv.FormatBool(req.Enabled),
		"smtp_host":      req.Host,
		"smtp_port":      strconv.Itoa(req.Port),
		"smtp_user":      req.User,
		"smtp_from_addr": req.FromAddr,
		"smtp_from_name": req.FromName,
	}

	// Only update password if a new one is provided (not masked)
	if req.Password != "" && req.Password != "••••••••" {
		settings["smtp_password"] = req.Password
	}

	if err := h.db.SetSettings(c.Context(), settings, h.encryptionKey); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to update email settings: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Email settings updated successfully",
	})
}

// StorageConfigResponse contains the S3 storage configuration for the frontend
type StorageConfigResponse struct {
	Enabled      bool   `json:"enabled"`
	Endpoint     string `json:"endpoint"`
	AccessKey    string `json:"accessKey"`
	Bucket       string `json:"bucket"`
	Region       string `json:"region"`
	UseSSL       bool   `json:"useSSL"`
	HasSecretKey bool   `json:"hasSecretKey"`
	Configured   bool   `json:"configured"`
}

// GetStorageConfig returns the current S3 storage configuration
func (h *SettingsHandler) GetStorageConfig(c *fiber.Ctx) error {
	settings, err := h.db.GetSettingsByCategoryAsMap(c.Context(), "storage", h.encryptionKey, true)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get storage settings: "+err.Error())
	}

	// Helper to get string from interface{}
	getString := func(key string) string {
		if v, ok := settings[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	enabled := getString("s3_enabled") == "true"
	endpoint := getString("s3_endpoint")
	accessKey := getString("s3_access_key")
	secretKey := getString("s3_secret_key")
	bucket := getString("s3_bucket")
	region := getString("s3_region")
	useSSL := getString("s3_use_ssl") == "true"

	// Check if configured (has all required fields)
	configured := endpoint != "" && accessKey != "" && secretKey != "" && bucket != ""

	return c.JSON(fiber.Map{
		"success": true,
		"data": StorageConfigResponse{
			Enabled:      enabled,
			Endpoint:     endpoint,
			AccessKey:    accessKey,
			Bucket:       bucket,
			Region:       region,
			UseSSL:       useSSL,
			HasSecretKey: secretKey != "",
			Configured:   configured,
		},
	})
}

// UpdateStorageSettingsRequest is the request body for updating storage settings
type UpdateStorageSettingsRequest struct {
	Enabled   bool   `json:"s3_enabled"`
	Endpoint  string `json:"s3_endpoint"`
	AccessKey string `json:"s3_access_key"`
	SecretKey string `json:"s3_secret_key"`
	Bucket    string `json:"s3_bucket"`
	Region    string `json:"s3_region"`
	UseSSL    bool   `json:"s3_use_ssl"`
}

// UpdateStorageSettings updates S3 storage configuration
func (h *SettingsHandler) UpdateStorageSettings(c *fiber.Ctx) error {
	var req UpdateStorageSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Build settings map
	settings := map[string]string{
		"s3_enabled":    strconv.FormatBool(req.Enabled),
		"s3_endpoint":   req.Endpoint,
		"s3_access_key": req.AccessKey,
		"s3_bucket":     req.Bucket,
		"s3_region":     req.Region,
		"s3_use_ssl":    strconv.FormatBool(req.UseSSL),
	}

	// Only update secret key if a new one is provided (not masked)
	if req.SecretKey != "" && req.SecretKey != "••••••••" {
		settings["s3_secret_key"] = req.SecretKey
	}

	if err := h.db.SetSettings(c.Context(), settings, h.encryptionKey); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to update storage settings: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Storage settings updated successfully",
	})
}

// TestStorageConnection tests the S3 storage connection
func (h *SettingsHandler) TestStorageConnection(c *fiber.Ctx) error {
	settings, err := h.db.GetSettingsByCategoryAsMap(c.Context(), "storage", h.encryptionKey, true)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get storage settings: "+err.Error())
	}

	// Helper to get string from interface{}
	getString := func(key string) string {
		if v, ok := settings[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	endpoint := getString("s3_endpoint")
	accessKey := getString("s3_access_key")
	secretKey := getString("s3_secret_key")
	bucket := getString("s3_bucket")
	region := getString("s3_region")
	useSSL := getString("s3_use_ssl") == "true"

	if endpoint == "" || accessKey == "" || secretKey == "" {
		return Error(c, fiber.StatusBadRequest, "Storage is not configured. Please save settings first.")
	}

	// Try to create a storage service and test connection
	storageService, err := services.NewStorageService(endpoint, accessKey, secretKey, bucket, region, useSSL)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "Failed to connect: "+err.Error())
	}

	// Try to ensure bucket exists (this tests the connection)
	if err := storageService.EnsureBucket(c.Context()); err != nil {
		return Error(c, fiber.StatusInternalServerError, "Connection failed: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Successfully connected to S3 storage. Bucket '%s' is accessible.", bucket),
	})
}

// RegenerateJWTSecret generates a new JWT secret and stores it in the database
func (h *SettingsHandler) RegenerateJWTSecret(c *fiber.Ctx) error {
	// Generate a cryptographically secure random secret (32 bytes = 256 bits)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate random secret: "+err.Error())
	}

	// Encode as base64 for storage
	newSecret := base64.StdEncoding.EncodeToString(secretBytes)

	// Store the new secret in the database (encrypted)
	settings := map[string]string{
		"jwt_secret": newSecret,
	}

	if err := h.db.SetSettings(c.Context(), settings, h.encryptionKey); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to save JWT secret: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "JWT secret regenerated successfully. All existing sessions have been invalidated.",
	})
}
