package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/foxxcyber/price-feed/internal/models"
)

var (
	ErrStoreNotFound = errors.New("store not found")
	ErrStoreExists   = errors.New("store already exists at this address")
)

// ListStores returns a paginated list of stores with optional filtering
func (db *DB) ListStores(ctx context.Context, params *models.StoreListParams) ([]*models.StoreWithStats, int, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if params.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(LOWER(s.name) LIKE LOWER($%d) OR LOWER(s.street_address) LIKE LOWER($%d) OR LOWER(s.chain) LIKE LOWER($%d))",
			argIndex, argIndex, argIndex,
		))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.RegionID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.region_id = $%d", argIndex))
		args = append(args, *params.RegionID)
		argIndex++
	}

	if params.State != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("s.state = $%d", argIndex))
		args = append(args, strings.ToUpper(params.State))
		argIndex++
	}

	if params.Verified != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.verified = $%d", argIndex))
		args = append(args, *params.Verified)
		argIndex++
	}

	// Filter by privacy
	if params.IsPrivate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.is_private = $%d", argIndex))
		args = append(args, *params.IsPrivate)
		argIndex++
	}

	// Filter by creator (for private stores)
	if params.UserID != nil {
		// Show stores: either public OR (private AND created by this user)
		whereClauses = append(whereClauses, fmt.Sprintf("(s.is_private = false OR s.created_by = $%d)", argIndex))
		args = append(args, *params.UserID)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM stores s %s", whereClause)
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get stores with stats
	query := fmt.Sprintf(`
		SELECT
			s.id, s.name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, s.store_type, s.chain, s.latitude, s.longitude,
			s.verified, s.verification_count, s.is_private, s.created_by, s.created_at, s.updated_at,
			r.name as region_name,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE store_id = s.id), 0) as price_count,
			COALESCE((SELECT COUNT(DISTINCT user_id) FROM store_prices WHERE store_id = s.id AND user_id IS NOT NULL), 0) as contributor_count
		FROM stores s
		LEFT JOIN regions r ON s.region_id = r.id
		%s
		ORDER BY s.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var stores []*models.StoreWithStats
	for rows.Next() {
		s := &models.StoreWithStats{}
		err := rows.Scan(
			&s.ID, &s.Name, &s.StreetAddress, &s.City, &s.State, &s.ZipCode,
			&s.RegionID, &s.StoreType, &s.Chain, &s.Latitude, &s.Longitude,
			&s.Verified, &s.VerificationCount, &s.IsPrivate, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
			&s.RegionName,
			&s.PriceCount,
			&s.ContributorCount,
		)
		if err != nil {
			return nil, 0, err
		}
		stores = append(stores, s)
	}

	return stores, total, nil
}

// GetStoreByID retrieves a store by ID with stats
func (db *DB) GetStoreByID(ctx context.Context, id int) (*models.StoreWithStats, error) {
	s := &models.StoreWithStats{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			s.id, s.name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, s.store_type, s.chain, s.latitude, s.longitude,
			s.verified, s.verification_count, s.is_private, s.created_by, s.created_at, s.updated_at,
			r.name as region_name,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE store_id = s.id), 0) as price_count,
			COALESCE((SELECT COUNT(DISTINCT user_id) FROM store_prices WHERE store_id = s.id AND user_id IS NOT NULL), 0) as contributor_count
		FROM stores s
		LEFT JOIN regions r ON s.region_id = r.id
		WHERE s.id = $1
	`, id).Scan(
		&s.ID, &s.Name, &s.StreetAddress, &s.City, &s.State, &s.ZipCode,
		&s.RegionID, &s.StoreType, &s.Chain, &s.Latitude, &s.Longitude,
		&s.Verified, &s.VerificationCount, &s.IsPrivate, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
		&s.RegionName,
		&s.PriceCount,
		&s.ContributorCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}

	return s, nil
}

