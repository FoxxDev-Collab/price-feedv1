package database

import (
	"context"
	"errors"
	"sort"

	"github.com/jackc/pgx/v5"

	"github.com/foxxcyber/price-feed/internal/models"
)

var (
	ErrListNotFound     = errors.New("shopping list not found")
	ErrListItemNotFound = errors.New("list item not found")
	ErrNotListOwner     = errors.New("not the owner of this list")
)

// ListShoppingLists returns all shopping lists for a user
func (db *DB) ListShoppingLists(ctx context.Context, params *models.ListListParams) ([]*models.ShoppingListSummary, int, error) {
	// Build where clause based on status filter
	statusFilter := ""
	if params.Status != "" {
		statusFilter = " AND sl.status = '" + string(params.Status) + "'"
	}

	// Get total count
	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM shopping_lists sl WHERE user_id = $1`+statusFilter,
		params.UserID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get lists with item counts and estimated totals
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sl.id, sl.name, sl.target_date, sl.status, sl.completed_at, sl.created_at, sl.updated_at,
			COALESCE((SELECT COUNT(*) FROM shopping_list_items WHERE list_id = sl.id), 0) as item_count,
			COALESCE((
				SELECT SUM(
					sli.quantity * COALESCE(
						(SELECT MIN(sp.price) FROM store_prices sp WHERE sp.item_id = sli.item_id),
						0
					)
				)
				FROM shopping_list_items sli
				WHERE sli.list_id = sl.id
			), 0) as estimated_total
		FROM shopping_lists sl
		WHERE sl.user_id = $1`+statusFilter+`
		ORDER BY sl.updated_at DESC
		LIMIT $2 OFFSET $3
	`, params.UserID, params.Limit, params.Offset)
	if err != nil {
		return nil, 0, err
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
			return nil, 0, err
		}
		lists = append(lists, l)
	}

	return lists, total, nil
}

// GetShoppingListByID retrieves a shopping list with all its items
func (db *DB) GetShoppingListByID(ctx context.Context, id int, userID int) (*models.ShoppingListWithItems, error) {
	// Get the list
	list := &models.ShoppingListWithItems{}
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, name, status, target_date, completed_at, created_at, updated_at
		FROM shopping_lists
		WHERE id = $1
	`, id).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Status, &list.TargetDate, &list.CompletedAt, &list.CreatedAt, &list.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	// Check ownership
	if list.UserID != userID {
		return nil, ErrNotListOwner
	}

	// Get items with details
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sli.id, sli.list_id, sli.item_id, sli.quantity, sli.created_at,
			i.name, i.brand, i.size, i.unit,
			(SELECT MIN(sp.price) FROM store_prices sp WHERE sp.item_id = sli.item_id) as best_price,
			(SELECT s.name FROM stores s
			 JOIN store_prices sp ON s.id = sp.store_id
			 WHERE sp.item_id = sli.item_id
			 ORDER BY sp.price ASC
			 LIMIT 1) as best_store
		FROM shopping_list_items sli
		JOIN items i ON sli.item_id = i.id
		WHERE sli.list_id = $1
		ORDER BY i.name ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list.Items = []models.ShoppingListItemWithDetails{}
	var estimatedTotal float64

	for rows.Next() {
		item := models.ShoppingListItemWithDetails{}
		err := rows.Scan(
			&item.ID, &item.ListID, &item.ItemID, &item.Quantity, &item.CreatedAt,
			&item.ItemName, &item.ItemBrand, &item.ItemSize, &item.ItemUnit,
			&item.BestPrice, &item.BestStore,
		)
		if err != nil {
			return nil, err
		}
		list.Items = append(list.Items, item)
		if item.BestPrice != nil {
			estimatedTotal += *item.BestPrice * float64(item.Quantity)
		}
	}

	list.ItemCount = len(list.Items)
	list.EstimatedTotal = estimatedTotal

	return list, nil
}

// CreateShoppingList creates a new shopping list
func (db *DB) CreateShoppingList(ctx context.Context, req *models.CreateListRequest, userID int) (*models.ShoppingList, error) {
	list := &models.ShoppingList{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO shopping_lists (user_id, name, status, target_date, created_at, updated_at)
		VALUES ($1, $2, 'active', $3, NOW(), NOW())
		RETURNING id, user_id, name, status, target_date, completed_at, created_at, updated_at
	`, userID, req.Name, req.TargetDate).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Status, &list.TargetDate, &list.CompletedAt, &list.CreatedAt, &list.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return list, nil
}

