package models

import (
	"time"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAdmin     Role = "admin"
	RoleModerator Role = "moderator"
)

type User struct {
	ID               int        `json:"id"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"` // Never expose in JSON
	Username         *string    `json:"username,omitempty"`
	RegionID         *int       `json:"region_id,omitempty"`
	ReputationPoints int        `json:"reputation_points"`
	Role             Role       `json:"role"`
	EmailVerified    bool       `json:"email_verified"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	// Location fields for Google Maps integration
	StreetAddress *string  `json:"street_address,omitempty"`
	City          *string  `json:"city,omitempty"`
	State         *string  `json:"state,omitempty"`
	ZipCode       *string  `json:"zip_code,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	GooglePlaceID *string  `json:"google_place_id,omitempty"`
}

// UserPublic is the public-safe representation of a user
type UserPublic struct {
	ID               int       `json:"id"`
	Username         *string   `json:"username,omitempty"`
	ReputationPoints int       `json:"reputation_points"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToPublic converts a User to its public representation
func (u *User) ToPublic() *UserPublic {
	return &UserPublic{
		ID:               u.ID,
		Username:         u.Username,
		ReputationPoints: u.ReputationPoints,
		CreatedAt:        u.CreatedAt,
	}
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsModerator checks if the user has moderator role or higher
func (u *User) IsModerator() bool {
	return u.Role == RoleModerator || u.Role == RoleAdmin
}

// Session represents a user session for token management
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest is the request body for user registration
type RegisterRequest struct {
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Username *string `json:"username,omitempty"`
	RegionID *int    `json:"region_id,omitempty"`
	// Location fields for Google Maps integration
	StreetAddress *string  `json:"street_address,omitempty"`
	City          *string  `json:"city,omitempty"`
	State         *string  `json:"state,omitempty"`
	ZipCode       *string  `json:"zip_code,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	GooglePlaceID *string  `json:"google_place_id,omitempty"`
}

// LoginRequest is the request body for user login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse is returned after successful login/register
type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// UpdateUserRequest is the request body for updating user profile
type UpdateUserRequest struct {
	Username *string `json:"username,omitempty"`
	RegionID *int    `json:"region_id,omitempty"`
	// Location fields for Google Maps integration
	StreetAddress *string  `json:"street_address,omitempty"`
	City          *string  `json:"city,omitempty"`
	State         *string  `json:"state,omitempty"`
	ZipCode       *string  `json:"zip_code,omitempty"`
	Latitude      *float64 `json:"latitude,omitempty"`
	Longitude     *float64 `json:"longitude,omitempty"`
	GooglePlaceID *string  `json:"google_place_id,omitempty"`
}

// AdminUpdateUserRequest is the request body for admin user updates
type AdminUpdateUserRequest struct {
	Email         *string `json:"email,omitempty"`
	Username      *string `json:"username,omitempty"`
	Role          *Role   `json:"role,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	RegionID      *int    `json:"region_id,omitempty"`
}

// AdminCreateUserRequest is the request body for admin user creation
type AdminCreateUserRequest struct {
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	Username      *string `json:"username,omitempty"`
	Role          Role    `json:"role"`
	EmailVerified bool    `json:"email_verified"`
	RegionID      *int    `json:"region_id,omitempty"`
}

// UserStats represents aggregated user statistics
type UserStats struct {
	StoresAdded    int `json:"stores_added"`
	ItemsAdded     int `json:"items_added"`
	PricesReported int `json:"prices_reported"`
	Verifications  int `json:"verifications"`
	ListsCreated   int `json:"lists_created"`
}

// AdminStats represents system-wide statistics
type AdminStats struct {
	TotalUsers     int `json:"total_users"`
	ActiveUsers24h int `json:"active_users_24h"`
	TotalStores    int `json:"total_stores"`
	VerifiedStores int `json:"verified_stores"`
	TotalItems     int `json:"total_items"`
	TotalPrices    int `json:"total_prices"`
	PricesToday    int `json:"prices_today"`
}
