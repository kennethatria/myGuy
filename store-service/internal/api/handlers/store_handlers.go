package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"store-service/internal/media"
	"store-service/internal/models"
	"store-service/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StoreHandler struct {
	service services.StoreServiceInterface
}

func NewStoreHandler(service services.StoreServiceInterface) *StoreHandler {
	return &StoreHandler{service: service}
}

// CreateItem creates a new store item
func (h *StoreHandler) CreateItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	var req models.CreateStoreItemRequest
	
	// Check Content-Type and parse accordingly
	contentType := c.GetHeader("Content-Type")
	
	if strings.Contains(contentType, "application/json") {
		// Handle JSON request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
	} else {
		// Handle form data (legacy support)
		title := c.PostForm("title")
		description := c.PostForm("description")
		category := c.PostForm("category")
		condition := c.PostForm("condition")
		isAuction := c.PostForm("price_type") == "bidding" || c.PostForm("is_auction") == "true"
		
		if title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
			return
		}
		
		req = models.CreateStoreItemRequest{
			Title:       title,
			Description: description,
			Category:    category,
			Condition:   condition,
		}
		if requestID, err := parseID(c.PostForm("request_id")); err == nil {
			req.RequestID = &requestID
		}
		req.Lat, req.Lng = formFloat(c, "lat"), formFloat(c, "lng")
		
		if isAuction {
			req.PriceType = "bidding"
			if startingBid, err := strconv.ParseFloat(c.PostForm("starting_bid"), 64); err == nil {
				req.StartingBid = startingBid
			}
			// Try both field names for backward compatibility
			if bidIncrement, err := strconv.ParseFloat(c.PostForm("min_bid_increment"), 64); err == nil {
				req.MinBidIncrement = bidIncrement
			} else if bidIncrement, err := strconv.ParseFloat(c.PostForm("bid_increment"), 64); err == nil {
				req.MinBidIncrement = bidIncrement
			}
		} else {
			req.PriceType = "fixed"
			// Try both field names for backward compatibility
			if price, err := strconv.ParseFloat(c.PostForm("fixed_price"), 64); err == nil {
				req.FixedPrice = price
			} else if price, err := strconv.ParseFloat(c.PostForm("price"), 64); err == nil {
				req.FixedPrice = price
			}
		}
	}
	
	// Photos: each is cleaned of metadata (EXIF can hold the GPS position
	// where it was taken) and stored under a random name
	if form, err := c.MultipartForm(); err == nil && form != nil {
		var imageURLs []string
		for _, fileHeader := range form.File["images"] {
			if len(imageURLs) >= 3 { // Limit to 3 images
				break
			}
			if fileHeader.Size > 5*1024*1024 { // 5MB limit
				continue
			}
			imageURL, err := saveCleanPhoto(fileHeader)
			if err != nil {
				log.Printf("photo upload by user %d skipped: %v", userID, err)
				continue
			}
			imageURLs = append(imageURLs, imageURL)
		}
		req.Images = imageURLs
	}

	item, err := h.service.CreateItem(userID, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, item)
}