// UpdateShoppingList updates a shopping list
func (db *DB) UpdateShoppingList(ctx context.Context, id int, userID int, req *models.UpdateListRequest) (*models.ShoppingList, error) {
	list := &models.ShoppingList{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE shopping_lists
		SET name = COALESCE($3, name),
		    target_date = COALESCE($4, target_date),
		    updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, status, target_date, completed_at, created_at, updated_at
	`, id, userID, req.Name, req.TargetDate).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Status, &list.TargetDate, &list.CompletedAt, &list.CreatedAt, &list.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	return list, nil
}

// DeleteShoppingList deletes a shopping list
func (db *DB) DeleteShoppingList(ctx context.Context, id int, userID int) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM shopping_lists WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrListNotFound
	}

	return nil
}

// AddItemToList adds an item to a shopping list
func (db *DB) AddItemToList(ctx context.Context, listID int, userID int, req *models.AddListItemRequest) (*models.ShoppingListItem, error) {
	// Verify list ownership
	var ownerID int
	err := db.Pool.QueryRow(ctx, `SELECT user_id FROM shopping_lists WHERE id = $1`, listID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotListOwner
	}

	item := &models.ShoppingListItem{}
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO shopping_list_items (list_id, item_id, quantity, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (list_id, item_id) DO UPDATE SET quantity = shopping_list_items.quantity + $3
		RETURNING id, list_id, item_id, quantity, created_at
	`, listID, req.ItemID, req.Quantity).Scan(
		&item.ID, &item.ListID, &item.ItemID, &item.Quantity, &item.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Update list's updated_at
	_, _ = db.Pool.Exec(ctx, `UPDATE shopping_lists SET updated_at = NOW() WHERE id = $1`, listID)

	return item, nil
}

// UpdateListItem updates the quantity of an item in a list
func (db *DB) UpdateListItem(ctx context.Context, listID int, itemID int, userID int, req *models.UpdateListItemRequest) (*models.ShoppingListItem, error) {
	// Verify list ownership
	var ownerID int
	err := db.Pool.QueryRow(ctx, `SELECT user_id FROM shopping_lists WHERE id = $1`, listID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotListOwner
	}

	item := &models.ShoppingListItem{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE shopping_list_items
		SET quantity = $3
		WHERE list_id = $1 AND item_id = $2
		RETURNING id, list_id, item_id, quantity, created_at
	`, listID, itemID, req.Quantity).Scan(
		&item.ID, &item.ListID, &item.ItemID, &item.Quantity, &item.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListItemNotFound
		}
		return nil, err
	}

	// Update list's updated_at
	_, _ = db.Pool.Exec(ctx, `UPDATE shopping_lists SET updated_at = NOW() WHERE id = $1`, listID)

	return item, nil
}

// RemoveItemFromList removes an item from a shopping list
func (db *DB) RemoveItemFromList(ctx context.Context, listID int, itemID int, userID int) error {
	// Verify list ownership
	var ownerID int
	err := db.Pool.QueryRow(ctx, `SELECT user_id FROM shopping_lists WHERE id = $1`, listID).Scan(&ownerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrListNotFound
		}
		return err
	}
	if ownerID != userID {
		return ErrNotListOwner
	}

	result, err := db.Pool.Exec(ctx, `
		DELETE FROM shopping_list_items WHERE list_id = $1 AND item_id = $2
	`, listID, itemID)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrListItemNotFound
	}

	// Update list's updated_at
	_, _ = db.Pool.Exec(ctx, `UPDATE shopping_lists SET updated_at = NOW() WHERE id = $1`, listID)

	return nil
}

// BuildShoppingPlan generates an optimized shopping plan for a list
func (db *DB) BuildShoppingPlan(ctx context.Context, listID int, userID int, regionID *int) (*models.ShoppingPlanResult, error) {
	// Verify list ownership and get items
	list, err := db.GetShoppingListByID(ctx, listID, userID)
	if err != nil {
		return nil, err
	}

	if len(list.Items) == 0 {
		return nil, errors.New("shopping list is empty")
	}

	// Get all item IDs from the list
	itemIDs := make([]int, len(list.Items))
	itemQuantities := make(map[int]int)
	for i, item := range list.Items {
		itemIDs[i] = item.ItemID
		itemQuantities[item.ItemID] = item.Quantity
	}

	// Build price matrix: map[storeID]map[itemID]price
	priceMatrix := make(map[int]map[int]float64)
	storeNames := make(map[int]string)
	itemNames := make(map[int]string)

	// Query all prices for the items in the list
	// Include both shared prices and user's private prices
	rows, err := db.Pool.Query(ctx, `
		SELECT
			sp.store_id, sp.item_id, sp.price,
			s.name as store_name, i.name as item_name
		FROM store_prices sp
		JOIN stores s ON sp.store_id = s.id
		JOIN items i ON sp.item_id = i.id
		WHERE sp.item_id = ANY($1)
		AND (sp.is_shared = true OR sp.user_id = $2)
		AND (s.is_private = false OR s.created_by = $2)
		AND ($3::int IS NULL OR s.region_id = $3)
		ORDER BY sp.verified_count DESC, sp.updated_at DESC
	`, itemIDs, userID, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var storeID, itemID int
		var price float64
		var storeName, itemName string
		if err := rows.Scan(&storeID, &itemID, &price, &storeName, &itemName); err != nil {
			return nil, err
		}

		if priceMatrix[storeID] == nil {
			priceMatrix[storeID] = make(map[int]float64)
		}
		// Only keep the first (best verified/most recent) price per store/item
		if _, exists := priceMatrix[storeID][itemID]; !exists {
			priceMatrix[storeID][itemID] = price
		}
		storeNames[storeID] = storeName
		itemNames[itemID] = itemName
	}

	// Calculate single-store options
	var singleStoreOptions []models.SingleStoreOption
	for storeID, prices := range priceMatrix {
		option := models.SingleStoreOption{
			StoreID:   storeID,
			StoreName: storeNames[storeID],
		}

		for _, itemID := range itemIDs {
			if price, exists := prices[itemID]; exists {
				option.TotalCost += price * float64(itemQuantities[itemID])
				option.ItemsFound++
			} else {
				option.ItemsMissing = append(option.ItemsMissing, itemNames[itemID])
			}
		}
		singleStoreOptions = append(singleStoreOptions, option)
	}

	// Sort by total cost (stores with all items first, then by price)
	sort.Slice(singleStoreOptions, func(i, j int) bool {
		// Prefer stores with more items
		if singleStoreOptions[i].ItemsFound != singleStoreOptions[j].ItemsFound {
			return singleStoreOptions[i].ItemsFound > singleStoreOptions[j].ItemsFound
		}
		// Then by total cost
		return singleStoreOptions[i].TotalCost < singleStoreOptions[j].TotalCost
	})

	var bestSingleStore *models.SingleStoreOption
	if len(singleStoreOptions) > 0 {
		bestSingleStore = &singleStoreOptions[0]
	}

	// Calculate multi-store option (best price per item)
	multiStore := &models.MultiStoreOption{
		Stores: []models.MultiStoreBreakdown{},
	}

	storeItems := make(map[int][]models.StorePlanItemWithDetails)
	storeSubtotals := make(map[int]float64)

	for _, itemID := range itemIDs {
		var bestPrice float64 = -1
		var bestStoreID int

		// Find the best price across all stores
		for storeID, prices := range priceMatrix {
			if price, exists := prices[itemID]; exists {
				if bestPrice < 0 || price < bestPrice {
					bestPrice = price
					bestStoreID = storeID
				}
			}
		}

		if bestPrice >= 0 {
			quantity := itemQuantities[itemID]
			item := models.StorePlanItemWithDetails{
				StorePlanItem: models.StorePlanItem{
					StoreID:  bestStoreID,
					ItemID:   itemID,
					Quantity: quantity,
					Price:    bestPrice,
				},
				StoreName: storeNames[bestStoreID],
				ItemName:  itemNames[itemID],
			}
			storeItems[bestStoreID] = append(storeItems[bestStoreID], item)
			storeSubtotals[bestStoreID] += bestPrice * float64(quantity)
			multiStore.TotalCost += bestPrice * float64(quantity)
		}
	}

	// Build store breakdowns
	for storeID, items := range storeItems {
		breakdown := models.MultiStoreBreakdown{
			StoreID:   storeID,
			StoreName: storeNames[storeID],
			Items:     items,
			Subtotal:  storeSubtotals[storeID],
		}
		multiStore.Stores = append(multiStore.Stores, breakdown)
	}

	multiStore.TripCount = len(multiStore.Stores)

	// Calculate savings
	if bestSingleStore != nil {
		multiStore.TotalSavings = bestSingleStore.TotalCost - multiStore.TotalCost
	}

	// Determine recommendation
	recommendation := "single_store"
	savingsThreshold := 10.00 // Configurable threshold

	if multiStore.TotalSavings >= savingsThreshold && multiStore.TripCount <= 3 {
		recommendation = "multi_store"
	}

	result := &models.ShoppingPlanResult{
		ListID:         listID,
		SingleStore:    bestSingleStore,
		MultiStore:     multiStore,
		Recommendation: recommendation,
	}

	return result, nil
}

// GetPriceComparison generates a price comparison grid
func (db *DB) GetPriceComparison(ctx context.Context, params *models.CompareParams) (*models.PriceComparisonResult, error) {
	result := &models.PriceComparisonResult{
		Stores: []models.StoreBasic{},
		Items:  []models.PriceComparisonRow{},
	}

	// Get store info
	storeRows, err := db.Pool.Query(ctx, `
		SELECT id, name FROM stores WHERE id = ANY($1) ORDER BY name
	`, params.StoreIDs)
	if err != nil {
		return nil, err
	}
	defer storeRows.Close()

	for storeRows.Next() {
		var s models.StoreBasic
		if err := storeRows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		result.Stores = append(result.Stores, s)
	}

	// Build the query for prices
	var priceQuery string
	var args []interface{}

	if len(params.ItemIDs) > 0 {
		// Specific items
		priceQuery = `
			SELECT
				i.id, i.name, i.brand, i.size, i.unit,
				sp.store_id, sp.price, sp.verified_count, u.username, sp.updated_at
			FROM items i
			LEFT JOIN store_prices sp ON i.id = sp.item_id AND sp.store_id = ANY($1)
				AND (sp.is_shared = true OR sp.user_id = $3)
			LEFT JOIN users u ON sp.user_id = u.id
			WHERE i.id = ANY($2)
			ORDER BY i.name, sp.store_id
		`
		args = []interface{}{params.StoreIDs, params.ItemIDs, params.UserID}
	} else {
		// All items that have prices at any of the selected stores
		priceQuery = `
			SELECT
				i.id, i.name, i.brand, i.size, i.unit,
				sp.store_id, sp.price, sp.verified_count, u.username, sp.updated_at
			FROM items i
			JOIN store_prices sp ON i.id = sp.item_id
			LEFT JOIN users u ON sp.user_id = u.id
			WHERE sp.store_id = ANY($1)
				AND (sp.is_shared = true OR sp.user_id = $2)
			ORDER BY i.name, sp.store_id
		`
		args = []interface{}{params.StoreIDs, params.UserID}
	}

	rows, err := db.Pool.Query(ctx, priceQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Build the grid
	itemMap := make(map[int]*models.PriceComparisonRow)

	for rows.Next() {
		var itemID int
		var itemName string
		var itemBrand, itemUnit, username *string
		var itemSize *float64
		var storeID *int
		var price *float64
		var verifiedCount *int
		var updatedAt *string

		if err := rows.Scan(&itemID, &itemName, &itemBrand, &itemSize, &itemUnit,
			&storeID, &price, &verifiedCount, &username, &updatedAt); err != nil {
			return nil, err
		}

		row, exists := itemMap[itemID]
		if !exists {
			row = &models.PriceComparisonRow{
				ItemID:    itemID,
				ItemName:  itemName,
				ItemBrand: itemBrand,
				ItemSize:  itemSize,
				ItemUnit:  itemUnit,
				Prices:    make(map[int]models.PriceComparisonCell),
			}
			itemMap[itemID] = row
		}

		if storeID != nil && price != nil {
			vc := 0
			if verifiedCount != nil {
				vc = *verifiedCount
			}
			row.Prices[*storeID] = models.PriceComparisonCell{
				Price:         price,
				VerifiedCount: vc,
				SubmittedBy:   username,
				UpdatedAt:     updatedAt,
			}

			// Track best price
			if row.BestPrice == nil || *price < *row.BestPrice {
				row.BestPrice = price
				row.BestStore = storeID
			}
		}
	}

	// Convert map to slice and mark best prices
	for _, row := range itemMap {
		// Mark the best price cell
		if row.BestStore != nil {
			if cell, exists := row.Prices[*row.BestStore]; exists {
				cell.IsBest = true
				row.Prices[*row.BestStore] = cell
			}
		}
		result.Items = append(result.Items, *row)
	}

	// Sort items by name
	sort.Slice(result.Items, func(i, j int) bool {
		return result.Items[i].ItemName < result.Items[j].ItemName
	})

	return result, nil
}

// CompleteShoppingList marks a shopping list as completed and processes price confirmations
func (db *DB) CompleteShoppingList(ctx context.Context, listID int, userID int, req *models.CompleteListRequest) (*models.ShoppingList, error) {
	// Verify list ownership
	var ownerID int
	var currentStatus string
	err := db.Pool.QueryRow(ctx, `SELECT user_id, status FROM shopping_lists WHERE id = $1`, listID).Scan(&ownerID, &currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrNotListOwner
	}

	// Check if already completed
	if currentStatus == string(models.ListStatusCompleted) {
		return nil, errors.New("list is already completed")
	}

	// Process price confirmations if provided
	if req != nil && len(req.PriceConfirmations) > 0 {
		for _, confirmation := range req.PriceConfirmations {
			if confirmation.IsAccurate {
				// Verify the existing price
				_, err := db.Pool.Exec(ctx, `
					INSERT INTO price_verifications (price_id, user_id, is_accurate, created_at)
					SELECT sp.id, $1, true, NOW()
					FROM store_prices sp
					WHERE sp.item_id = $2 AND sp.store_id = $3
					ON CONFLICT (price_id, user_id) DO UPDATE SET is_accurate = true, created_at = NOW()
				`, userID, confirmation.ItemID, confirmation.StoreID)
				if err != nil {
					return nil, err
				}

				// Update verified count on the price
				_, _ = db.Pool.Exec(ctx, `
					UPDATE store_prices
					SET verified_count = verified_count + 1, last_verified = NOW()
					WHERE item_id = $1 AND store_id = $2
				`, confirmation.ItemID, confirmation.StoreID)
			} else if confirmation.NewPrice != nil {
				// User provided a corrected price - update or insert new price
				_, err := db.Pool.Exec(ctx, `
					INSERT INTO store_prices (store_id, item_id, price, user_id, is_shared, verified_count, created_at, updated_at)
					VALUES ($1, $2, $3, $4, true, 1, NOW(), NOW())
					ON CONFLICT (store_id, item_id) DO UPDATE SET
						price = $3,
						user_id = $4,
						verified_count = 1,
						last_verified = NOW(),
						updated_at = NOW()
				`, confirmation.StoreID, confirmation.ItemID, *confirmation.NewPrice, userID)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// Mark list as completed
	list := &models.ShoppingList{}
	err = db.Pool.QueryRow(ctx, `
		UPDATE shopping_lists
		SET status = 'completed', completed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, status, target_date, completed_at, created_at, updated_at
	`, listID, userID).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Status, &list.TargetDate, &list.CompletedAt, &list.CreatedAt, &list.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return list, nil
}

// ReopenShoppingList marks a completed list as active again
func (db *DB) ReopenShoppingList(ctx context.Context, listID int, userID int) (*models.ShoppingList, error) {
	list := &models.ShoppingList{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE shopping_lists
		SET status = 'active', completed_at = NULL, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, name, status, target_date, completed_at, created_at, updated_at
	`, listID, userID).Scan(
		&list.ID, &list.UserID, &list.Name, &list.Status, &list.TargetDate, &list.CompletedAt, &list.CreatedAt, &list.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrListNotFound
		}
		return nil, err
	}

	return list, nil
}
