package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
)

// Handler holds all handler dependencies
type Handler struct {
	db  *database.DB
	cfg *config.Config
}

// New creates a new Handler instance
func New(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
	}
}

// ErrorHandler is a custom error handler for Fiber
func ErrorHandler(c *fiber.Ctx, err error) error {
	// Default to 500
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error": message,
	})
}

// APIResponse is a standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Success returns a successful response
func Success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(APIResponse{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta returns a successful response with pagination
func SuccessWithMeta(c *fiber.Ctx, data interface{}, total, limit, offset int) error {
	return c.JSON(APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	})
}

// Error returns an error response
func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(APIResponse{
		Success: false,
		Error:   message,
	})
}
