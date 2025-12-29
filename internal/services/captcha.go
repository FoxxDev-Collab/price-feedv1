package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
)

// CaptchaService handles Cloudflare Turnstile verification
type CaptchaService struct {
	db            *database.DB
	cfg           *config.Config
	encryptionKey []byte
	httpClient    *http.Client
}

// TurnstileResponse represents the response from Cloudflare's siteverify endpoint
type TurnstileResponse struct {
	Success     bool     `json:"success"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
	ChallengeTS string   `json:"challenge_ts,omitempty"`
	Hostname    string   `json:"hostname,omitempty"`
}

// CaptchaConfig holds the captcha configuration for public use
type CaptchaConfig struct {
	Enabled bool   `json:"enabled"`
	SiteKey string `json:"site_key"`
}

// NewCaptchaService creates a new captcha service instance
func NewCaptchaService(db *database.DB, cfg *config.Config) *CaptchaService {
	return &CaptchaService{
		db:            db,
		cfg:           cfg,
		encryptionKey: DeriveEncryptionKey(cfg.JWTSecret),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetConfig returns the public captcha configuration (site key only)
func (s *CaptchaService) GetConfig(ctx context.Context) *CaptchaConfig {
	enabled := s.db.GetSettingBool(ctx, "captcha_enabled", false, s.encryptionKey)
	siteKey := s.db.GetSettingString(ctx, "captcha_site_key", "", s.encryptionKey)

	return &CaptchaConfig{
		Enabled: enabled && siteKey != "",
		SiteKey: siteKey,
	}
}

// IsEnabled returns whether captcha is enabled and configured
func (s *CaptchaService) IsEnabled(ctx context.Context) bool {
	config := s.GetConfig(ctx)
	return config.Enabled
}

// Verify verifies a Turnstile token with Cloudflare
func (s *CaptchaService) Verify(ctx context.Context, token string, remoteIP string) error {
	// If captcha is disabled, skip verification
	if !s.IsEnabled(ctx) {
		return nil
	}

	// Get secret key from database
	secretSetting, err := s.db.GetSetting(ctx, "captcha_secret_key", s.encryptionKey)
	if err != nil || secretSetting.Value == "" {
		// If no secret key configured, skip verification (shouldn't happen if enabled)
		return nil
	}
	secretKey := secretSetting.Value

	// Prepare the request to Cloudflare
	data := url.Values{}
	data.Set("secret", secretKey)
	data.Set("response", token)
	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://challenges.cloudflare.com/turnstile/v0/siteverify", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create verification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to verify captcha: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read verification response: %w", err)
	}

	var result TurnstileResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse verification response: %w", err)
	}

	if !result.Success {
		errMsg := "captcha verification failed"
		if len(result.ErrorCodes) > 0 {
			errMsg = fmt.Sprintf("captcha verification failed: %v", result.ErrorCodes)
		}
		return fmt.Errorf(errMsg)
	}

	return nil
}
