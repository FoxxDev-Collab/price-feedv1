package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/foxxcyber/price-feed/internal/models"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// CreateUser creates a new user in the database
func (db *DB) CreateUser(ctx context.Context, email, passwordHash string, username *string, regionID *int, req *models.RegisterRequest) (*models.User, error) {
	user := &models.User{}

	// Extract location fields from request (if provided)
	var streetAddress, city, state, zipCode, googlePlaceID *string
	var latitude, longitude *float64
	if req != nil {
		streetAddress = req.StreetAddress
		city = req.City
		state = req.State
		zipCode = req.ZipCode
		latitude = req.Latitude
		longitude = req.Longitude
		googlePlaceID = req.GooglePlaceID
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, username, region_id, street_address, city, state, zip_code, latitude, longitude, google_place_id, role, email_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 'user', false, NOW(), NOW())
		RETURNING id, email, password_hash, username, region_id, reputation_points, role, email_verified, created_at, updated_at, last_login_at,
			street_address, city, state, zip_code, latitude, longitude, google_place_id
	`, email, passwordHash, username, regionID, streetAddress, city, state, zipCode, latitude, longitude, googlePlaceID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.RegionID,
		&user.ReputationPoints,
		&user.Role,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.StreetAddress,
		&user.City,
		&user.State,
		&user.ZipCode,
		&user.Latitude,
		&user.Longitude,
		&user.GooglePlaceID,
	)

	if err != nil {
		// Check for unique constraint violations
		if err.Error() == `ERROR: duplicate key value violates unique constraint "users_email_key" (SQLSTATE 23505)` {
			return nil, ErrEmailExists
		}
		if err.Error() == `ERROR: duplicate key value violates unique constraint "users_username_key" (SQLSTATE 23505)` {
			return nil, ErrUsernameExists
		}
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.password_hash, u.username, u.region_id, r.name as region_name, u.reputation_points, u.role, u.email_verified, u.created_at, u.updated_at, u.last_login_at,
			u.street_address, u.city, u.state, u.zip_code, u.latitude, u.longitude, u.google_place_id
		FROM users u
		LEFT JOIN regions r ON u.region_id = r.id
		WHERE u.id = $1
	`, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.RegionID,
		&user.RegionName,
		&user.ReputationPoints,
		&user.Role,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.StreetAddress,
		&user.City,
		&user.State,
		&user.ZipCode,
		&user.Latitude,
		&user.Longitude,
		&user.GooglePlaceID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by their email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, username, region_id, reputation_points, role, email_verified, created_at, updated_at, last_login_at,
			street_address, city, state, zip_code, latitude, longitude, google_place_id
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.RegionID,
		&user.ReputationPoints,
		&user.Role,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.StreetAddress,
		&user.City,
		&user.State,
		&user.ZipCode,
		&user.Latitude,
		&user.Longitude,
		&user.GooglePlaceID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// UpdateUser updates a user's profile
func (db *DB) UpdateUser(ctx context.Context, id int, req *models.UpdateUserRequest) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE users
		SET username = COALESCE($2, username),
		    region_id = COALESCE($3, region_id),
		    street_address = COALESCE($4, street_address),
		    city = COALESCE($5, city),
		    state = COALESCE($6, state),
		    zip_code = COALESCE($7, zip_code),
		    latitude = COALESCE($8, latitude),
		    longitude = COALESCE($9, longitude),
		    google_place_id = COALESCE($10, google_place_id),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, username, region_id, reputation_points, role, email_verified, created_at, updated_at, last_login_at,
			street_address, city, state, zip_code, latitude, longitude, google_place_id
	`, id, req.Username, req.RegionID, req.StreetAddress, req.City, req.State, req.ZipCode, req.Latitude, req.Longitude, req.GooglePlaceID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.RegionID,
		&user.ReputationPoints,
		&user.Role,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.StreetAddress,
		&user.City,
		&user.State,
		&user.ZipCode,
		&user.Latitude,
		&user.Longitude,
		&user.GooglePlaceID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// UpdateUserLastLogin updates the user's last login timestamp
func (db *DB) UpdateUserLastLogin(ctx context.Context, id int) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, id)
	return err
}

// UpdateUserPassword updates a user's password
func (db *DB) UpdateUserPassword(ctx context.Context, id int, newPasswordHash string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, id, newPasswordHash)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// AdminUpdateUser updates a user with admin privileges
func (db *DB) AdminUpdateUser(ctx context.Context, id int, req *models.AdminUpdateUserRequest) (*models.User, error) {
	user := &models.User{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE users
		SET email = COALESCE($2, email),
		    username = COALESCE($3, username),
		    role = COALESCE($4, role),
		    email_verified = COALESCE($5, email_verified),
		    region_id = COALESCE($6, region_id),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, email, password_hash, username, region_id, reputation_points, role, email_verified, created_at, updated_at, last_login_at,
			street_address, city, state, zip_code, latitude, longitude, google_place_id
	`, id, req.Email, req.Username, req.Role, req.EmailVerified, req.RegionID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Username,
		&user.RegionID,
		&user.ReputationPoints,
		&user.Role,
		&user.EmailVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.StreetAddress,
		&user.City,
		&user.State,
		&user.ZipCode,
		&user.Latitude,
		&user.Longitude,
		&user.GooglePlaceID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

// DeleteUser deletes a user by ID
func (db *DB) DeleteUser(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ListUsers returns a paginated list of users
func (db *DB) ListUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get users
	rows, err := db.Pool.Query(ctx, `
		SELECT id, email, password_hash, username, region_id, reputation_points, role, email_verified, created_at, updated_at, last_login_at,
			street_address, city, state, zip_code, latitude, longitude, google_place_id
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Username,
			&user.RegionID,
			&user.ReputationPoints,
			&user.Role,
			&user.EmailVerified,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.LastLoginAt,
			&user.StreetAddress,
			&user.City,
			&user.State,
			&user.ZipCode,
			&user.Latitude,
			&user.Longitude,
			&user.GooglePlaceID,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, total, nil
}

// GetUserStats retrieves statistics for a user
func (db *DB) GetUserStats(ctx context.Context, userID int) (*models.UserStats, error) {
	stats := &models.UserStats{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM stores WHERE created_by = $1), 0) as stores_added,
			COALESCE((SELECT COUNT(*) FROM items WHERE created_by = $1), 0) as items_added,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE user_id = $1), 0) as prices_reported,
			COALESCE((SELECT COUNT(*) FROM price_verifications WHERE user_id = $1), 0) as verifications,
			COALESCE((SELECT COUNT(*) FROM shopping_lists WHERE user_id = $1), 0) as lists_created
	`, userID).Scan(
		&stats.StoresAdded,
		&stats.ItemsAdded,
		&stats.PricesReported,
		&stats.Verifications,
		&stats.ListsCreated,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetAdminStats retrieves system-wide statistics
func (db *DB) GetAdminStats(ctx context.Context) (*models.AdminStats, error) {
	stats := &models.AdminStats{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM users), 0) as total_users,
			COALESCE((SELECT COUNT(*) FROM users WHERE last_login_at > NOW() - INTERVAL '24 hours'), 0) as active_users_24h,
			COALESCE((SELECT COUNT(*) FROM stores), 0) as total_stores,
			COALESCE((SELECT COUNT(*) FROM stores WHERE verified = true), 0) as verified_stores,
			COALESCE((SELECT COUNT(*) FROM items), 0) as total_items,
			COALESCE((SELECT COUNT(*) FROM store_prices), 0) as total_prices,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE created_at > NOW() - INTERVAL '24 hours'), 0) as prices_today
	`).Scan(
		&stats.TotalUsers,
		&stats.ActiveUsers24h,
		&stats.TotalStores,
		&stats.VerifiedStores,
		&stats.TotalItems,
		&stats.TotalPrices,
		&stats.PricesToday,
	)

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// CreateSession creates a new user session
func (db *DB) CreateSession(ctx context.Context, userID int, token string, expiresAt time.Time) (*models.Session, error) {
	session := &models.Session{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO user_sessions (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token, expires_at, created_at
	`, userID, token, expiresAt).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes a user session
