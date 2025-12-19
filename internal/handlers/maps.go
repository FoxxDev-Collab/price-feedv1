package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/services"
)

// MapsHandler handles Google Maps related endpoints
type MapsHandler struct {
	mapsService *services.GoogleMapsService
	frontendKey string
}

// NewMapsHandler creates a new MapsHandler instance
func NewMapsHandler(mapsService *services.GoogleMapsService, frontendKey string) *MapsHandler {
	return &MapsHandler{
		mapsService: mapsService,
		frontendKey: frontendKey,
	}
}

// GeocodeRequest is the request body for geocoding
type GeocodeRequest struct {
	Address string `json:"address"`
}

// ReverseGeocodeRequest is the request body for reverse geocoding
type ReverseGeocodeRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// NearbyStoresRequest is the request body for nearby stores search
type NearbyStoresRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    int     `json:"radius"` // in meters, optional
}

// MapsConfigResponse is the response for the config endpoint
type MapsConfigResponse struct {
	FrontendKey string `json:"frontend_key"`
}

// Geocode converts an address to coordinates
// POST /api/maps/geocode
func (h *MapsHandler) Geocode(c *fiber.Ctx) error {
	var req GeocodeRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Address == "" {
		return Error(c, fiber.StatusBadRequest, "address is required")
	}

	result, err := h.mapsService.Geocode(c.Context(), req.Address)
	if err != nil {
		return handleMapsError(c, err)
	}

	return Success(c, result)
}

// ReverseGeocode converts coordinates to an address
// POST /api/maps/reverse-geocode
func (h *MapsHandler) ReverseGeocode(c *fiber.Ctx) error {
	var req ReverseGeocodeRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 {
		return Error(c, fiber.StatusBadRequest, "latitude must be between -90 and 90")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return Error(c, fiber.StatusBadRequest, "longitude must be between -180 and 180")
	}

	result, err := h.mapsService.ReverseGeocode(c.Context(), req.Latitude, req.Longitude)
	if err != nil {
		return handleMapsError(c, err)
	}

	return Success(c, result)
}

// NearbyStores searches for grocery stores near a location
// POST /api/maps/nearby-stores
func (h *MapsHandler) NearbyStores(c *fiber.Ctx) error {
	var req NearbyStoresRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 {
		return Error(c, fiber.StatusBadRequest, "latitude must be between -90 and 90")
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		return Error(c, fiber.StatusBadRequest, "longitude must be between -180 and 180")
	}

	// Default radius to 5km if not provided
	radius := req.Radius
	if radius <= 0 {
		radius = 5000
	}
	// Cap radius at 50km
	if radius > 50000 {
		radius = 50000
	}

	// Search for supermarkets/grocery stores
	results, err := h.mapsService.NearbySearch(c.Context(), req.Latitude, req.Longitude, radius, "supermarket")
	if err != nil {
		return handleMapsError(c, err)
	}

	return Success(c, results)
}

// GetPlaceDetails retrieves detailed information about a place
// GET /api/maps/place/:place_id
func (h *MapsHandler) GetPlaceDetails(c *fiber.Ctx) error {
	placeID := c.Params("place_id")
	if placeID == "" {
		return Error(c, fiber.StatusBadRequest, "place_id is required")
	}

	details, err := h.mapsService.GetPlaceDetails(c.Context(), placeID)
	if err != nil {
		return handleMapsError(c, err)
	}

	return Success(c, details)
}

// GetConfig returns the frontend API key configuration
// GET /api/maps/config
func (h *MapsHandler) GetConfig(c *fiber.Ctx) error {
	return Success(c, MapsConfigResponse{
		FrontendKey: h.frontendKey,
	})
}

// handleMapsError converts Google Maps service errors to HTTP responses
func handleMapsError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, services.ErrNoResults):
		return Error(c, fiber.StatusNotFound, "no results found for the given location")
	case errors.Is(err, services.ErrInvalidAPIKey):
		return Error(c, fiber.StatusServiceUnavailable, "maps service is not configured")
	case errors.Is(err, services.ErrRequestDenied):
		return Error(c, fiber.StatusForbidden, "maps request was denied")
	case errors.Is(err, services.ErrOverQueryLimit):
		return Error(c, fiber.StatusTooManyRequests, "maps api quota exceeded")
	case errors.Is(err, services.ErrInvalidRequest):
		return Error(c, fiber.StatusBadRequest, "invalid maps request")
	default:
		return Error(c, fiber.StatusInternalServerError, "failed to process maps request")
	}
}
