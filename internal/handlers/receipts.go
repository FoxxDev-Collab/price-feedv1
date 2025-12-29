package handlers

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/config"
	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/middleware"
	"github.com/foxxcyber/price-feed/internal/models"
	"github.com/foxxcyber/price-feed/internal/services"
)

// ReceiptHandler handles receipt-related endpoints
type ReceiptHandler struct {
	db      *database.DB
	cfg     *config.Config
	storage *services.StorageService
	ocr     *services.OCRService
	parser  *services.ReceiptParser
	matcher *services.ItemMatcher
}

// NewReceiptHandler creates a new receipt handler
func NewReceiptHandler(
	db *database.DB,
	cfg *config.Config,
	storage *services.StorageService,
	ocr *services.OCRService,
	parser *services.ReceiptParser,
	matcher *services.ItemMatcher,
) *ReceiptHandler {
	return &ReceiptHandler{
		db:      db,
		cfg:     cfg,
		storage: storage,
		ocr:     ocr,
		parser:  parser,
		matcher: matcher,
	}
}

// UploadReceipt handles receipt image upload and processing
func (h *ReceiptHandler) UploadReceipt(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	// Get the uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "image file is required")
	}

	// Validate file type
	contentType := file.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		return Error(c, fiber.StatusBadRequest, "invalid image type. Supported: JPEG, PNG, WebP")
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return Error(c, fiber.StatusBadRequest, "file too large. Maximum size is 10MB")
	}

	// Optional store ID
	var storeID *int
	if storeIDStr := c.FormValue("store_id"); storeIDStr != "" {
		if id, err := strconv.Atoi(storeIDStr); err == nil {
			storeID = &id
		}
	}

	// Generate unique S3 key
	s3Key := generateS3Key(userID, file.Filename)

	// Open file for reading
	src, err := file.Open()
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	// Read file into memory for both S3 upload and OCR processing
	imageBytes, err := io.ReadAll(src)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to read file")
	}

	// Upload to S3
	uploadResult, err := h.storage.Upload(c.Context(), s3Key, strings.NewReader(string(imageBytes)), file.Size, contentType)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to upload image")
	}

	// Create receipt record
	receipt, err := h.db.CreateReceipt(c.Context(), &models.CreateReceiptRequest{
		UserID:           userID,
		StoreID:          storeID,
		S3Bucket:         uploadResult.Bucket,
		S3Key:            s3Key,
		OriginalFilename: file.Filename,
		ContentType:      contentType,
		FileSizeBytes:    file.Size,
	})
	if err != nil {
		// Clean up S3 on failure
		if deleteErr := h.storage.Delete(c.Context(), s3Key); deleteErr != nil {
			log.Printf("Warning: Failed to clean up S3 object %s after receipt creation failure: %v", s3Key, deleteErr)
		}
		return Error(c, fiber.StatusInternalServerError, "failed to create receipt record")
	}

	// Update status to processing
	if err := h.db.UpdateReceiptStatus(c.Context(), receipt.ID, models.ReceiptStatusProcessing, nil, nil); err != nil {
		log.Printf("Warning: Failed to update receipt %d status to processing: %v", receipt.ID, err)
	}

	// Process with OCR
	ocrResult, err := h.ocr.ProcessImage(imageBytes)
	if err != nil {
		errMsg := err.Error()
		if statusErr := h.db.UpdateReceiptStatus(c.Context(), receipt.ID, models.ReceiptStatusFailed, nil, &errMsg); statusErr != nil {
			log.Printf("Warning: Failed to update receipt %d status to failed: %v", receipt.ID, statusErr)
		}
		return Error(c, fiber.StatusInternalServerError, "OCR processing failed")
	}

	// Parse the OCR text
	parsed, err := h.parser.Parse(ocrResult.Text)
	if err != nil {
		errMsg := err.Error()
		if statusErr := h.db.UpdateReceiptStatus(c.Context(), receipt.ID, models.ReceiptStatusFailed, &ocrResult.Text, &errMsg); statusErr != nil {
			log.Printf("Warning: Failed to update receipt %d status to failed: %v", receipt.ID, statusErr)
		}
		return Error(c, fiber.StatusInternalServerError, "failed to parse receipt")
	}

	// Update receipt with OCR text and metadata
	if err := h.db.UpdateReceiptStatus(c.Context(), receipt.ID, models.ReceiptStatusCompleted, &ocrResult.Text, nil); err != nil {
		log.Printf("Warning: Failed to update receipt %d status to completed: %v", receipt.ID, err)
	}
	if err := h.db.UpdateReceiptMetadata(c.Context(), receipt.ID, parsed.Date, parsed.Total); err != nil {
		log.Printf("Warning: Failed to update receipt %d metadata: %v", receipt.ID, err)
	}

	// Match items and save to database
	matched, err := h.matcher.MatchReceiptItems(c.Context(), parsed.Items)
	if err != nil {
		// Continue even if matching fails
		matched = []services.MatchedReceiptItem{}
	}

	// Create receipt items
	for _, item := range matched {
		var matchedItemID *int
		var matchConfidence *float64
		matchStatus := models.MatchStatusPending

		if item.BestMatch != nil {
			matchedItemID = &item.BestMatch.ItemID
			matchConfidence = &item.BestMatch.Confidence
			matchStatus = models.MatchStatusMatched
		}

		_, err := h.db.CreateReceiptItem(c.Context(), &models.CreateReceiptItemRequest{
			ReceiptID:         receipt.ID,
			RawText:           item.ParsedItem.RawText,
			ExtractedName:     &item.ParsedItem.Name,
			ExtractedPrice:    &item.ParsedItem.Price,
			ExtractedQuantity: item.ParsedItem.Quantity,
			MatchedItemID:     matchedItemID,
			MatchConfidence:   matchConfidence,
			MatchStatus:       matchStatus,
			LineNumber:        item.ParsedItem.LineNumber,
		})
		if err != nil {
			// Continue even if individual item creation fails
			continue
		}
	}

	// Get the complete receipt with items
	fullReceipt, err := h.db.GetReceiptByID(c.Context(), receipt.ID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to retrieve receipt")
	}

	// Generate presigned URL for the image
	imageURL, _ := h.storage.GetPresignedURL(c.Context(), s3Key, 1*time.Hour)
	fullReceipt.ImageURL = &imageURL

	// Add suggestions to items
	for i := range fullReceipt.Items {
		if fullReceipt.Items[i].ExtractedName != nil {
			suggestions, _ := h.matcher.FindMatches(c.Context(), *fullReceipt.Items[i].ExtractedName, 5)
			for _, s := range suggestions {
				fullReceipt.Items[i].Suggestions = append(fullReceipt.Items[i].Suggestions, models.ItemSuggestion{
					ItemID:     s.ItemID,
					Name:       s.Name,
					Brand:      s.Brand,
					Confidence: s.Confidence,
					MatchType:  s.MatchType,
				})
			}
		}
	}

	return Success(c, fullReceipt)
}

