package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/foxxcyber/price-feed/internal/models"
)

var (
	ErrReceiptNotFound     = errors.New("receipt not found")
	ErrReceiptItemNotFound = errors.New("receipt item not found")
)

// CreateReceipt creates a new receipt record
func (db *DB) CreateReceipt(ctx context.Context, req *models.CreateReceiptRequest) (*models.Receipt, error) {
	receipt := &models.Receipt{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO receipts (user_id, store_id, s3_bucket, s3_key, original_filename, content_type, file_size_bytes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'pending')
		RETURNING id, user_id, store_id, s3_bucket, s3_key, original_filename, content_type, file_size_bytes,
		          status, ocr_text, error_message, receipt_date, receipt_total,
		          uploaded_at, processed_at, confirmed_at, expires_at, created_at, updated_at
	`, req.UserID, req.StoreID, req.S3Bucket, req.S3Key, req.OriginalFilename, req.ContentType, req.FileSizeBytes).Scan(
		&receipt.ID, &receipt.UserID, &receipt.StoreID, &receipt.S3Bucket, &receipt.S3Key,
		&receipt.OriginalFilename, &receipt.ContentType, &receipt.FileSizeBytes,
		&receipt.Status, &receipt.OCRText, &receipt.ErrorMessage, &receipt.ReceiptDate, &receipt.ReceiptTotal,
		&receipt.UploadedAt, &receipt.ProcessedAt, &receipt.ConfirmedAt, &receipt.ExpiresAt, &receipt.CreatedAt, &receipt.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return receipt, nil
}

// GetReceiptByID retrieves a receipt by ID
func (db *DB) GetReceiptByID(ctx context.Context, id int) (*models.ReceiptWithItems, error) {
	receipt := &models.ReceiptWithItems{}

	err := db.Pool.QueryRow(ctx, `
		SELECT r.id, r.user_id, r.store_id, r.s3_bucket, r.s3_key, r.original_filename, r.content_type, r.file_size_bytes,
		       r.status, r.ocr_text, r.error_message, r.receipt_date, r.receipt_total,
		       r.uploaded_at, r.processed_at, r.confirmed_at, r.expires_at, r.created_at, r.updated_at,
		       s.name as store_name
		FROM receipts r
		LEFT JOIN stores s ON r.store_id = s.id
		WHERE r.id = $1
	`, id).Scan(
		&receipt.ID, &receipt.UserID, &receipt.StoreID, &receipt.S3Bucket, &receipt.S3Key,
		&receipt.OriginalFilename, &receipt.ContentType, &receipt.FileSizeBytes,
		&receipt.Status, &receipt.OCRText, &receipt.ErrorMessage, &receipt.ReceiptDate, &receipt.ReceiptTotal,
		&receipt.UploadedAt, &receipt.ProcessedAt, &receipt.ConfirmedAt, &receipt.ExpiresAt, &receipt.CreatedAt, &receipt.UpdatedAt,
		&receipt.StoreName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReceiptNotFound
		}
		return nil, err
	}

	// Get receipt items
	items, err := db.GetReceiptItems(ctx, id)
	if err != nil {
		return nil, err
	}
	receipt.Items = items

	return receipt, nil
}

// GetReceiptItems retrieves all items for a receipt
func (db *DB) GetReceiptItems(ctx context.Context, receiptID int) ([]models.ReceiptItemWithSuggestions, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT ri.id, ri.receipt_id, ri.raw_text, ri.extracted_name, ri.extracted_price, ri.extracted_quantity,
		       ri.matched_item_id, ri.match_confidence, ri.match_status,
		       ri.confirmed_item_id, ri.confirmed_price, ri.is_confirmed, ri.created_item_id,
		       ri.line_number, ri.created_at, ri.updated_at,
		       i.name as matched_item_name
		FROM receipt_items ri
		LEFT JOIN items i ON ri.matched_item_id = i.id
		WHERE ri.receipt_id = $1
		ORDER BY ri.line_number ASC, ri.id ASC
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.ReceiptItemWithSuggestions
	for rows.Next() {
		item := models.ReceiptItemWithSuggestions{}
		err := rows.Scan(
			&item.ID, &item.ReceiptID, &item.RawText, &item.ExtractedName, &item.ExtractedPrice, &item.ExtractedQuantity,
			&item.MatchedItemID, &item.MatchConfidence, &item.MatchStatus,
			&item.ConfirmedItemID, &item.ConfirmedPrice, &item.IsConfirmed, &item.CreatedItemID,
			&item.LineNumber, &item.CreatedAt, &item.UpdatedAt,
			&item.MatchedItemName,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if items == nil {
		items = []models.ReceiptItemWithSuggestions{}
	}

	return items, nil
}

// ListReceipts returns a paginated list of receipts for a user
func (db *DB) ListReceipts(ctx context.Context, params *models.ReceiptListParams) ([]*models.ReceiptWithItems, int, error) {
	var args []interface{}
	argIndex := 1

	whereClause := "WHERE r.user_id = $1"
	args = append(args, params.UserID)
	argIndex++

	if params.Status != nil && *params.Status != "" {
		whereClause += " AND r.status = $" + string(rune('0'+argIndex))
		args = append(args, *params.Status)
		argIndex++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM receipts r " + whereClause
	err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get receipts
	query := `
		SELECT r.id, r.user_id, r.store_id, r.s3_bucket, r.s3_key, r.original_filename, r.content_type, r.file_size_bytes,
		       r.status, r.ocr_text, r.error_message, r.receipt_date, r.receipt_total,
		       r.uploaded_at, r.processed_at, r.confirmed_at, r.expires_at, r.created_at, r.updated_at,
		       s.name as store_name
		FROM receipts r
		LEFT JOIN stores s ON r.store_id = s.id
		` + whereClause + `
		ORDER BY r.uploaded_at DESC
		LIMIT $` + string(rune('0'+argIndex)) + ` OFFSET $` + string(rune('0'+argIndex+1))

	args = append(args, params.Limit, params.Offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var receipts []*models.ReceiptWithItems
	for rows.Next() {
		receipt := &models.ReceiptWithItems{}
		err := rows.Scan(
			&receipt.ID, &receipt.UserID, &receipt.StoreID, &receipt.S3Bucket, &receipt.S3Key,
			&receipt.OriginalFilename, &receipt.ContentType, &receipt.FileSizeBytes,
			&receipt.Status, &receipt.OCRText, &receipt.ErrorMessage, &receipt.ReceiptDate, &receipt.ReceiptTotal,
			&receipt.UploadedAt, &receipt.ProcessedAt, &receipt.ConfirmedAt, &receipt.ExpiresAt, &receipt.CreatedAt, &receipt.UpdatedAt,
			&receipt.StoreName,
		)
		if err != nil {
			return nil, 0, err
		}
		receipts = append(receipts, receipt)
	}

	return receipts, total, nil
}

// UpdateReceiptStatus updates the status and optionally OCR text
func (db *DB) UpdateReceiptStatus(ctx context.Context, id int, status models.ReceiptStatus, ocrText *string, errMsg *string) error {
	var processedAt *time.Time
	if status == models.ReceiptStatusCompleted || status == models.ReceiptStatusFailed {
		now := time.Now()
		processedAt = &now
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE receipts
		SET status = $2, ocr_text = COALESCE($3, ocr_text), error_message = $4, processed_at = $5, updated_at = NOW()
		WHERE id = $1
	`, id, status, ocrText, errMsg, processedAt)

	return err
}

