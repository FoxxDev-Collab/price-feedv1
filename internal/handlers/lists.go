package handlers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/foxxcyber/price-feed/internal/database"
	"github.com/foxxcyber/price-feed/internal/models"
	"github.com/foxxcyber/price-feed/internal/services"
)

// getUserID extracts user ID from context using the middleware helper
func getUserID(c *fiber.Ctx) (int, error) {
	userID, ok := c.Locals("user_id").(int)
	if !ok || userID == 0 {
		return 0, errors.New("user not authenticated")
	}
	return userID, nil
}

// ListShoppingLists returns all shopping lists for the current user
func (h *Handler) ListShoppingLists(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	params := &models.ListListParams{
		Limit:  c.QueryInt("limit", 50),
		Offset: c.QueryInt("offset", 0),
		UserID: userID,
		Status: models.ListStatus(c.Query("status")), // Optional: "active" or "completed"
	}

	// Validate limits
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 50
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	lists, total, err := h.db.ListShoppingLists(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to list shopping lists")
	}

	return SuccessWithMeta(c, lists, total, params.Limit, params.Offset)
}

// GetShoppingList returns a single shopping list with items
func (h *Handler) GetShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	list, err := h.db.GetShoppingListByID(c.Context(), id, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get shopping list")
	}

	return Success(c, list)
}

// CreateShoppingList creates a new shopping list
func (h *Handler) CreateShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	var req models.CreateListRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.Name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}

	list, err := h.db.CreateShoppingList(c.Context(), &req, userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to create shopping list")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    list,
	})
}

// UpdateShoppingList updates a shopping list
func (h *Handler) UpdateShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.UpdateListRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	list, err := h.db.UpdateShoppingList(c.Context(), id, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update shopping list")
	}

	return Success(c, list)
}

// DeleteShoppingList deletes a shopping list
func (h *Handler) DeleteShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	if err := h.db.DeleteShoppingList(c.Context(), id, userID); err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to delete shopping list")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "shopping list deleted successfully",
	})
}

// AddItemToList adds an item to a shopping list
func (h *Handler) AddItemToList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.AddListItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	// Validate required fields
	if req.ItemID == 0 {
		return Error(c, fiber.StatusBadRequest, "item_id is required")
	}
	if req.Quantity < 1 {
		req.Quantity = 1
	}

	item, err := h.db.AddItemToList(c.Context(), listID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to add item to list")
	}

	return c.Status(fiber.StatusCreated).JSON(APIResponse{
		Success: true,
		Data:    item,
	})
}

// UpdateListItem updates the quantity of an item in a list
func (h *Handler) UpdateListItem(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	var req models.UpdateListItemRequest
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Quantity < 1 {
		return Error(c, fiber.StatusBadRequest, "quantity must be at least 1")
	}

	item, err := h.db.UpdateListItem(c.Context(), listID, itemID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrListItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found in list")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to update list item")
	}

	return Success(c, item)
}

// RemoveItemFromList removes an item from a shopping list
func (h *Handler) RemoveItemFromList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	itemID, err := strconv.Atoi(c.Params("item_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	if err := h.db.RemoveItemFromList(c.Context(), listID, itemID, userID); err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrListItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found in list")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to remove item from list")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "item removed from list successfully",
	})
}

// BuildShoppingPlan generates an optimized shopping plan for a list
func (h *Handler) BuildShoppingPlan(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	// Get user's region if available
	var regionID *int
	if user, err := h.db.GetUserByID(c.Context(), userID); err == nil && user.RegionID != nil {
		regionID = user.RegionID
	}

	plan, err := h.db.BuildShoppingPlan(c.Context(), listID, userID, regionID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		if err.Error() == "shopping list is empty" {
			return Error(c, fiber.StatusBadRequest, "shopping list is empty")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to build shopping plan")
	}

	return Success(c, plan)
}

// GetPriceComparison returns a price comparison grid
func (h *Handler) GetPriceComparison(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	// Parse store IDs (required)
	storeIDsParam := c.Query("store_ids")
	if storeIDsParam == "" {
		return Error(c, fiber.StatusBadRequest, "store_ids is required")
	}

	var storeIDs []int
	for _, idStr := range strings.Split(storeIDsParam, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return Error(c, fiber.StatusBadRequest, "invalid store_ids format")
		}
		storeIDs = append(storeIDs, id)
	}

	if len(storeIDs) < 1 || len(storeIDs) > 5 {
		return Error(c, fiber.StatusBadRequest, "select 1-5 stores to compare")
	}

	// Parse item IDs (optional)
	var itemIDs []int
	if itemIDsParam := c.Query("item_ids"); itemIDsParam != "" {
		for _, idStr := range strings.Split(itemIDsParam, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				return Error(c, fiber.StatusBadRequest, "invalid item_ids format")
			}
			itemIDs = append(itemIDs, id)
		}
	}

	// Get user's region if available
	var regionID *int
	if user, err := h.db.GetUserByID(c.Context(), userID); err == nil && user.RegionID != nil {
		regionID = user.RegionID
	}

	params := &models.CompareParams{
		StoreIDs: storeIDs,
		ItemIDs:  itemIDs,
		RegionID: regionID,
		UserID:   &userID,
	}

	comparison, err := h.db.GetPriceComparison(c.Context(), params)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get price comparison")
	}

	return Success(c, comparison)
}