// ListReceipts returns a paginated list of user's receipts
func (h *ReceiptHandler) ListReceipts(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	params := &models.ReceiptListParams{
		UserID: userID,
		Limit:  c.QueryInt("limit", 20),
		Offset: c.QueryInt("offset", 0),
	}

	if status := c.Query("status"); status != "" {
		params.Status = &status
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	receipts, total, err := h.db.ListReceipts(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list receipts")
	}

	return SuccessWithMeta(c, receipts, total, params.Limit, params.Offset)
}

// GetReceipt returns a single receipt with items
func (h *ReceiptHandler) GetReceipt(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid receipt ID")
	}

	receipt, err := h.db.GetReceiptByID(c.Context(), id)
	if err != nil {
		if err == database.ErrReceiptNotFound {
			return Error(c, fiber.StatusNotFound, "receipt not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get receipt")
	}

	// Check ownership
	if receipt.UserID != userID {
		return Error(c, fiber.StatusForbidden, "access denied")
	}

	// Generate presigned URL
	imageURL, _ := h.storage.GetPresignedURL(c.Context(), receipt.S3Key, 1*time.Hour)
	receipt.ImageURL = &imageURL

	// Add suggestions to items
	for i := range receipt.Items {
		if receipt.Items[i].ExtractedName != nil {
			suggestions, _ := h.matcher.FindMatches(c.Context(), *receipt.Items[i].ExtractedName, 5)
			for _, s := range suggestions {
				receipt.Items[i].Suggestions = append(receipt.Items[i].Suggestions, models.ItemSuggestion{
					ItemID:     s.ItemID,
					Name:       s.Name,
					Brand:      s.Brand,
					Confidence: s.Confidence,
					MatchType:  s.MatchType,
				})
			}
		}
	}

	return Success(c, receipt)
}

// UpdateReceiptItem updates a single receipt item
func (h *ReceiptHandler) UpdateReceiptItem(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	receiptID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid receipt ID")
	}

	itemID, err := strconv.Atoi(c.Params("itemId"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item ID")
	}

	// Verify receipt ownership
	receipt, err := h.db.GetReceiptByID(c.Context(), receiptID)
	if err != nil {
		if err == database.ErrReceiptNotFound {
			return Error(c, fiber.StatusNotFound, "receipt not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get receipt")
	}

	if receipt.UserID != userID {
		return Error(c, fiber.StatusForbidden, "access denied")
	}

	var req models.UpdateReceiptItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	item, err := h.db.UpdateReceiptItem(c.Context(), itemID, &req)
	if err != nil {
		if err == database.ErrReceiptItemNotFound {
			return Error(c, fiber.StatusNotFound, "receipt item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update item")
	}

	return Success(c, item)
}

// ConfirmReceipt confirms all items and creates prices
func (h *ReceiptHandler) ConfirmReceipt(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid receipt ID")
	}

	// Verify receipt ownership
	receipt, err := h.db.GetReceiptByID(c.Context(), id)
	if err != nil {
		if err == database.ErrReceiptNotFound {
			return Error(c, fiber.StatusNotFound, "receipt not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get receipt")
	}

	if receipt.UserID != userID {
		return Error(c, fiber.StatusForbidden, "access denied")
	}

	if receipt.Status == models.ReceiptStatusConfirmed {
		return Error(c, fiber.StatusBadRequest, "receipt already confirmed")
	}

	var req models.ConfirmReceiptRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.StoreID == 0 {
		return Error(c, fiber.StatusBadRequest, "store_id is required")
	}

	// Confirm receipt and create prices
	err = h.db.ConfirmReceipt(c.Context(), id, req.StoreID, userID, req.Items)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to confirm receipt")
	}

	// Get updated receipt
	updatedReceipt, err := h.db.GetReceiptByID(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get updated receipt")
	}

	return Success(c, updatedReceipt)
}

// DeleteReceipt deletes a receipt and its image
func (h *ReceiptHandler) DeleteReceipt(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid receipt ID")
	}

	// Get receipt to verify ownership and get S3 key
	receipt, err := h.db.GetReceiptByID(c.Context(), id)
	if err != nil {
		if err == database.ErrReceiptNotFound {
			return Error(c, fiber.StatusNotFound, "receipt not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get receipt")
	}

	if receipt.UserID != userID {
		return Error(c, fiber.StatusForbidden, "access denied")
	}

	// Delete from S3 (log error but continue with database deletion)
	if err := h.storage.Delete(c.Context(), receipt.S3Key); err != nil {
		log.Printf("Warning: Failed to delete S3 object %s for receipt %d: %v", receipt.S3Key, id, err)
	}

	// Delete from database
	err = h.db.DeleteReceipt(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to delete receipt")
	}

	return Success(c, fiber.Map{"deleted": true})
}

// GetReceiptImage returns a presigned URL for the receipt image
func (h *ReceiptHandler) GetReceiptImage(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid receipt ID")
	}

	receipt, err := h.db.GetReceiptByID(c.Context(), id)
	if err != nil {
		if err == database.ErrReceiptNotFound {
			return Error(c, fiber.StatusNotFound, "receipt not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get receipt")
	}

	if receipt.UserID != userID {
		return Error(c, fiber.StatusForbidden, "access denied")
	}

	// Generate presigned URL (valid for 1 hour)
	url, err := h.storage.GetPresignedURL(c.Context(), receipt.S3Key, 1*time.Hour)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate image URL")
	}

	return Success(c, fiber.Map{"url": url})
}

// isValidImageType checks if the content type is a valid image
func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/webp",
	}

	for _, t := range validTypes {
		if strings.EqualFold(contentType, t) {
			return true
		}
	}
	return false
}

// generateS3Key generates a unique S3 key for a receipt image
func generateS3Key(userID int, filename string) string {
	timestamp := time.Now().UnixNano()
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = strings.ToLower(filename[idx:])
	}
	if ext == "" {
		ext = ".jpg"
	}
	return fmt.Sprintf("receipts/%d/%d%s", userID, timestamp, ext)
}