func (db *DB) DeleteSession(ctx context.Context, token string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM user_sessions WHERE token = $1`, token)
	return err
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(ctx context.Context, userID int) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (db *DB) CleanupExpiredSessions(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM user_sessions WHERE expires_at < NOW()`)
	return err
}

// EmailVerificationToken represents a token for email verification
type EmailVerificationToken struct {
	ID        int
	UserID    int
	Token     string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// CreateEmailVerificationToken creates a new email verification token
func (db *DB) CreateEmailVerificationToken(ctx context.Context, userID int, token string, expiresAt time.Time) (*EmailVerificationToken, error) {
	// Delete any existing unused tokens for this user
	_, _ = db.Pool.Exec(ctx, `DELETE FROM email_verification_tokens WHERE user_id = $1 AND used_at IS NULL`, userID)

	evt := &EmailVerificationToken{}
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO email_verification_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token, expires_at, used_at, created_at
	`, userID, token, expiresAt).Scan(
		&evt.ID,
		&evt.UserID,
		&evt.Token,
		&evt.ExpiresAt,
		&evt.UsedAt,
		&evt.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return evt, nil
}

// GetEmailVerificationToken retrieves a verification token by its token string
func (db *DB) GetEmailVerificationToken(ctx context.Context, token string) (*EmailVerificationToken, error) {
	evt := &EmailVerificationToken{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM email_verification_tokens
		WHERE token = $1
	`, token).Scan(
		&evt.ID,
		&evt.UserID,
		&evt.Token,
		&evt.ExpiresAt,
		&evt.UsedAt,
		&evt.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return evt, nil
}

// MarkEmailVerificationTokenUsed marks a verification token as used
func (db *DB) MarkEmailVerificationTokenUsed(ctx context.Context, token string) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE email_verification_tokens SET used_at = NOW() WHERE token = $1 AND used_at IS NULL
	`, token)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errors.New("token not found or already used")
	}
	return nil
}

// SetUserEmailVerified sets the email_verified flag for a user
func (db *DB) SetUserEmailVerified(ctx context.Context, userID int, verified bool) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE users SET email_verified = $2, updated_at = NOW() WHERE id = $1
	`, userID, verified)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// CleanupExpiredVerificationTokens removes expired verification tokens
func (db *DB) CleanupExpiredVerificationTokens(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM email_verification_tokens WHERE expires_at < NOW()`)
	return err
}
