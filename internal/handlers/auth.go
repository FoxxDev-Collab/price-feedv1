package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
)

// encryptionSalt for PBKDF2 - must match settings_repo.go
var encryptionSalt = []byte("pricefeed-settings-v1")

// DeriveEncryptionKey derives a secure 32-byte key using PBKDF2
func DeriveEncryptionKey(secret string) []byte {
	return pbkdf2.Key([]byte(secret), encryptionSalt, 100000, 32, sha256.New)
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// getEncryptionKey returns the encryption key for settings using PBKDF2
func (h *Handler) getEncryptionKey() []byte {
	// Use the same key derivation as settings_repo
	return DeriveEncryptionKey(h.cfg.JWTSecret)
}

// isEmailVerificationRequired checks if email verification is enabled
func (h *Handler) isEmailVerificationRequired(c *fiber.Ctx) bool {
	return h.db.GetSettingBool(c.Context(), "require_email_verify", false, h.getEncryptionKey())
}

// GetCaptchaConfig returns the public captcha configuration
func (h *Handler) GetCaptchaConfig(c *fiber.Ctx) error {
	config := h.captchaService.GetConfig(c.Context())
	return Success(c, config)
}

// Register handles user registration
func (h *Handler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Verify captcha if enabled
	if err := h.captchaService.Verify(c.Context(), req.CaptchaToken, c.IP()); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	// Validate email
	if !emailRegex.MatchString(req.Email) {
		return Error(c, fiber.StatusBadRequest, "invalid email format")
	}

	// Validate password
	if len(req.Password) < 8 {
		return Error(c, fiber.StatusBadRequest, "password must be at least 8 characters")
	}

	// Validate username if provided
	if req.Username != nil {
		if len(*req.Username) < 3 || len(*req.Username) > 50 {
			return Error(c, fiber.StatusBadRequest, "username must be between 3 and 50 characters")
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to process password")
	}

	// Create user (pass full request to include location fields)
	user, err := h.db.CreateUser(c.Context(), req.Email, string(hashedPassword), req.Username, req.RegionID, &req)
	if err != nil {
		if errors.Is(err, database.ErrEmailExists) {
			return Error(c, fiber.StatusConflict, "email already registered")
		}
		if errors.Is(err, database.ErrUsernameExists) {
			return Error(c, fiber.StatusConflict, "username already taken")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to create user")
	}

	// Check if email verification is required
	requireVerification := h.isEmailVerificationRequired(c)

	// Send verification email if required and email service is configured
	if requireVerification && h.emailService.IsConfiguredWithContext(c.Context()) {
		verifyToken, err := generateSecureToken()
		if err == nil {
			// Token expires in 24 hours
			expiresAt := time.Now().Add(24 * time.Hour)
			_, err = h.db.CreateEmailVerificationToken(c.Context(), user.ID, verifyToken, expiresAt)
			if err == nil {
				// Get the base URL from the request
				scheme := "https"
				if c.Protocol() == "http" {
					scheme = "http"
				}
				baseURL := scheme + "://" + c.Hostname()
				verifyURL := baseURL + "/verify-email"

				// Send verification email in background with timeout
				go func(email, token, url string) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					// Use a channel to wait for completion or timeout
					done := make(chan error, 1)
					go func() {
						done <- h.emailService.SendEmailVerificationEmail(email, token, url)
					}()

					select {
					case err := <-done:
						if err != nil {
							log.Printf("Warning: Failed to send verification email to %s: %v", email, err)
						}
					case <-ctx.Done():
						log.Printf("Warning: Verification email to %s timed out", email)
					}
				}(user.Email, verifyToken, verifyURL)
			}
		}
	}

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	response := fiber.Map{
		"token":                    token,
		"user":                     user,
		"email_verification_sent":  requireVerification && h.emailService.IsConfiguredWithContext(c.Context()),
		"email_verification_required": requireVerification,
	}

	return c.Status(fiber.StatusCreated).JSON(response)
}

// Login handles user authentication
func (h *Handler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Verify captcha if enabled
	if err := h.captchaService.Verify(c.Context(), req.CaptchaToken, c.IP()); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		return Error(c, fiber.StatusBadRequest, "email and password are required")
	}

	// Get user by email
	user, err := h.db.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusUnauthorized, "invalid credentials")
		}
		return Error(c, fiber.StatusInternalServerError, "authentication failed")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return Error(c, fiber.StatusUnauthorized, "invalid credentials")
	}

	// Update last login
	h.db.UpdateUserLastLogin(c.Context(), user.ID)

	// Generate JWT token
	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(models.AuthResponse{
		Token: token,
		User:  user,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *fiber.Ctx) error {
	// In a JWT-based system, logout is typically handled client-side
	// by removing the token. However, we can also invalidate sessions
	// if we're tracking them.

	// For now, just return success
	return c.JSON(fiber.Map{
		"message": "logged out successfully",
	})
}

// GetCurrentUser returns the currently authenticated user
func (h *Handler) GetCurrentUser(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			return Error(c, fiber.StatusNotFound, "user not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get user")
	}

	return Success(c, user)
}

