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
	ErrRegionNotFound = errors.New("region not found")
	ErrRegionExists   = errors.New("region already exists")
)

// ListRegions returns a paginated list of regions with optional filtering
func (db *DB) ListRegions(ctx context.Context, params *models.RegionListParams) ([]*models.RegionWithStats, int, error) {
	// Build the WHERE clause
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if params.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(LOWER(name) LIKE LOWER($%d) OR state ILIKE $%d)",
			argIndex, argIndex,
		))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.State != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("state = $%d", argIndex))
		args = append(args, strings.ToUpper(params.State))
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM regions %s", whereClause)
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get regions with stats
	query := fmt.Sprintf(`
		SELECT
			r.id, r.name, r.state, r.zip_codes, r.created_at, r.updated_at,
			COALESCE((SELECT COUNT(*) FROM stores WHERE region_id = r.id), 0) as store_count,
			COALESCE((SELECT COUNT(*) FROM users WHERE region_id = r.id), 0) as user_count,
			COALESCE((SELECT COUNT(*) FROM store_prices sp
				JOIN stores s ON sp.store_id = s.id
				WHERE s.region_id = r.id), 0) as price_count
		FROM regions r
		%s
		ORDER BY r.state ASC, r.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var regions []*models.RegionWithStats
	for rows.Next() {
		r := &models.RegionWithStats{}
		err := rows.Scan(
			&r.ID,
			&r.Name,
			&r.State,
			&r.ZipCodes,
			&r.CreatedAt,
			&r.UpdatedAt,
			&r.StoreCount,
			&r.UserCount,
			&r.PriceCount,
		)
		if err != nil {
			return nil, 0, err
		}
		regions = append(regions, r)
	}

	return regions, total, nil
}

// GetRegionByID retrieves a region by ID with stats
func (db *DB) GetRegionByID(ctx context.Context, id int) (*models.RegionWithStats, error) {
	r := &models.RegionWithStats{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			r.id, r.name, r.state, r.zip_codes, r.created_at, r.updated_at,
			COALESCE((SELECT COUNT(*) FROM stores WHERE region_id = r.id), 0) as store_count,
			COALESCE((SELECT COUNT(*) FROM users WHERE region_id = r.id), 0) as user_count,
			COALESCE((SELECT COUNT(*) FROM store_prices sp
				JOIN stores s ON sp.store_id = s.id
				WHERE s.region_id = r.id), 0) as price_count
		FROM regions r
		WHERE r.id = $1
	`, id).Scan(
		&r.ID,
		&r.Name,
		&r.State,
		&r.ZipCodes,
		&r.CreatedAt,
		&r.UpdatedAt,
		&r.StoreCount,
		&r.UserCount,
		&r.PriceCount,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRegionNotFound
		}
		return nil, err
	}

	return r, nil
}

// CreateRegion creates a new region
func (db *DB) CreateRegion(ctx context.Context, req *models.CreateRegionRequest) (*models.Region, error) {
	region := &models.Region{}

	// Normalize state to uppercase
	state := strings.ToUpper(req.State)

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO regions (name, state, zip_codes, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		RETURNING id, name, state, zip_codes, created_at, updated_at
	`, req.Name, state, req.ZipCodes).Scan(
		&region.ID,
		&region.Name,
		&region.State,
		&region.ZipCodes,
		&region.CreatedAt,
		&region.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return region, nil
}

// UpdateRegion updates an existing region
func (db *DB) UpdateRegion(ctx context.Context, id int, req *models.UpdateRegionRequest) (*models.Region, error) {
	region := &models.Region{}

	// Normalize state to uppercase if provided
	var state *string
	if req.State != nil {
		upper := strings.ToUpper(*req.State)
		state = &upper
	}

	err := db.Pool.QueryRow(ctx, `
		UPDATE regions
		SET name = COALESCE($2, name),
		    state = COALESCE($3, state),
		    zip_codes = COALESCE($4, zip_codes),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, state, zip_codes, created_at, updated_at
	`, id, req.Name, state, req.ZipCodes).Scan(
		&region.ID,
		&region.Name,
		&region.State,
		&region.ZipCodes,
		&region.CreatedAt,
		&region.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRegionNotFound
		}
		return nil, err
	}

	return region, nil
}

// DeleteRegion deletes a region by ID
func (db *DB) DeleteRegion(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM regions WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrRegionNotFound
	}

	return nil
}

// GetDistinctStates returns all unique states that have regions
func (db *DB) GetDistinctStates(ctx context.Context) ([]string, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT state FROM regions ORDER BY state
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []string
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			return nil, err
		}
		states = append(states, state)
	}

	return states, nil
}

// GetRegionStats returns aggregate statistics for regions
func (db *DB) GetRegionStats(ctx context.Context) (map[string]int, error) {
	var totalRegions, totalStates, totalZips int

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_regions,
			COUNT(DISTINCT state) as total_states,
			COALESCE(SUM(array_length(zip_codes, 1)), 0) as total_zips
		FROM regions
	`).Scan(&totalRegions, &totalStates, &totalZips)

	if err != nil {
		return nil, err
	}

	return map[string]int{
		"total_regions": totalRegions,
		"total_states":  totalStates,
		"total_zips":    totalZips,
	}, nil
}

// SearchRegions performs a fuzzy search on regions
func (db *DB) SearchRegions(ctx context.Context, query string, limit int) ([]*models.Region, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, state, zip_codes, created_at, updated_at
		FROM regions
		WHERE name ILIKE $1 OR state ILIKE $1 OR $2 = ANY(zip_codes)
		ORDER BY
			CASE WHEN state = UPPER($2) THEN 0 ELSE 1 END,
			CASE WHEN name ILIKE $2 || '%' THEN 0 ELSE 1 END,
			name
		LIMIT $3
	`, "%"+query+"%", query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var regions []*models.Region
	for rows.Next() {
		r := &models.Region{}
		if err := rows.Scan(&r.ID, &r.Name, &r.State, &r.ZipCodes, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		regions = append(regions, r)
	}

	return regions, nil
}
