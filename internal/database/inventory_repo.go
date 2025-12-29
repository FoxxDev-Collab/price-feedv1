package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/foxxcyber/price-feed/internal/models"
)

var (
	ErrInventoryItemNotFound = errors.New("inventory item not found")
	ErrNotInventoryOwner     = errors.New("not the owner of this inventory item")
)

// ListInventoryItems returns paginated inventory for a user
func (db *DB) ListInventoryItems(ctx context.Context, params *models.InventoryListParams) ([]*models.InventoryItemWithDetails, int, error) {
	// Build where clauses
	whereClauses := []string{"ii.user_id = $1"}
	args := []interface{}{params.UserID}
	argCount := 1

	if params.Location != "" {
		argCount++
		whereClauses = append(whereClauses, fmt.Sprintf("ii.location = $%d", argCount))
		args = append(args, params.Location)
	}

	if params.Search != "" {
		argCount++
		searchPattern := "%" + strings.ToLower(params.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(LOWER(COALESCE(i.name, ii.custom_name, '')) LIKE $%d OR LOWER(COALESCE(i.brand, ii.custom_brand, '')) LIKE $%d)", argCount, argCount))
		args = append(args, searchPattern)
	}

	if params.LowStock != nil && *params.LowStock {
		whereClauses = append(whereClauses, "ii.low_stock_alert_enabled = true AND ii.quantity <= ii.low_stock_threshold")
	}

	if params.Expired != nil && *params.Expired {
		whereClauses = append(whereClauses, "ii.expiration_date IS NOT NULL AND ii.expiration_date < CURRENT_DATE")
	}

	if params.ExpiringSoon != nil && *params.ExpiringSoon {
		whereClauses = append(whereClauses, "ii.expiration_date IS NOT NULL AND ii.expiration_date >= CURRENT_DATE AND ii.expiration_date <= CURRENT_DATE + INTERVAL '7 days'")
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Determine sort order
	sortColumn := "ii.updated_at"
	sortOrder := "DESC"
	switch params.SortBy {
	case "name":
		sortColumn = "COALESCE(i.name, ii.custom_name)"
	case "expiration":
		sortColumn = "ii.expiration_date"
	case "quantity":
		sortColumn = "ii.quantity"
	case "updated":
		sortColumn = "ii.updated_at"
	}
	if params.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM inventory_items ii
		LEFT JOIN items i ON ii.item_id = i.id
		WHERE %s
	`, whereClause)
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get items with details
	argCount++
	limitArg := argCount
	argCount++
	offsetArg := argCount
	args = append(args, params.Limit, params.Offset)

	query := fmt.Sprintf(`
		SELECT
			ii.id, ii.user_id, ii.item_id,
			ii.custom_name, ii.custom_brand, ii.custom_size, ii.custom_unit,
			ii.quantity, ii.unit,
			ii.low_stock_threshold, ii.low_stock_alert_enabled,
			ii.purchase_date, ii.expiration_date,
			ii.location, ii.notes,
			ii.created_at, ii.updated_at,
			i.name, i.brand, i.size, i.unit,
			CASE
				WHEN ii.low_stock_alert_enabled AND ii.quantity <= ii.low_stock_threshold THEN true
				ELSE false
			END as is_low_stock,
			CASE
				WHEN ii.expiration_date IS NOT NULL AND ii.expiration_date < CURRENT_DATE THEN true
				ELSE false
			END as is_expired,
			CASE
				WHEN ii.expiration_date IS NOT NULL
					AND ii.expiration_date >= CURRENT_DATE
					AND ii.expiration_date <= CURRENT_DATE + INTERVAL '7 days' THEN true
				ELSE false
			END as expires_soon,
			CASE
				WHEN ii.expiration_date IS NOT NULL THEN (ii.expiration_date - CURRENT_DATE)::int
				ELSE NULL
			END as days_until_expiry
		FROM inventory_items ii
		LEFT JOIN items i ON ii.item_id = i.id
		WHERE %s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, sortOrder, limitArg, offsetArg)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*models.InventoryItemWithDetails
	for rows.Next() {
		item := &models.InventoryItemWithDetails{}
		err := rows.Scan(
			&item.ID, &item.UserID, &item.ItemID,
			&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
			&item.Quantity, &item.Unit,
			&item.LowStockThreshold, &item.LowStockAlertEnabled,
			&item.PurchaseDate, &item.ExpirationDate,
			&item.Location, &item.Notes,
			&item.CreatedAt, &item.UpdatedAt,
			&item.ItemName, &item.ItemBrand, &item.ItemSize, &item.ItemUnit,
			&item.IsLowStock, &item.IsExpired, &item.ExpiresSoon, &item.DaysUntilExpiry,
		)
		if err != nil {
			return nil, 0, err
		}

		// Set display name
		if item.ItemName != nil {
			item.DisplayName = *item.ItemName
			item.DisplayBrand = item.ItemBrand
		} else if item.CustomName != nil {
			item.DisplayName = *item.CustomName
			item.DisplayBrand = item.CustomBrand
		} else {
			item.DisplayName = "Unknown Item"
		}

		items = append(items, item)
	}

	return items, total, nil
}

// GetInventoryItemByID retrieves a single inventory item with details
func (db *DB) GetInventoryItemByID(ctx context.Context, id int, userID int) (*models.InventoryItemWithDetails, error) {
	item := &models.InventoryItemWithDetails{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			ii.id, ii.user_id, ii.item_id,
			ii.custom_name, ii.custom_brand, ii.custom_size, ii.custom_unit,
			ii.quantity, ii.unit,
			ii.low_stock_threshold, ii.low_stock_alert_enabled,
			ii.purchase_date, ii.expiration_date,
			ii.location, ii.notes,
			ii.created_at, ii.updated_at,
			i.name, i.brand, i.size, i.unit,
			CASE
				WHEN ii.low_stock_alert_enabled AND ii.quantity <= ii.low_stock_threshold THEN true
				ELSE false
			END as is_low_stock,
			CASE
				WHEN ii.expiration_date IS NOT NULL AND ii.expiration_date < CURRENT_DATE THEN true
				ELSE false
			END as is_expired,
			CASE
				WHEN ii.expiration_date IS NOT NULL
					AND ii.expiration_date >= CURRENT_DATE
					AND ii.expiration_date <= CURRENT_DATE + INTERVAL '7 days' THEN true
				ELSE false
			END as expires_soon,
			CASE
				WHEN ii.expiration_date IS NOT NULL THEN (ii.expiration_date - CURRENT_DATE)::int
				ELSE NULL
			END as days_until_expiry
		FROM inventory_items ii
		LEFT JOIN items i ON ii.item_id = i.id
		WHERE ii.id = $1
	`, id).Scan(
		&item.ID, &item.UserID, &item.ItemID,
		&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
		&item.Quantity, &item.Unit,
		&item.LowStockThreshold, &item.LowStockAlertEnabled,
		&item.PurchaseDate, &item.ExpirationDate,
		&item.Location, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
		&item.ItemName, &item.ItemBrand, &item.ItemSize, &item.ItemUnit,
		&item.IsLowStock, &item.IsExpired, &item.ExpiresSoon, &item.DaysUntilExpiry,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemNotFound
		}
		return nil, err
	}

	// Check ownership
	if item.UserID != userID {
		return nil, ErrNotInventoryOwner
	}

	// Set display name
	if item.ItemName != nil {
		item.DisplayName = *item.ItemName
		item.DisplayBrand = item.ItemBrand
	} else if item.CustomName != nil {
		item.DisplayName = *item.CustomName
		item.DisplayBrand = item.CustomBrand
	} else {
		item.DisplayName = "Unknown Item"
	}

	return item, nil
}

// CreateInventoryItem adds a new item to user's inventory
func (db *DB) CreateInventoryItem(ctx context.Context, req *models.CreateInventoryItemRequest, userID int) (*models.InventoryItem, error) {
	item := &models.InventoryItem{}

	// Set defaults
	lowStockThreshold := 1.0
	if req.LowStockThreshold != nil {
		lowStockThreshold = *req.LowStockThreshold
	}
	lowStockAlertEnabled := true
	if req.LowStockAlertEnabled != nil {
		lowStockAlertEnabled = *req.LowStockAlertEnabled
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO inventory_items (
			user_id, item_id,
			custom_name, custom_brand, custom_size, custom_unit,
			quantity, unit,
			low_stock_threshold, low_stock_alert_enabled,
			purchase_date, expiration_date,
			location, notes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, NOW(), NOW()
		)
		RETURNING id, user_id, item_id,
			custom_name, custom_brand, custom_size, custom_unit,
			quantity, unit,
			low_stock_threshold, low_stock_alert_enabled,
			purchase_date, expiration_date,
			location, notes,
			created_at, updated_at
	`,
		userID, req.ItemID,
		req.CustomName, req.CustomBrand, req.CustomSize, req.CustomUnit,
		req.Quantity, req.Unit,
		lowStockThreshold, lowStockAlertEnabled,
		req.PurchaseDate, req.ExpirationDate,
		req.Location, req.Notes,
	).Scan(
		&item.ID, &item.UserID, &item.ItemID,
		&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
		&item.Quantity, &item.Unit,
		&item.LowStockThreshold, &item.LowStockAlertEnabled,
		&item.PurchaseDate, &item.ExpirationDate,
		&item.Location, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return item, nil
}

// UpdateInventoryItem updates an inventory item
func (db *DB) UpdateInventoryItem(ctx context.Context, id int, userID int, req *models.UpdateInventoryItemRequest) (*models.InventoryItem, error) {
	// First check ownership
	var ownerID int
	err := db.Pool.QueryRow(ctx, `SELECT user_id FROM inventory_items WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemNotFound
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotInventoryOwner
	}

	item := &models.InventoryItem{}

	err = db.Pool.QueryRow(ctx, `
		UPDATE inventory_items
		SET
			quantity = COALESCE($3, quantity),
			unit = COALESCE($4, unit),
			low_stock_threshold = COALESCE($5, low_stock_threshold),
			low_stock_alert_enabled = COALESCE($6, low_stock_alert_enabled),
			purchase_date = COALESCE($7, purchase_date),
			expiration_date = COALESCE($8, expiration_date),
			location = COALESCE($9, location),
			notes = COALESCE($10, notes),
			custom_name = COALESCE($11, custom_name),
			custom_brand = COALESCE($12, custom_brand),
			custom_size = COALESCE($13, custom_size),
			custom_unit = COALESCE($14, custom_unit),
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, item_id,
			custom_name, custom_brand, custom_size, custom_unit,
			quantity, unit,
			low_stock_threshold, low_stock_alert_enabled,
			purchase_date, expiration_date,
			location, notes,
			created_at, updated_at
	`, id, userID,
		req.Quantity, req.Unit,
		req.LowStockThreshold, req.LowStockAlertEnabled,
		req.PurchaseDate, req.ExpirationDate,
		req.Location, req.Notes,
		req.CustomName, req.CustomBrand, req.CustomSize, req.CustomUnit,
	).Scan(
		&item.ID, &item.UserID, &item.ItemID,
		&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
		&item.Quantity, &item.Unit,
		&item.LowStockThreshold, &item.LowStockAlertEnabled,
		&item.PurchaseDate, &item.ExpirationDate,
		&item.Location, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemNotFound
		}
		return nil, err
	}

	return item, nil
}

// DeleteInventoryItem removes an item from inventory
func (db *DB) DeleteInventoryItem(ctx context.Context, id int, userID int) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM inventory_items WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrInventoryItemNotFound
	}

	return nil
}

// AdjustInventoryQuantity adds or subtracts from current quantity
func (db *DB) AdjustInventoryQuantity(ctx context.Context, id int, userID int, adjustment float64) (*models.InventoryItem, error) {
	// First check ownership
	var ownerID int
	err := db.Pool.QueryRow(ctx, `SELECT user_id FROM inventory_items WHERE id = $1`, id).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemNotFound
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotInventoryOwner
	}

	item := &models.InventoryItem{}

	err = db.Pool.QueryRow(ctx, `
		UPDATE inventory_items
		SET quantity = GREATEST(0, quantity + $3), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, item_id,
			custom_name, custom_brand, custom_size, custom_unit,
			quantity, unit,
			low_stock_threshold, low_stock_alert_enabled,
			purchase_date, expiration_date,
			location, notes,
			created_at, updated_at
	`, id, userID, adjustment).Scan(
		&item.ID, &item.UserID, &item.ItemID,
		&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
		&item.Quantity, &item.Unit,
		&item.LowStockThreshold, &item.LowStockAlertEnabled,
		&item.PurchaseDate, &item.ExpirationDate,
		&item.Location, &item.Notes,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInventoryItemNotFound
		}
		return nil, err
	}

	return item, nil
}

// GetInventorySummary returns aggregate stats for user's inventory
func (db *DB) GetInventorySummary(ctx context.Context, userID int) (*models.InventorySummary, error) {
	summary := &models.InventorySummary{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_items,
			COUNT(*) FILTER (WHERE low_stock_alert_enabled AND quantity <= low_stock_threshold) as low_stock_count,
			COUNT(*) FILTER (WHERE expiration_date IS NOT NULL AND expiration_date < CURRENT_DATE) as expired_count,
			COUNT(*) FILTER (WHERE expiration_date IS NOT NULL AND expiration_date >= CURRENT_DATE AND expiration_date <= CURRENT_DATE + INTERVAL '7 days') as expiring_soon_count
		FROM inventory_items
		WHERE user_id = $1
	`, userID).Scan(&summary.TotalItems, &summary.LowStockCount, &summary.ExpiredCount, &summary.ExpiringSoonCount)

	if err != nil {
		return nil, err
	}

	// Get unique locations
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT location
		FROM inventory_items
		WHERE user_id = $1 AND location IS NOT NULL AND location != ''
		ORDER BY location
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary.UniqueLocations = []string{}
	for rows.Next() {
		var location string
		if err := rows.Scan(&location); err != nil {
			return nil, err
		}
		summary.UniqueLocations = append(summary.UniqueLocations, location)
	}

	return summary, nil
}

// GetLowStockItems returns items below their threshold
func (db *DB) GetLowStockItems(ctx context.Context, userID int) ([]*models.InventoryItemWithDetails, error) {
	params := &models.InventoryListParams{
		UserID:   userID,
		Limit:    100,
		Offset:   0,
		LowStock: boolPtr(true),
		SortBy:   "quantity",
		SortOrder: "asc",
	}
	items, _, err := db.ListInventoryItems(ctx, params)
	return items, err
}

// GetExpiringItems returns items expiring within specified days
func (db *DB) GetExpiringItems(ctx context.Context, userID int, daysAhead int) ([]*models.InventoryItemWithDetails, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			ii.id, ii.user_id, ii.item_id,
			ii.custom_name, ii.custom_brand, ii.custom_size, ii.custom_unit,
			ii.quantity, ii.unit,
			ii.low_stock_threshold, ii.low_stock_alert_enabled,
			ii.purchase_date, ii.expiration_date,
			ii.location, ii.notes,
			ii.created_at, ii.updated_at,
			i.name, i.brand, i.size, i.unit,
			CASE
				WHEN ii.low_stock_alert_enabled AND ii.quantity <= ii.low_stock_threshold THEN true
				ELSE false
			END as is_low_stock,
			CASE
				WHEN ii.expiration_date IS NOT NULL AND ii.expiration_date < CURRENT_DATE THEN true
				ELSE false
			END as is_expired,
			CASE
				WHEN ii.expiration_date IS NOT NULL
					AND ii.expiration_date >= CURRENT_DATE
					AND ii.expiration_date <= CURRENT_DATE + INTERVAL '7 days' THEN true
				ELSE false
			END as expires_soon,
			CASE
				WHEN ii.expiration_date IS NOT NULL THEN (ii.expiration_date - CURRENT_DATE)::int
				ELSE NULL
			END as days_until_expiry
		FROM inventory_items ii
		LEFT JOIN items i ON ii.item_id = i.id
		WHERE ii.user_id = $1
			AND ii.expiration_date IS NOT NULL
			AND ii.expiration_date <= CURRENT_DATE + ($2 || ' days')::INTERVAL
		ORDER BY ii.expiration_date ASC
	`, userID, daysAhead)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.InventoryItemWithDetails
	for rows.Next() {
		item := &models.InventoryItemWithDetails{}
		err := rows.Scan(
			&item.ID, &item.UserID, &item.ItemID,
			&item.CustomName, &item.CustomBrand, &item.CustomSize, &item.CustomUnit,
			&item.Quantity, &item.Unit,
			&item.LowStockThreshold, &item.LowStockAlertEnabled,
			&item.PurchaseDate, &item.ExpirationDate,
			&item.Location, &item.Notes,
			&item.CreatedAt, &item.UpdatedAt,
			&item.ItemName, &item.ItemBrand, &item.ItemSize, &item.ItemUnit,
			&item.IsLowStock, &item.IsExpired, &item.ExpiresSoon, &item.DaysUntilExpiry,
		)
		if err != nil {
			return nil, err
		}

		// Set display name
		if item.ItemName != nil {
			item.DisplayName = *item.ItemName
			item.DisplayBrand = item.ItemBrand
		} else if item.CustomName != nil {
			item.DisplayName = *item.CustomName
			item.DisplayBrand = item.CustomBrand
		} else {
			item.DisplayName = "Unknown Item"
		}

		items = append(items, item)
	}

	return items, nil
}

// GetInventoryLocations returns unique locations for a user's inventory
func (db *DB) GetInventoryLocations(ctx context.Context, userID int) ([]string, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT location
		FROM inventory_items
		WHERE user_id = $1 AND location IS NOT NULL AND location != ''
		ORDER BY location
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []string
	for rows.Next() {
		var location string
		if err := rows.Scan(&location); err != nil {
			return nil, err
		}
		locations = append(locations, location)
	}

	return locations, nil
}

// AddInventoryItemToShoppingList adds an inventory item to a shopping list
func (db *DB) AddInventoryItemToShoppingList(ctx context.Context, inventoryID int, userID int, listID int, quantity int) error {
	// Verify inventory item ownership and get item details
	var itemID *int
	var inventoryOwnerID int
	err := db.Pool.QueryRow(ctx, `
		SELECT user_id, item_id FROM inventory_items WHERE id = $1
	`, inventoryID).Scan(&inventoryOwnerID, &itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrInventoryItemNotFound
		}
		return err
	}
	if inventoryOwnerID != userID {
		return ErrNotInventoryOwner
	}

	// Check if this is a catalog item
	if itemID == nil {
		return errors.New("cannot add custom inventory items to shopping list (no catalog item linked)")
	}

	// Verify list ownership
	var listOwnerID int
	err = db.Pool.QueryRow(ctx, `SELECT user_id FROM shopping_lists WHERE id = $1`, listID).Scan(&listOwnerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrListNotFound
		}
		return err
	}
	if listOwnerID != userID {
		return ErrNotListOwner
	}

	// Add to shopping list (upsert - add quantity if already exists)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO shopping_list_items (list_id, item_id, quantity, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (list_id, item_id) DO UPDATE SET quantity = shopping_list_items.quantity + $3
	`, listID, *itemID, quantity)
	if err != nil {
		return err
	}

	// Update list's updated_at
	_, _ = db.Pool.Exec(ctx, `UPDATE shopping_lists SET updated_at = NOW() WHERE id = $1`, listID)

	return nil
}

// GetActiveShoppingLists returns user's active shopping lists (for quick-add dropdown)
func (db *DB) GetActiveShoppingLists(ctx context.Context, userID int) ([]*models.ShoppingListSummary, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sl.id, sl.name, sl.target_date, sl.status, sl.completed_at, sl.created_at, sl.updated_at,
			COALESCE((SELECT COUNT(*) FROM shopping_list_items WHERE list_id = sl.id), 0) as item_count,
			0 as estimated_total
		FROM shopping_lists sl
		WHERE sl.user_id = $1 AND sl.status = 'active'
		ORDER BY sl.updated_at DESC
		LIMIT 10
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []*models.ShoppingListSummary
	for rows.Next() {
		l := &models.ShoppingListSummary{}
		err := rows.Scan(
			&l.ID, &l.Name, &l.TargetDate, &l.Status, &l.CompletedAt, &l.CreatedAt, &l.UpdatedAt,
			&l.ItemCount, &l.EstimatedTotal,
		)
		if err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}

	return lists, nil
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}

// Ensure time package is used
var _ = time.Now