// RefreshToken generates a new JWT token
func (h *Handler) RefreshToken(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	token, err := h.generateToken(user)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate token")
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}

// generateToken creates a new JWT token for a user
func (h *Handler) generateToken(user *models.User) (string, error) {
	claims := &middleware.JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.cfg.JWTExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.Email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

// VerifyEmail handles email verification
func (h *Handler) VerifyEmail(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return Error(c, fiber.StatusBadRequest, "verification token is required")
	}

	// Get the verification token
	evt, err := h.db.GetEmailVerificationToken(c.Context(), token)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid or expired verification token")
	}

	// Check if token is expired
	if time.Now().After(evt.ExpiresAt) {
		return Error(c, fiber.StatusBadRequest, "verification token has expired")
	}

	// Check if token was already used
	if evt.UsedAt != nil {
		return Error(c, fiber.StatusBadRequest, "verification token has already been used")
	}

	// Mark token as used
	if err := h.db.MarkEmailVerificationTokenUsed(c.Context(), token); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to verify email")
	}

	// Set user email as verified
	if err := h.db.SetUserEmailVerified(c.Context(), evt.UserID, true); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to verify email")
	}

	return Success(c, fiber.Map{
		"message": "Email verified successfully",
	})
}

// ResendVerificationEmail sends a new verification email
func (h *Handler) ResendVerificationEmail(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get user
	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get user")
	}

	// Check if already verified
	if user.EmailVerified {
		return Error(c, fiber.StatusBadRequest, "email is already verified")
	}

	// Check if email service is configured
	if !h.emailService.IsConfiguredWithContext(c.Context()) {
		return Error(c, fiber.StatusServiceUnavailable, "email service is not configured")
	}

	// Generate new verification token
	verifyToken, err := generateSecureToken()
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate verification token")
	}

	// Token expires in 24 hours
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = h.db.CreateEmailVerificationToken(c.Context(), user.ID, verifyToken, expiresAt)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create verification token")
	}

	// Get the base URL from the request
	scheme := "https"
	if c.Protocol() == "http" {
		scheme = "http"
	}
	baseURL := scheme + "://" + c.Hostname()
	verifyURL := baseURL + "/verify-email"

	// Send verification email
	if err := h.emailService.SendEmailVerificationEmail(user.Email, verifyToken, verifyURL); err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to send verification email")
	}

	return Success(c, fiber.Map{
		"message": "Verification email sent",
	})
}

// GetEmailVerificationStatus returns the current user's email verification status
func (h *Handler) GetEmailVerificationStatus(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get user")
	}

	requireVerification := h.isEmailVerificationRequired(c)

	return Success(c, fiber.Map{
		"email_verified":             user.EmailVerified,
		"verification_required":      requireVerification,
		"email_service_configured":   h.emailService.IsConfiguredWithContext(c.Context()),
	})
}
