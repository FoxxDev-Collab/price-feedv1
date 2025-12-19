package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/models"
)

// JWTClaims represents the claims in our JWT token
type JWTClaims struct {
	UserID int         `json:"user_id"`
	Email  string      `json:"email"`
	Role   models.Role `json:"role"`
	jwt.RegisteredClaims
}

// AuthRequired middleware checks for a valid JWT token
func AuthRequired(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// Check for Bearer prefix
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization format",
			})
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		// Extract claims
		claims, ok := token.Claims.(*JWTClaims)
		if !ok || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		// Store user info in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)

		return c.Next()
	}
}

// AdminRequired middleware checks if the user has admin role
func AdminRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("user_role").(models.Role)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		if role != models.RoleAdmin {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "admin access required",
			})
		}

		return c.Next()
	}
}

// ModeratorRequired middleware checks if the user has moderator or admin role
func ModeratorRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("user_role").(models.Role)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		if role != models.RoleAdmin && role != models.RoleModerator {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "moderator access required",
			})
		}

		return c.Next()
	}
}

// GetUserID extracts the user ID from the context
func GetUserID(c *fiber.Ctx) int {
	if id, ok := c.Locals("user_id").(int); ok {
		return id
	}
	return 0
}

// GetUserRole extracts the user role from the context
func GetUserRole(c *fiber.Ctx) models.Role {
	if role, ok := c.Locals("user_role").(models.Role); ok {
		return role
	}
	return models.RoleUser
}

// GetUserEmail extracts the user email from the context
func GetUserEmail(c *fiber.Ctx) string {
	if email, ok := c.Locals("user_email").(string); ok {
		return email
	}
	return ""
}
