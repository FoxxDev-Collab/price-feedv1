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
	ErrItemNotFound = errors.New("item not found")
)

// ListItems returns a paginated list of items with optional filtering
func (db *DB) ListItems(ctx context.Context, params *models.ItemListParams) ([]*models.ItemWithStats, int, error) {
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	if params.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"(LOWER(i.name) LIKE LOWER($%d) OR LOWER(i.brand) LIKE LOWER($%d))",
			argIndex, argIndex,
		))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	}

	if params.Tag != "" {
		whereClauses = append(whereClauses, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM item_tags it JOIN tags t ON it.tag_id = t.id WHERE it.item_id = i.id AND LOWER(t.name) = LOWER($%d))",
			argIndex,
		))
		args = append(args, params.Tag)
		argIndex++
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM items i %s", whereClause)
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get items with stats
	query := fmt.Sprintf(`
		SELECT
			i.id, i.name, i.brand, i.size, i.unit, i.description,
			i.verified, i.verification_count, i.created_by, i.created_at, i.updated_at,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE item_id = i.id), 0) as price_count,
			(SELECT AVG(price) FROM store_prices WHERE item_id = i.id) as avg_price,
			(SELECT MIN(price) FROM store_prices WHERE item_id = i.id) as min_price,
			(SELECT MAX(price) FROM store_prices WHERE item_id = i.id) as max_price,
			COALESCE(
				(SELECT array_agg(t.name ORDER BY t.name)
				 FROM item_tags it JOIN tags t ON it.tag_id = t.id
				 WHERE it.item_id = i.id),
				ARRAY[]::TEXT[]
			) as tags
		FROM items i
		%s
		ORDER BY i.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*models.ItemWithStats
	for rows.Next() {
		item := &models.ItemWithStats{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.Brand, &item.Size, &item.Unit, &item.Description,
			&item.Verified, &item.VerificationCount, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
			&item.PriceCount, &item.AvgPrice, &item.MinPrice, &item.MaxPrice,
			&item.Tags,
		)
		if err != nil {
			return nil, 0, err
		}
		if item.Tags == nil {
			item.Tags = []string{}
		}
		items = append(items, item)
	}

	return items, total, nil
}

// GetItemByID retrieves an item by ID with stats
func (db *DB) GetItemByID(ctx context.Context, id int) (*models.ItemWithStats, error) {
	item := &models.ItemWithStats{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			i.id, i.name, i.brand, i.size, i.unit, i.description,
			i.verified, i.verification_count, i.created_by, i.created_at, i.updated_at,
			COALESCE((SELECT COUNT(*) FROM store_prices WHERE item_id = i.id), 0) as price_count,
			(SELECT AVG(price) FROM store_prices WHERE item_id = i.id) as avg_price,
			(SELECT MIN(price) FROM store_prices WHERE item_id = i.id) as min_price,
			(SELECT MAX(price) FROM store_prices WHERE item_id = i.id) as max_price,
			COALESCE(
				(SELECT array_agg(t.name ORDER BY t.name)
				 FROM item_tags it JOIN tags t ON it.tag_id = t.id
				 WHERE it.item_id = i.id),
				ARRAY[]::TEXT[]
			) as tags
		FROM items i
		WHERE i.id = $1
	`, id).Scan(
		&item.ID, &item.Name, &item.Brand, &item.Size, &item.Unit, &item.Description,
		&item.Verified, &item.VerificationCount, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		&item.PriceCount, &item.AvgPrice, &item.MinPrice, &item.MaxPrice,
		&item.Tags,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	if item.Tags == nil {
		item.Tags = []string{}
	}

	return item, nil
}

// CreateItem creates a new item
func (db *DB) CreateItem(ctx context.Context, req *models.CreateItemRequest, createdBy *int) (*models.Item, error) {
	item := &models.Item{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO items (name, brand, size, unit, description, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, name, brand, size, unit, description, verified, verification_count, created_by, created_at, updated_at
	`, req.Name, req.Brand, req.Size, req.Unit, req.Description, createdBy).Scan(
		&item.ID, &item.Name, &item.Brand, &item.Size, &item.Unit, &item.Description,
		&item.Verified, &item.VerificationCount, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Add tags if provided
	if len(req.Tags) > 0 {
		for _, tagName := range req.Tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))

			// Insert or get tag
			var tagID int
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO tags (name, slug, usage_count, created_at)
				VALUES ($1, $2, 1, NOW())
				ON CONFLICT (slug) DO UPDATE SET usage_count = tags.usage_count + 1
				RETURNING id
			`, tagName, slug).Scan(&tagID)
			if err != nil {
				continue
			}

			// Link tag to item
			_, _ = db.Pool.Exec(ctx, `
				INSERT INTO item_tags (item_id, tag_id, created_by, created_at)
				VALUES ($1, $2, $3, NOW())
				ON CONFLICT DO NOTHING
			`, item.ID, tagID, createdBy)
		}
	}

	return item, nil
}

