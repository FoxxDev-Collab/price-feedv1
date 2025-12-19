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
	ErrPriceNotFound = errors.New("price not found")
)

// ListPrices returns a paginated list of prices with optional filtering
func (db *DB) ListPrices(ctx context.Context, params *models.PriceListParams) ([]*models.StorePriceWithDetails, int, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if params.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(LOWER(i.name) LIKE LOWER($%d) OR LOWER(i.brand) LIKE LOWER($%d) OR LOWER(s.name) LIKE LOWER($%d))",
			argIndex, argIndex, argIndex,
		))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.StoreID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.store_id = $%d", argIndex))
		args = append(args, *params.StoreID)
		argIndex++
	}

	if params.ItemID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.item_id = $%d", argIndex))
		args = append(args, *params.ItemID)
		argIndex++
	}

	if params.RegionID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("s.region_id = $%d", argIndex))
		args = append(args, *params.RegionID)
		argIndex++
	}

	if params.Verified != nil {
		if *params.Verified {
			whereClauses = append(whereClauses, "sp.verified_count > 0")
		} else {
			whereClauses = append(whereClauses, "sp.verified_count = 0")
		}
	}

	if params.DateFrom != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.created_at >= $%d", argIndex))
		args = append(args, *params.DateFrom)
		argIndex++
	}

	if params.DateTo != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.created_at <= $%d", argIndex))
		args = append(args, *params.DateTo)
		argIndex++
	}

	// Filter by sharing status
	if params.IsShared != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("sp.is_shared = $%d", argIndex))
		args = append(args, *params.IsShared)
		argIndex++
	}

	// Filter by submitter (for private prices visibility)
	if params.UserID != nil {
		// Show prices: either shared OR submitted by this user
		whereClauses = append(whereClauses, fmt.Sprintf("(sp.is_shared = true OR sp.user_id = $%d)", argIndex))
		args = append(args, *params.UserID)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM store_prices sp
		JOIN items i ON sp.item_id = i.id
		JOIN stores s ON sp.store_id = s.id
		%s
	`, whereClause)
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get prices with details
	query := fmt.Sprintf(`
		SELECT
			sp.id, sp.store_id, sp.item_id, sp.price, sp.user_id, sp.is_shared,
			sp.verified_count, sp.last_verified, sp.created_at, sp.updated_at,
			i.name as item_name, i.brand as item_brand,
			s.name as store_name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, r.name as region_name,
			u.username as user_name, u.email as user_email
		FROM store_prices sp
		JOIN items i ON sp.item_id = i.id
		JOIN stores s ON sp.store_id = s.id
		LEFT JOIN regions r ON s.region_id = r.id
		LEFT JOIN users u ON sp.user_id = u.id
		%s
		ORDER BY sp.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var prices []*models.StorePriceWithDetails
	for rows.Next() {
		p := &models.StorePriceWithDetails{}
		err := rows.Scan(
			&p.ID, &p.StoreID, &p.ItemID, &p.Price, &p.UserID, &p.IsShared,
			&p.VerifiedCount, &p.LastVerified, &p.CreatedAt, &p.UpdatedAt,
			&p.ItemName, &p.ItemBrand,
			&p.StoreName, &p.StoreAddress, &p.StoreCity, &p.StoreState, &p.StoreZipCode,
			&p.RegionID, &p.RegionName,
			&p.UserName, &p.UserEmail,
		)
		if err != nil {
			return nil, 0, err
		}
		prices = append(prices, p)
	}

	return prices, total, nil
}