// DuplicateShoppingList creates a copy of an existing shopping list
func (h *Handler) DuplicateShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" {
		req.Name = "Copy of list"
	}

	newList, err := h.db.DuplicateShoppingList(c.Context(), listID, userID, req.Name)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to duplicate shopping list")
	}

	return Success(c, newList)
}

// CompleteShoppingList marks a shopping list as completed with optional price confirmations
func (h *Handler) CompleteShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	var req models.CompleteListRequest
	if err := c.BodyParser(&req); err != nil {
		// Body is optional, so just use empty request
		req = models.CompleteListRequest{}
	}

	list, err := h.db.CompleteShoppingList(c.Context(), listID, userID, &req)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		if err.Error() == "list is already completed" {
			return Error(c, fiber.StatusBadRequest, "list is already completed")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to complete shopping list")
	}

	return Success(c, list)
}

// ReopenShoppingList marks a completed list as active again
func (h *Handler) ReopenShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	list, err := h.db.ReopenShoppingList(c.Context(), listID, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		if errors.Is(err, database.ErrNotListOwner) {
			return Error(c, fiber.StatusForbidden, "you do not own this list")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to reopen shopping list")
	}

	return Success(c, list)
}

// GenerateShareLink creates a shareable link for a shopping list
func (h *Handler) GenerateShareLink(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	// Verify ownership first
	list, err := h.db.GetShoppingListByID(c.Context(), listID, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get shopping list")
	}
	if list.UserID != userID {
		return Error(c, fiber.StatusForbidden, "you do not own this list")
	}

	// Default 7 day expiration
	expiresIn := 7 * 24 * time.Hour

	token, err := h.db.CreateShareToken(c.Context(), listID, userID, expiresIn)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to generate share link")
	}

	// Build full share URL
	baseURL := c.Protocol() + "://" + c.Hostname()
	shareURL := baseURL + "/share/" + token

	return Success(c, fiber.Map{
		"token":      token,
		"share_url":  shareURL,
		"expires_at": time.Now().Add(expiresIn),
	})
}

// GetSharedList retrieves a shopping list by share token (public endpoint)
func (h *Handler) GetSharedList(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return Error(c, fiber.StatusBadRequest, "share token required")
	}

	list, err := h.db.GetShoppingListByShareToken(c.Context(), token)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shared list not found or expired")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get shared list")
	}

	return Success(c, list)
}