// UpdateReceiptMetadata updates extracted metadata
func (db *DB) UpdateReceiptMetadata(ctx context.Context, id int, receiptDate *time.Time, total *float64) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE receipts
		SET receipt_date = $2, receipt_total = $3, updated_at = NOW()
		WHERE id = $1
	`, id, receiptDate, total)

	return err
}

// CreateReceiptItem creates a parsed item from a receipt
func (db *DB) CreateReceiptItem(ctx context.Context, req *models.CreateReceiptItemRequest) (*models.ReceiptItem, error) {
	item := &models.ReceiptItem{}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO receipt_items (receipt_id, raw_text, extracted_name, extracted_price, extracted_quantity,
		                          matched_item_id, match_confidence, match_status, line_number)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, receipt_id, raw_text, extracted_name, extracted_price, extracted_quantity,
		          matched_item_id, match_confidence, match_status,
		          confirmed_item_id, confirmed_price, is_confirmed, created_item_id,
		          line_number, created_at, updated_at
	`, req.ReceiptID, req.RawText, req.ExtractedName, req.ExtractedPrice, req.ExtractedQuantity,
		req.MatchedItemID, req.MatchConfidence, req.MatchStatus, req.LineNumber).Scan(
		&item.ID, &item.ReceiptID, &item.RawText, &item.ExtractedName, &item.ExtractedPrice, &item.ExtractedQuantity,
		&item.MatchedItemID, &item.MatchConfidence, &item.MatchStatus,
		&item.ConfirmedItemID, &item.ConfirmedPrice, &item.IsConfirmed, &item.CreatedItemID,
		&item.LineNumber, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return item, nil
}