// GetPriceByID retrieves a price by ID with details
func (db *DB) GetPriceByID(ctx context.Context, id int) (*models.StorePriceWithDetails, error) {
	p := &models.StorePriceWithDetails{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			sp.id, sp.store_id, sp.item_id, sp.price, sp.user_id, sp.is_shared,
			sp.verified_count, sp.last_verified, sp.created_at, sp.updated_at,
			i.name as item_name, i.brand as item_brand,
			s.name as store_name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, r.name as region_name,
			u.username as user_name, u.email as user_email
		FROM store_prices sp
		JOIN items i ON sp.item_id = i.id
		JOIN stores s ON sp.store_id = s.id
		LEFT JOIN regions r ON s.region_id = r.id
		LEFT JOIN users u ON sp.user_id = u.id
		WHERE sp.id = $1
	`, id).Scan(
		&p.ID, &p.StoreID, &p.ItemID, &p.Price, &p.UserID, &p.IsShared,
		&p.VerifiedCount, &p.LastVerified, &p.CreatedAt, &p.UpdatedAt,
		&p.ItemName, &p.ItemBrand,
		&p.StoreName, &p.StoreAddress, &p.StoreCity, &p.StoreState, &p.StoreZipCode,
		&p.RegionID, &p.RegionName,
		&p.UserName, &p.UserEmail,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPriceNotFound
		}
		return nil, err
	}

	return p, nil
}

// CreatePrice creates a new price
func (db *DB) CreatePrice(ctx context.Context, req *models.CreatePriceRequest, userID *int) (*models.StorePrice, error) {
	price := &models.StorePrice{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO store_prices (store_id, item_id, price, user_id, is_shared, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, store_id, item_id, price, user_id, is_shared, verified_count, last_verified, created_at, updated_at
	`, req.StoreID, req.ItemID, req.Price, userID, req.IsShared).Scan(
		&price.ID, &price.StoreID, &price.ItemID, &price.Price, &price.UserID, &price.IsShared,
		&price.VerifiedCount, &price.LastVerified, &price.CreatedAt, &price.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return price, nil
}

// UpdatePrice updates an existing price
func (db *DB) UpdatePrice(ctx context.Context, id int, req *models.UpdatePriceRequest) (*models.StorePrice, error) {
	price := &models.StorePrice{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE store_prices
		SET price = COALESCE($2, price),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, store_id, item_id, price, user_id, is_shared, verified_count, last_verified, created_at, updated_at
	`, id, req.Price).Scan(
		&price.ID, &price.StoreID, &price.ItemID, &price.Price, &price.UserID, &price.IsShared,
		&price.VerifiedCount, &price.LastVerified, &price.CreatedAt, &price.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPriceNotFound
		}
		return nil, err
	}

	return price, nil
}

// DeletePrice deletes a price by ID
func (db *DB) DeletePrice(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM store_prices WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrPriceNotFound
	}

	return nil
}

// VerifyPrice adds a verification for a price
func (db *DB) VerifyPrice(ctx context.Context, priceID int, userID int, isAccurate bool) error {
	// Insert verification
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO price_verifications (price_id, user_id, is_accurate, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (price_id, user_id) DO UPDATE SET is_accurate = $3, created_at = NOW()
	`, priceID, userID, isAccurate)
	if err != nil {
		return err
	}

	// Update price verified count
	_, err = db.Pool.Exec(ctx, `
		UPDATE store_prices
		SET verified_count = (SELECT COUNT(*) FROM price_verifications WHERE price_id = $1 AND is_accurate = true),
		    last_verified = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`, priceID)

	return err
}

// GetPriceStats returns aggregate statistics for prices
func (db *DB) GetPriceStats(ctx context.Context) (*models.PriceStats, error) {
	var totalPrices, todayCount, weekCount, verifiedCount, flaggedCount int

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_prices,
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE) as today_count,
			COUNT(*) FILTER (WHERE created_at >= CURRENT_DATE - INTERVAL '7 days') as week_count,
			COUNT(*) FILTER (WHERE verified_count > 0) as verified_count,
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM price_verifications pv WHERE pv.price_id = store_prices.id AND pv.is_accurate = false)) as flagged_count
		FROM store_prices
	`).Scan(&totalPrices, &todayCount, &weekCount, &verifiedCount, &flaggedCount)

	if err != nil {
		return nil, err
	}

	return &models.PriceStats{
		TotalPrices:   totalPrices,
		TodayCount:    todayCount,
		WeekCount:     weekCount,
		VerifiedCount: verifiedCount,
		FlaggedCount:  flaggedCount,
	}, nil
}

// GetPricesByStore returns all prices for a store
func (db *DB) GetPricesByStore(ctx context.Context, storeID int) ([]*models.StorePriceWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sp.id, sp.store_id, sp.item_id, sp.price, sp.user_id, sp.is_shared,
			sp.verified_count, sp.last_verified, sp.created_at, sp.updated_at,
			i.name as item_name, i.brand as item_brand,
			s.name as store_name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, r.name as region_name,
			u.username as user_name, u.email as user_email
		FROM store_prices sp
		JOIN items i ON sp.item_id = i.id
		JOIN stores s ON sp.store_id = s.id
		LEFT JOIN regions r ON s.region_id = r.id
		LEFT JOIN users u ON sp.user_id = u.id
		WHERE sp.store_id = $1
		ORDER BY i.name ASC
	`, storeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*models.StorePriceWithDetails
	for rows.Next() {
		p := &models.StorePriceWithDetails{}
		err := rows.Scan(
			&p.ID, &p.StoreID, &p.ItemID, &p.Price, &p.UserID, &p.IsShared,
			&p.VerifiedCount, &p.LastVerified, &p.CreatedAt, &p.UpdatedAt,
			&p.ItemName, &p.ItemBrand,
			&p.StoreName, &p.StoreAddress, &p.StoreCity, &p.StoreState, &p.StoreZipCode,
			&p.RegionID, &p.RegionName,
			&p.UserName, &p.UserEmail,
		)
		if err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}

	return prices, nil
}

// GetPricesByItem returns all prices for an item
func (db *DB) GetPricesByItem(ctx context.Context, itemID int) ([]*models.StorePriceWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sp.id, sp.store_id, sp.item_id, sp.price, sp.user_id, sp.is_shared,
			sp.verified_count, sp.last_verified, sp.created_at, sp.updated_at,
			i.name as item_name, i.brand as item_brand,
			s.name as store_name, s.street_address, s.city, s.state, s.zip_code,
			s.region_id, r.name as region_name,
			u.username as user_name, u.email as user_email
		FROM store_prices sp
		JOIN items i ON sp.item_id = i.id
		JOIN stores s ON sp.store_id = s.id
		LEFT JOIN regions r ON s.region_id = r.id
		LEFT JOIN users u ON sp.user_id = u.id
		WHERE sp.item_id = $1
		ORDER BY sp.price ASC
	`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*models.StorePriceWithDetails
	for rows.Next() {
		p := &models.StorePriceWithDetails{}
		err := rows.Scan(
			&p.ID, &p.StoreID, &p.ItemID, &p.Price, &p.UserID, &p.IsShared,
			&p.VerifiedCount, &p.LastVerified, &p.CreatedAt, &p.UpdatedAt,
			&p.ItemName, &p.ItemBrand,
			&p.StoreName, &p.StoreAddress, &p.StoreCity, &p.StoreState, &p.StoreZipCode,
			&p.RegionID, &p.RegionName,
			&p.UserName, &p.UserEmail,
		)
		if err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}

	return prices, nil
}
