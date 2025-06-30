package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"store-service/internal/models"
	"store-service/internal/services"

	"github.com/gin-gonic/gin"
)

type StoreHandler struct {
	service *services.StoreService
}

func NewStoreHandler(service *services.StoreService) *StoreHandler {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
			return
		}
	} else {
		// Handle form data (legacy support)
		title := c.PostForm("title")
		description := c.PostForm("description")
		category := c.PostForm("category")
		condition := c.PostForm("condition")
		isAuction := c.PostForm("is_auction") == "true"
		
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
		
		if isAuction {
			req.PriceType = "bidding"
			if startingBid, err := strconv.ParseFloat(c.PostForm("starting_bid"), 64); err == nil {
				req.StartingBid = startingBid
			}
			if bidIncrement, err := strconv.ParseFloat(c.PostForm("bid_increment"), 64); err == nil {
				req.MinBidIncrement = bidIncrement
			}
		} else {
			req.PriceType = "fixed"
			if price, err := strconv.ParseFloat(c.PostForm("price"), 64); err == nil {
				req.FixedPrice = price
			}
		}
	}
	
	// Handle image uploads
	form, err := c.MultipartForm()
	if err == nil && form != nil && form.File["images"] != nil {
		var imageURLs []string
		
		// Create uploads directory structure
		uploadsDir := "./uploads/store"
		userDir := filepath.Join(uploadsDir, strconv.FormatUint(uint64(userID), 10))
		if err := os.MkdirAll(userDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
			return
		}
		
		for i, fileHeader := range form.File["images"] {
			if i >= 3 { // Limit to 3 images
				break
			}
			
			// Validate file size (5MB limit)
			if fileHeader.Size > 5*1024*1024 {
				continue
			}
			
			// Check file extension
			ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
			if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
				continue
			}
			
			// Open uploaded file
			file, err := fileHeader.Open()
			if err != nil {
				continue
			}
			defer file.Close()
			
			// Generate unique filename
			timestamp := time.Now().Unix()
			filename := fmt.Sprintf("%d_%d%s", timestamp, i, ext)
			filePath := filepath.Join(userDir, filename)
			
			// Create destination file
			dst, err := os.Create(filePath)
			if err != nil {
				continue
			}
			defer dst.Close()
			
			// Copy file content
			if _, err := io.Copy(dst, file); err != nil {
				continue
			}
			
			// Create URL for the uploaded file
			imageURL := fmt.Sprintf("/uploads/store/%d/%s", userID, filename)
			imageURLs = append(imageURLs, imageURL)
		}
		
		req.Images = imageURLs
	}

	item, err := h.service.CreateItem(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// GetItem retrieves a specific store item
func (h *StoreHandler) GetItem(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	item, err := h.service.GetItem(uint(id))
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
		if id, err := strconv.ParseUint(sellerID, 10, 32); err == nil {
			filter.SellerID = uint(id)
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
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req models.UpdateStoreItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.service.UpdateItem(uint(id), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteItem deletes a store item
func (h *StoreHandler) DeleteItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	err = h.service.DeleteItem(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "item deleted successfully"})
}

// PlaceBid places a bid on an item
func (h *StoreHandler) PlaceBid(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	var req models.CreateBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bid, err := h.service.PlaceBid(uint(itemID), userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bid)
}

// GetItemBids retrieves all bids for an item
func (h *StoreHandler) GetItemBids(c *gin.Context) {
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	bids, err := h.service.GetItemBids(uint(itemID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve bids"})
		return
	}

	c.JSON(http.StatusOK, bids)
}

// AcceptBid accepts a bid for an item
func (h *StoreHandler) AcceptBid(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	bidID, err := strconv.ParseUint(c.Param("bidId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bid id"})
		return
	}

	err = h.service.AcceptBid(uint(itemID), uint(bidID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "bid accepted successfully"})
}

// PurchaseItem purchases a fixed-price item
func (h *StoreHandler) PurchaseItem(c *gin.Context) {
	userID := c.GetUint("userID")
	
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid item id"})
		return
	}

	err = h.service.PurchaseItem(uint(itemID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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