// UpdateReceiptItem updates a receipt item with user confirmation
func (db *DB) UpdateReceiptItem(ctx context.Context, id int, req *models.UpdateReceiptItemRequest) (*models.ReceiptItem, error) {
	item := &models.ReceiptItem{}

	err := db.Pool.QueryRow(ctx, `
		UPDATE receipt_items
		SET confirmed_item_id = COALESCE($2, confirmed_item_id),
		    confirmed_price = COALESCE($3, confirmed_price),
		    match_status = COALESCE($4, match_status),
		    is_confirmed = COALESCE($5, is_confirmed),
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, receipt_id, raw_text, extracted_name, extracted_price, extracted_quantity,
		          matched_item_id, match_confidence, match_status,
		          confirmed_item_id, confirmed_price, is_confirmed, created_item_id,
		          line_number, created_at, updated_at
	`, id, req.ConfirmedItemID, req.ConfirmedPrice, req.MatchStatus, req.IsConfirmed).Scan(
		&item.ID, &item.ReceiptID, &item.RawText, &item.ExtractedName, &item.ExtractedPrice, &item.ExtractedQuantity,
		&item.MatchedItemID, &item.MatchConfidence, &item.MatchStatus,
		&item.ConfirmedItemID, &item.ConfirmedPrice, &item.IsConfirmed, &item.CreatedItemID,
		&item.LineNumber, &item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrReceiptItemNotFound
		}
		return nil, err
	}

	return item, nil
}

// ConfirmReceipt confirms all items and creates prices
func (db *DB) ConfirmReceipt(ctx context.Context, receiptID int, storeID int, userID int, items []models.ConfirmReceiptItemData) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Update receipt store and status
	_, err = tx.Exec(ctx, `
		UPDATE receipts
		SET store_id = $2, status = 'confirmed', confirmed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, receiptID, storeID)
	if err != nil {
		return err
	}

	// Process each item
	for _, item := range items {
		if item.Skip {
			// Mark as skipped
			_, err = tx.Exec(ctx, `
				UPDATE receipt_items SET match_status = 'skipped', is_confirmed = true, updated_at = NOW()
				WHERE id = $1
			`, item.ReceiptItemID)
			if err != nil {
				return err
			}
			continue
		}

		var itemID int
		var price float64

		if item.CreateNewItem && item.NewItemName != nil {
			// Create new item
			err = tx.QueryRow(ctx, `
				INSERT INTO items (name, created_by, created_at, updated_at)
				VALUES ($1, $2, NOW(), NOW())
				RETURNING id
			`, *item.NewItemName, userID).Scan(&itemID)
			if err != nil {
				return err
			}

			// Update receipt item with created item ID
			_, err = tx.Exec(ctx, `
				UPDATE receipt_items SET created_item_id = $2, match_status = 'new_item' WHERE id = $1
			`, item.ReceiptItemID, itemID)
			if err != nil {
				return err
			}
		} else if item.ItemID != nil {
			itemID = *item.ItemID
		} else {
			// No item to create price for
			continue
		}

		if item.Price != nil {
			price = *item.Price
		} else {
			continue
		}

		// Create or update store price
		_, err = tx.Exec(ctx, `
			INSERT INTO store_prices (store_id, item_id, price, user_id, is_shared, created_at, updated_at)
			VALUES ($1, $2, $3, $4, true, NOW(), NOW())
			ON CONFLICT (store_id, item_id) WHERE store_id = $1 AND item_id = $2
			DO UPDATE SET price = $3, user_id = $4, updated_at = NOW()
		`, storeID, itemID, price, userID)
		if err != nil {
			// If conflict handling fails, try simple insert/update
			_, err = tx.Exec(ctx, `
				INSERT INTO store_prices (store_id, item_id, price, user_id, is_shared, created_at, updated_at)
				VALUES ($1, $2, $3, $4, true, NOW(), NOW())
			`, storeID, itemID, price, userID)
			if err != nil {
				return err
			}
		}

		// Update receipt item as confirmed
		_, err = tx.Exec(ctx, `
			UPDATE receipt_items
			SET confirmed_item_id = $2, confirmed_price = $3, is_confirmed = true, match_status = 'matched', updated_at = NOW()
			WHERE id = $1
		`, item.ReceiptItemID, itemID, price)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// DeleteReceipt deletes a receipt and its items
func (db *DB) DeleteReceipt(ctx context.Context, id int) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM receipts WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrReceiptNotFound
	}

	return nil
}

// CleanupExpiredReceipts deletes receipts past their expiration date and returns S3 keys to delete
func (db *DB) CleanupExpiredReceipts(ctx context.Context) ([]string, error) {
	// Get S3 keys of expired receipts
	rows, err := db.Pool.Query(ctx, `
		SELECT s3_key FROM receipts WHERE expires_at < NOW()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}

	// Delete expired receipts
	_, err = db.Pool.Exec(ctx, `DELETE FROM receipts WHERE expires_at < NOW()`)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

// FindSimilarItems finds items similar to the given name using trigram similarity
func (db *DB) FindSimilarItems(ctx context.Context, name string, limit int) ([]models.MatchResult, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, brand, similarity(LOWER(name), LOWER($1)) as confidence
		FROM items
		WHERE similarity(LOWER(name), LOWER($1)) > 0.2
		ORDER BY confidence DESC
		LIMIT $2
	`, name, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.MatchResult
	for rows.Next() {
		var result models.MatchResult
		err := rows.Scan(&result.ItemID, &result.Name, &result.Brand, &result.Confidence)
		if err != nil {
			return nil, err
		}
		result.MatchType = "fuzzy"
		results = append(results, result)
	}

	return results, nil
}
