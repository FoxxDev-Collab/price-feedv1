package handlers

import (
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
	key := make([]byte, 32)
	copy(key, []byte(cfg.JWTSecret))

	return &SettingsHandler{
		db:            db,
		cfg:           cfg,
		emailService:  emailService,
		encryptionKey: key,
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