// CreateStore creates a new store
func (db *DB) CreateStore(ctx context.Context, req *models.CreateStoreRequest, createdBy *int) (*models.Store, error) {
	store := &models.Store{}

	// Normalize state to uppercase
	state := strings.ToUpper(req.State)

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO stores (name, street_address, city, state, zip_code, region_id, store_type, chain, latitude, longitude, verified, is_private, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
		RETURNING id, name, street_address, city, state, zip_code, region_id, store_type, chain, latitude, longitude, verified, verification_count, is_private, created_by, created_at, updated_at
	`, req.Name, req.StreetAddress, req.City, state, req.ZipCode, req.RegionID, req.StoreType, req.Chain, req.Latitude, req.Longitude, req.Verified, req.IsPrivate, createdBy).Scan(
		&store.ID, &store.Name, &store.StreetAddress, &store.City, &store.State, &store.ZipCode,
		&store.RegionID, &store.StoreType, &store.Chain, &store.Latitude, &store.Longitude,
		&store.Verified, &store.VerificationCount, &store.IsPrivate, &store.CreatedBy, &store.CreatedAt, &store.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return store, nil
}

// UpdateStore updates an existing store
func (db *DB) UpdateStore(ctx context.Context, id int, req *models.UpdateStoreRequest) (*models.Store, error) {
	store := &models.Store{}

	// Normalize state to uppercase if provided
	var state *string
	if req.State != nil {
		upper := strings.ToUpper(*req.State)
		state = &upper
	}

	err := db.Pool.QueryRow(ctx, `
		UPDATE stores
		SET name = COALESCE($2, name),
		    street_address = COALESCE($3, street_address),
		    city = COALESCE($4, city),
		    state = COALESCE($5, state),
		    zip_code = COALESCE($6, zip_code),
		    region_id = COALESCE($7, region_id),
		    store_type = COALESCE($8, store_type),
		    chain = COALESCE($9, chain),
		    latitude = COALESCE($10, latitude),
		    longitude = COALESCE($11, longitude),
		    verified = COALESCE($12, verified),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, street_address, city, state, zip_code, region_id, store_type, chain, latitude, longitude, verified, verification_count, is_private, created_by, created_at, updated_at
	`, id, req.Name, req.StreetAddress, req.City, state, req.ZipCode, req.RegionID, req.StoreType, req.Chain, req.Latitude, req.Longitude, req.Verified).Scan(
		&store.ID, &store.Name, &store.StreetAddress, &store.City, &store.State, &store.ZipCode,
		&store.RegionID, &store.StoreType, &store.Chain, &store.Latitude, &store.Longitude,
		&store.Verified, &store.VerificationCount, &store.IsPrivate, &store.CreatedBy, &store.CreatedAt, &store.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}

	return store, nil
}

// DeleteStore deletes a store by ID
func (db *DB) DeleteStore(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM stores WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrStoreNotFound
	}

	return nil
}

// VerifyStore marks a store as verified
func (db *DB) VerifyStore(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `
		UPDATE stores
		SET verified = true, verification_count = verification_count + 1, updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrStoreNotFound
	}

	return nil
}