// respondError answers with err's own message when it was written for
// users, 404 for a missing record, and otherwise logs it and answers with a
// generic 500 so database details never reach clients.
func respondError(c *gin.Context, status int, err error) {
	switch {
	case services.IsUserError(err):
		c.JSON(status, gin.H{"error": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		log.Printf("%s %s failed: %v", c.Request.Method, c.FullPath(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong; please try again"})
	}
}

// formFloat reads an optional number from a multipart form.
func formFloat(c *gin.Context, field string) *float64 {
	v, err := strconv.ParseFloat(c.PostForm(field), 64)
	if err != nil {
		return nil
	}
	return &v
}

// UploadsDir is where cleaned listing photos are stored and served from
// (a variable so tests can use a temporary directory).
var UploadsDir = "./uploads/store"

// saveCleanPhoto stores a metadata-free copy of an uploaded photo under a
// random name and returns its URL path. The format is judged from the
// file's content, not its name.
func saveCleanPhoto(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	cleaned := &bytes.Buffer{}
	ext, err := media.Clean(file, cleaned)
	if err != nil {
		return "", err
	}

	name := make([]byte, 16)
	if _, err := rand.Read(name); err != nil {
		return "", err
	}
	filename := hex.EncodeToString(name) + ext
	if err := os.MkdirAll(UploadsDir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(UploadsDir, filename), cleaned.Bytes(), 0644); err != nil {
		return "", err
	}
	return "/uploads/store/" + filename, nil
}

// GetItem retrieves a specific store item
func (h *StoreHandler) GetItem(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.service.GetItem(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}

	c.JSON(http.StatusOK, item)
}

// GetItems retrieves all store items with filtering
func (h *StoreHandler) GetItems(c *gin.Context) {
	filter := models.StoreItemFilter{
		Search:    c.Query("search"),
		Category:  c.Query("category"),
		PriceType: c.Query("price_type"),
		Condition: c.Query("condition"),
		Status:    c.Query("status"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Page:      1,
		PerPage:   20,
	}

	if minPrice := c.Query("min_price"); minPrice != "" {
		if price, err := strconv.ParseFloat(minPrice, 64); err == nil {
			filter.MinPrice = price
		}
	}

	if maxPrice := c.Query("max_price"); maxPrice != "" {
		if price, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			filter.MaxPrice = price
		}
	}

	if sellerID := c.Query("seller_id"); sellerID != "" {
		if id, err := parseID(sellerID); err == nil {
			filter.SellerID = id
		}
	}

	if sellerID := c.Query("exclude_seller_id"); sellerID != "" {
		if id, err := parseID(sellerID); err == nil {
			filter.ExcludeSellerID = id
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filter.Page = p
		}
	}

	if perPage := c.Query("per_page"); perPage != "" {
		if pp, err := strconv.Atoi(perPage); err == nil && pp > 0 {
			filter.PerPage = pp
		}
	}

	items, total, err := h.service.GetItems(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": items,
		"total": total,
		"page":  filter.Page,
		"per_page": filter.PerPage,
	})
}

// UpdateItem updates a store item
func (h *StoreHandler) UpdateItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req models.UpdateStoreItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	item, err := h.service.UpdateItem(id, userID, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteItem deletes a store item
func (h *StoreHandler) DeleteItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	err = h.service.DeleteItem(id, userID)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item deleted successfully"})
}

// RepostItem puts the seller's expired listing back on the board
func (h *StoreHandler) RepostItem(c *gin.Context) {
	userID := c.GetUint("userID")

	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.service.RepostItem(id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
			return
		}
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, item)
}

// PlaceBid places a bid on an item
func (h *StoreHandler) PlaceBid(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req models.CreateBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	bid, err := h.service.PlaceBid(itemID, userID, req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, bid)
}

// GetItemBids retrieves all bids for an item
func (h *StoreHandler) GetItemBids(c *gin.Context) {
	itemID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	bids, err := h.service.GetItemBids(itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve bids"})
		return
	}

	c.JSON(http.StatusOK, bids)
}

// AcceptBid accepts a bid for an item
func (h *StoreHandler) AcceptBid(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	bidID, err := parseID(c.Param("bidId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bid id"})
		return
	}

	err = h.service.AcceptBid(itemID, bidID, userID)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bid accepted successfully"})
}

// PurchaseItem purchases a fixed-price item
func (h *StoreHandler) PurchaseItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	err = h.service.PurchaseItem(itemID, userID)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item purchased successfully"})
}