// UpdateItem updates an existing item
func (db *DB) UpdateItem(ctx context.Context, id int, req *models.UpdateItemRequest) (*models.Item, error) {
	item := &models.Item{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE items
		SET name = COALESCE($2, name),
		    brand = COALESCE($3, brand),
		    size = COALESCE($4, size),
		    unit = COALESCE($5, unit),
		    description = COALESCE($6, description),
		    verified = COALESCE($7, verified),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, brand, size, unit, description, verified, verification_count, created_by, created_at, updated_at
	`, id, req.Name, req.Brand, req.Size, req.Unit, req.Description, req.Verified).Scan(
		&item.ID, &item.Name, &item.Brand, &item.Size, &item.Unit, &item.Description,
		&item.Verified, &item.VerificationCount, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}

	// Update tags if provided
	if req.Tags != nil {
		// Remove existing tags
		_, _ = db.Pool.Exec(ctx, `DELETE FROM item_tags WHERE item_id = $1`, id)

		// Add new tags
		for _, tagName := range req.Tags {
			tagName = strings.TrimSpace(tagName)
			if tagName == "" {
				continue
			}
			slug := strings.ToLower(strings.ReplaceAll(tagName, " ", "-"))

			var tagID int
			err := db.Pool.QueryRow(ctx, `
				INSERT INTO tags (name, slug, usage_count, created_at)
				VALUES ($1, $2, 1, NOW())
				ON CONFLICT (slug) DO UPDATE SET usage_count = tags.usage_count + 1
				RETURNING id
			`, tagName, slug).Scan(&tagID)
			if err != nil {
				continue
			}

			_, _ = db.Pool.Exec(ctx, `
				INSERT INTO item_tags (item_id, tag_id, created_at)
				VALUES ($1, $2, NOW())
				ON CONFLICT DO NOTHING
			`, id, tagID)
		}
	}

	return item, nil
}

// DeleteItem deletes an item by ID
func (db *DB) DeleteItem(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrItemNotFound
	}

	return nil
}

// GetItemStats returns aggregate statistics for items
func (db *DB) GetItemStats(ctx context.Context) (*models.ItemStats, error) {
	var totalItems, verifiedCount, withPrices, totalTags int

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total_items,
			COUNT(*) FILTER (WHERE verified = true) as verified_count,
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM store_prices WHERE item_id = items.id)) as with_prices,
			(SELECT COUNT(*) FROM tags) as total_tags
		FROM items
	`).Scan(&totalItems, &verifiedCount, &withPrices, &totalTags)

	if err != nil {
		return nil, err
	}

	return &models.ItemStats{
		TotalItems:    totalItems,
		VerifiedCount: verifiedCount,
		WithPrices:    withPrices,
		TotalTags:     totalTags,
	}, nil
}

// SearchItems performs a fuzzy search on items
func (db *DB) SearchItems(ctx context.Context, query string, limit int) ([]*models.Item, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, brand, size, unit, description, verified, verification_count, created_by, created_at, updated_at
		FROM items
		WHERE name ILIKE $1 OR brand ILIKE $1
		ORDER BY
			CASE WHEN name ILIKE $2 || '%' THEN 0 ELSE 1 END,
			name
		LIMIT $3
	`, "%"+query+"%", query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.Item
	for rows.Next() {
		i := &models.Item{}
		if err := rows.Scan(&i.ID, &i.Name, &i.Brand, &i.Size, &i.Unit, &i.Description,
			&i.Verified, &i.VerificationCount, &i.CreatedBy, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}

	return items, nil
}

// ListTags returns all tags
func (db *DB) ListTags(ctx context.Context) ([]*models.Tag, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, slug, usage_count, created_at
		FROM tags
		ORDER BY usage_count DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		t := &models.Tag{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.UsageCount, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	return tags, nil
}