// ToggleSharedListItem toggles the checked status of an item on a shared list
func (h *Handler) ToggleSharedListItem(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return Error(c, fiber.StatusBadRequest, "share token required")
	}

	itemID, err := strconv.Atoi(c.Params("itemId"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid item id")
	}

	// Toggle the item - this verifies the token internally
	item, err := h.db.ToggleListItemChecked(c.Context(), token, itemID)
	if err != nil {
		if errors.Is(err, database.ErrShareTokenInvalid) {
			return Error(c, fiber.StatusNotFound, "shared list not found or expired")
		}
		if errors.Is(err, database.ErrListItemNotFound) {
			return Error(c, fiber.StatusNotFound, "item not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to toggle item")
	}

	return Success(c, fiber.Map{
		"toggled":    true,
		"is_checked": item.IsChecked,
	})
}

// EmailShoppingList sends the shopping list share link to the user's email
func (h *Handler) EmailShoppingList(c *fiber.Ctx) error {
	userID, err := getUserID(c)
	if err != nil {
		return Error(c, fiber.StatusUnauthorized, err.Error())
	}

	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid list id")
	}

	// Get the list with items
	list, err := h.db.GetShoppingListByID(c.Context(), listID, userID)
	if err != nil {
		if errors.Is(err, database.ErrListNotFound) {
			return Error(c, fiber.StatusNotFound, "shopping list not found")
		}
		return Error(c, fiber.StatusInternalServerError, "failed to get shopping list")
	}
	if list.UserID != userID {
		return Error(c, fiber.StatusForbidden, "you do not own this list")
	}

	// Get the user's email
	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to get user info")
	}

	// Generate or reuse share token
	var token string
	expiresIn := 7 * 24 * time.Hour
	if list.ShareToken != nil && list.ShareExpiresAt != nil && list.ShareExpiresAt.After(time.Now()) {
		token = *list.ShareToken
	} else {
		token, err = h.db.CreateShareToken(c.Context(), listID, userID, expiresIn)
		if err != nil {
			return Error(c, fiber.StatusInternalServerError, "failed to generate share link")
		}
	}

	// Build share URL
	baseURL := c.Protocol() + "://" + c.Hostname()
	shareURL := baseURL + "/share/" + token

	// Create email service and send
	emailService := services.NewEmailService(h.db, h.cfg)
	if !emailService.IsConfiguredWithContext(c.Context()) {
		return Error(c, fiber.StatusServiceUnavailable, "email service is not configured")
	}

	subject := "Your Shopping List: " + list.Name
	htmlBody := buildShoppingListEmail(list, shareURL)
	textBody := buildShoppingListEmailText(list, shareURL)

	err = emailService.SendEmail(user.Email, subject, htmlBody, textBody)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "failed to send email: "+err.Error())
	}

	return Success(c, fiber.Map{
		"message":   "Shopping list emailed successfully",
		"share_url": shareURL,
	})
}

// buildShoppingListEmailText creates the plain text email body for a shopping list
func buildShoppingListEmailText(list *models.ShoppingListWithItems, shareURL string) string {
	var items string
	for i, item := range list.Items {
		checked := "[ ]"
		if item.IsChecked {
			checked = "[x]"
		}
		items += checked + " " + item.ItemName
		if item.Quantity > 1 {
			items += " (x" + strconv.Itoa(item.Quantity) + ")"
		}
		if i < len(list.Items)-1 {
			items += "\n"
		}
	}

	return "Your Shopping List: " + list.Name + "\n\n" +
		"Items (" + strconv.Itoa(len(list.Items)) + "):\n" + items + "\n\n" +
		"Open Interactive List: " + shareURL + "\n\n" +
		"This link expires in 7 days. You can mark items as checked directly from your phone!"
}

// buildShoppingListEmail creates the HTML email body for a shopping list
func buildShoppingListEmail(list *models.ShoppingListWithItems, shareURL string) string {
	var itemsList string
	for _, item := range list.Items {
		checked := ""
		if item.IsChecked {
			checked = "✓ "
		}
		itemsList += "<li>" + checked + item.ItemName
		if item.Quantity > 1 {
			itemsList += " (x" + strconv.Itoa(item.Quantity) + ")"
		}
		itemsList += "</li>"
	}

	return `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Your Shopping List</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; background-color: #f5f5f5;">
    <div style="background-color: white; border-radius: 8px; padding: 30px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
        <h1 style="color: #333; margin-bottom: 20px;">` + list.Name + `</h1>
        
        <p style="color: #666; margin-bottom: 20px;">Here's your shopping list. Click the button below to view and interact with your list on your phone!</p>
        
        <div style="background-color: #f8f9fa; border-radius: 6px; padding: 20px; margin-bottom: 20px;">
            <h3 style="color: #333; margin-top: 0;">Items (` + strconv.Itoa(len(list.Items)) + `):</h3>
            <ul style="color: #666; padding-left: 20px;">
                ` + itemsList + `
            </ul>
        </div>
        
        <a href="` + shareURL + `" style="display: inline-block; background-color: #007bff; color: white; text-decoration: none; padding: 12px 24px; border-radius: 6px; font-weight: 500;">Open Interactive List</a>
        
        <p style="color: #999; font-size: 12px; margin-top: 30px;">This link expires in 7 days. You can mark items as checked directly from your phone!</p>
    </div>
</body>
</html>
`
}