// GetUserListings retrieves all items listed by a user
func (h *StoreHandler) GetUserListings(c *gin.Context) {
	userID := c.GetUint("userID")
	
	items, err := h.service.GetUserListings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve listings"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetUserPurchases retrieves all items purchased by a user
func (h *StoreHandler) GetUserPurchases(c *gin.Context) {
	userID := c.GetUint("userID")
	
	items, err := h.service.GetUserPurchases(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve purchases"})
		return
	}

	c.JSON(http.StatusOK, items)
}

// GetUserBids retrieves all bids placed by a user
func (h *StoreHandler) GetUserBids(c *gin.Context) {
	userID := c.GetUint("userID")
	
	bids, err := h.service.GetUserBids(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve bids"})
		return
	}

	c.JSON(http.StatusOK, bids)
}

// Booking Request Handlers

// CreateBookingRequest creates a new booking request for an item
func (h *StoreHandler) CreateBookingRequest(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemIDStr := c.Param("id")
	itemID, err := parseID(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	var req models.CreateBookingRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	bookingRequest, err := h.service.CreateBookingRequest(itemID, userID, req.Message)
	if err != nil {
		if err.Error() == "you already have a booking request for this item" {
			respondError(c, http.StatusConflict, err)
			return
		}
		if err.Error() == "cannot book your own item" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "item is not available for booking" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking request"})
		return
	}

	c.JSON(http.StatusCreated, bookingRequest)
}

// GetBookingRequest retrieves a booking request for an item
func (h *StoreHandler) GetBookingRequest(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemIDStr := c.Param("id")
	itemID, err := parseID(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	bookingRequest, err := h.service.GetBookingRequestByItem(itemID, userID)
	if err != nil {
		// Check if it's a GORM "record not found" error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Return 200 with null data when no booking request exists
			c.JSON(http.StatusOK, gin.H{"booking_request": nil})
			return
		}
		// Return 404 for other errors (like item not found)
		respondError(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"booking_request": bookingRequest})
}

// GetAllBookingRequests retrieves all booking requests for an item (owner only)
func (h *StoreHandler) GetAllBookingRequests(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemIDStr := c.Param("id")
	itemID, err := parseID(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item ID"})
		return
	}

	bookingRequests, err := h.service.GetAllBookingRequestsByItem(itemID, userID)
	if err != nil {
		if err.Error() == "unauthorized: you are not the owner of this item" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		respondError(c, http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"booking_requests": bookingRequests})
}

// ApproveBookingRequest approves a booking request
func (h *StoreHandler) ApproveBookingRequest(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	booking, err := h.service.ApproveBookingRequest(requestID, userID)
	if err != nil {
		if err.Error() == "unauthorized: you are not the owner of this item" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "booking request is not pending" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to approve booking request"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// RejectBookingRequest rejects a booking request
func (h *StoreHandler) RejectBookingRequest(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	booking, err := h.service.RejectBookingRequest(requestID, userID)
	if err != nil {
		if err.Error() == "unauthorized: you are not the owner of this item" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "booking request is not pending" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to reject booking request"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// GetUserBookingRequests retrieves all booking requests by a user
func (h *StoreHandler) GetUserBookingRequests(c *gin.Context) {
	userID := c.GetUint("userID")

	requests, err := h.service.GetUserBookingRequests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve booking requests"})
		return
	}

	c.JSON(http.StatusOK, requests)
}

// GetUserRatings lists the store ratings a user has received (as seller or
// buyer), for their profile.
func (h *StoreHandler) GetUserRatings(c *gin.Context) {
	userID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	ratings, err := h.service.GetUserRatings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve ratings"})
		return
	}

	c.JSON(http.StatusOK, ratings)
}

// ConfirmItemReceived allows buyer to confirm they received the item
func (h *StoreHandler) ConfirmItemReceived(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	booking, err := h.service.ConfirmItemReceived(requestID, userID)
	if err != nil {
		if err.Error() == "booking request not found" {
			respondError(c, http.StatusNotFound, err)
			return
		}
		if err.Error() == "only the buyer can confirm receipt" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "booking must be approved before confirming receipt" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm item received"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// ConfirmDelivery allows seller to confirm delivery is complete
func (h *StoreHandler) ConfirmDelivery(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	booking, err := h.service.ConfirmDelivery(requestID, userID)
	if err != nil {
		if err.Error() == "booking request not found" {
			respondError(c, http.StatusNotFound, err)
			return
		}
		if err.Error() == "only the seller can confirm delivery" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "buyer must confirm receipt before seller can confirm delivery" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm delivery"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// SubmitBuyerRating allows buyer to rate the seller
func (h *StoreHandler) SubmitBuyerRating(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	var req models.SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	booking, err := h.service.SubmitBuyerRating(requestID, userID, req.Rating, req.Review)
	if err != nil {
		if err.Error() == "booking request not found" {
			respondError(c, http.StatusNotFound, err)
			return
		}
		if err.Error() == "only the buyer can rate the seller" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "booking must be completed before rating" || err.Error() == "buyer has already rated this transaction" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit rating"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// SubmitSellerRating allows seller to rate the buyer
func (h *StoreHandler) SubmitSellerRating(c *gin.Context) {
	userID := c.GetUint("userID")

	requestIDStr := c.Param("requestId")
	requestID, err := parseID(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request ID"})
		return
	}

	var req models.SubmitRatingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	booking, err := h.service.SubmitSellerRating(requestID, userID, req.Rating, req.Review)
	if err != nil {
		if err.Error() == "booking request not found" {
			respondError(c, http.StatusNotFound, err)
			return
		}
		if err.Error() == "only the seller can rate the buyer" {
			respondError(c, http.StatusForbidden, err)
			return
		}
		if err.Error() == "booking must be completed before rating" || err.Error() == "seller has already rated this transaction" {
			respondError(c, http.StatusBadRequest, err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit rating"})
		return
	}

	c.JSON(http.StatusOK, booking)
}

// parseID parses a database id from a path or query value. 32 bits is far
// more than any table holds and fits uint on every platform; negative and
// oversized values are rejected rather than wrapped.
func parseID(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}