// GetStoreStats returns aggregate statistics for stores
func (db *DB) GetStoreStats(ctx context.Context) (*models.StoreStats, error) {
	var totalStores, verifiedCount, pendingCount, totalPrices int

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_stores,
			COUNT(*) FILTER (WHERE verified = true) as verified_count,
			COUNT(*) FILTER (WHERE verified = false) as pending_count,
			COALESCE((SELECT COUNT(*) FROM store_prices), 0) as total_prices
		FROM stores
	`).Scan(&totalStores, &verifiedCount, &pendingCount, &totalPrices)

	if err != nil {
		return nil, err
	}

	return &models.StoreStats{
		TotalStores:   totalStores,
		VerifiedCount: verifiedCount,
		PendingCount:  pendingCount,
		TotalPrices:   totalPrices,
	}, nil
}

// SearchStores performs a fuzzy search on stores
// Only returns stores visible to the user (public stores OR user's own private stores)
func (db *DB) SearchStores(ctx context.Context, query string, limit int, userID *int) ([]*models.Store, error) {
	var rows pgx.Rows
	var err error

	if userID != nil {
		// User is logged in: show public stores OR their own private stores
		rows, err = db.Pool.Query(ctx, `
			SELECT id, name, street_address, city, state, zip_code, region_id, store_type, chain, latitude, longitude, verified, verification_count, is_private, created_by, created_at, updated_at
			FROM stores
			WHERE (name ILIKE $1 OR street_address ILIKE $1 OR chain ILIKE $1 OR zip_code = $2)
			AND (is_private = false OR created_by = $4)
			ORDER BY
				CASE WHEN name ILIKE $2 || '%' THEN 0 ELSE 1 END,
				name
			LIMIT $3
		`, "%"+query+"%", query, limit, *userID)
	} else {
		// No user: show only public stores
		rows, err = db.Pool.Query(ctx, `
			SELECT id, name, street_address, city, state, zip_code, region_id, store_type, chain, latitude, longitude, verified, verification_count, is_private, created_by, created_at, updated_at
			FROM stores
			WHERE (name ILIKE $1 OR street_address ILIKE $1 OR chain ILIKE $1 OR zip_code = $2)
			AND is_private = false
			ORDER BY
				CASE WHEN name ILIKE $2 || '%' THEN 0 ELSE 1 END,
				name
			LIMIT $3
		`, "%"+query+"%", query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []*models.Store
	for rows.Next() {
		s := &models.Store{}
		if err := rows.Scan(&s.ID, &s.Name, &s.StreetAddress, &s.City, &s.State, &s.ZipCode,
			&s.RegionID, &s.StoreType, &s.Chain, &s.Latitude, &s.Longitude,
			&s.Verified, &s.VerificationCount, &s.IsPrivate, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}

	return stores, nil
}

// StoreWithDistance represents a store with its distance from a reference point
type StoreWithDistance struct {
	models.StoreWithStats
	DistanceKm float64 `json:"distance_km"`
}

// FindNearbyStores finds public stores within a given radius of a location
// Uses the Haversine formula to calculate distance
// Only returns public stores (is_private = false) that have coordinates set
func (db *DB) FindNearbyStores(ctx context.Context, lat, lng float64, radiusKm float64, limit int) ([]*StoreWithDistance, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Haversine formula to calculate distance in kilometers
	// 6371 is Earth's radius in km
	rows, err := db.Pool.Query(ctx, `
		SELECT
			s.id, s.name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, s.store_type, s.chain, s.latitude, s.longitude,
			s.verified, s.verification_count, s.is_private, s.created_by, s.created_at, s.updated_at,
			r.name as region_name,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE store_id = s.id), 0) as price_count,
			COALESCE((SELECT COUNT(DISTINCT user_id) FROM store_prices WHERE store_id = s.id AND user_id IS NOT NULL), 0) as contributor_count,
			(
				6371 * acos(
					LEAST(1.0, GREATEST(-1.0,
						cos(radians($1)) * cos(radians(s.latitude)) *
						cos(radians(s.longitude) - radians($2)) +
						sin(radians($1)) * sin(radians(s.latitude))
					))
				)
			) as distance_km
		FROM stores s
		LEFT JOIN regions r ON s.region_id = r.id
		WHERE s.is_private = false
			AND s.latitude IS NOT NULL
			AND s.longitude IS NOT NULL
			AND (
				6371 * acos(
					LEAST(1.0, GREATEST(-1.0,
						cos(radians($1)) * cos(radians(s.latitude)) *
						cos(radians(s.longitude) - radians($2)) +
						sin(radians($1)) * sin(radians(s.latitude))
					))
				)
			) <= $3
		ORDER BY distance_km ASC
		LIMIT $4
	`, lat, lng, radiusKm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stores []*StoreWithDistance
	for rows.Next() {
		s := &StoreWithDistance{}
		err := rows.Scan(
			&s.ID, &s.Name, &s.StreetAddress, &s.City, &s.State, &s.ZipCode,
			&s.RegionID, &s.StoreType, &s.Chain, &s.Latitude, &s.Longitude,
			&s.Verified, &s.VerificationCount, &s.IsPrivate, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt,
			&s.RegionName,
			&s.PriceCount,
			&s.ContributorCount,
			&s.DistanceKm,
		)
		if err != nil {
			return nil, err
		}
		stores = append(stores, s)
	}

	return stores, nil
}
