package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"store-service/internal/api/handlers"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"store-service/internal/services"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIntegrationTestDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.Bid{}, &models.BookingRequest{}, &models.User{})
	if err != nil {
		return nil, err
	}

	// Create test users
	users := []models.User{
		{ID: 1, Username: "seller1", Email: "seller1@example.com", Name: "Seller One"},
		{ID: 2, Username: "buyer1", Email: "buyer1@example.com", Name: "Buyer One"},
		{ID: 3, Username: "bidder1", Email: "bidder1@example.com", Name: "Bidder One"},
	}

	for _, user := range users {
		db.Create(&user)
	}

	return db, nil
}

func setupIntegrationTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)

	// Initialize repositories
	itemRepo := repositories.NewStoreItemRepository(db)
	bidRepo := repositories.NewBidRepository(db)
	bookingRepo := repositories.NewBookingRequestRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Initialize service
	storeService := services.NewStoreService(db, itemRepo, bidRepo, bookingRepo, userRepo)

	// Initialize handler
	storeHandler := handlers.NewStoreHandler(storeService)

	// Setup router
	router := gin.New()

	// Auth middleware mock
	router.Use(func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "1" // Default to user 1
		}

		switch userID {
		case "1":
			c.Set("userID", uint(1))
			c.Set("username", "seller1")
			c.Set("userEmail", "seller1@example.com")
			c.Set("userName", "Seller One")
		case "2":
			c.Set("userID", uint(2))
			c.Set("username", "buyer1")
			c.Set("userEmail", "buyer1@example.com")
			c.Set("userName", "Buyer One")
		case "3":
			c.Set("userID", uint(3))
			c.Set("username", "bidder1")
			c.Set("userEmail", "bidder1@example.com")
			c.Set("userName", "Bidder One")
		default:
			c.Set("userID", uint(1))
			c.Set("username", "seller1")
			c.Set("userEmail", "seller1@example.com")
			c.Set("userName", "Seller One")
		}
		c.Next()
	})

	api := router.Group("/api/v1")
	{
		api.POST("/items", storeHandler.CreateItem)
		api.GET("/items/:id", storeHandler.GetItem)
		api.GET("/items", storeHandler.GetItems)
		api.PUT("/items/:id", storeHandler.UpdateItem)
		api.DELETE("/items/:id", storeHandler.DeleteItem)
		api.POST("/items/:id/bids", storeHandler.PlaceBid)
		api.GET("/items/:id/bids", storeHandler.GetItemBids)
		api.POST("/items/:id/bids/:bidId/accept", storeHandler.AcceptBid)
		api.POST("/items/:id/purchase", storeHandler.PurchaseItem)
		api.GET("/user/listings", storeHandler.GetUserListings)
		api.GET("/user/purchases", storeHandler.GetUserPurchases)
		api.GET("/user/bids", storeHandler.GetUserBids)
		api.POST("/items/:id/booking-request", storeHandler.CreateBookingRequest)
		api.GET("/items/:id/booking-request", storeHandler.GetBookingRequest)
		api.POST("/booking-requests/:requestId/approve", storeHandler.ApproveBookingRequest)
		api.POST("/booking-requests/:requestId/reject", storeHandler.RejectBookingRequest)
		api.GET("/user/booking-requests", storeHandler.GetUserBookingRequests)
	}

	return router
}

func TestIntegration_ItemLifecycle(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	t.Run("Create fixed price item", func(t *testing.T) {
		req := models.CreateStoreItemRequest{
			Title:       "iPhone 15 Pro",
			Description: "Brand new iPhone 15 Pro in pristine condition",
			PriceType:   "fixed",
			FixedPrice:  999.99,
			Category:    "electronics",
			Condition:   "new",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, req.Title, response.Title)
		assert.Equal(t, req.PriceType, response.PriceType)
		assert.Equal(t, req.FixedPrice, response.FixedPrice)
		assert.Equal(t, uint(1), response.SellerID)

		itemID = response.ID
	})

	t.Run("Get created item", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d", itemID), nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, itemID, response.ID)
		assert.Equal(t, "iPhone 15 Pro", response.Title)
	})

	t.Run("Update item", func(t *testing.T) {
		updateReq := models.UpdateStoreItemRequest{
			Title:       "iPhone 15 Pro (Updated)",
			Description: "Updated description with more details",
		}

		jsonData, _ := json.Marshal(updateReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/items/%d", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, updateReq.Title, response.Title)
		assert.Equal(t, updateReq.Description, response.Description)
	})

	t.Run("Purchase item", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/purchase", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Different user

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Verify item is sold", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d", itemID), nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "sold", response.Status)
		assert.Equal(t, uint(2), *response.BuyerID)
	})

	t.Run("Cannot update sold item", func(t *testing.T) {
		updateReq := models.UpdateStoreItemRequest{
			Title: "Should not work",
		}

		jsonData, _ := json.Marshal(updateReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/items/%d", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIntegration_BiddingLifecycle(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	t.Run("Create auction item", func(t *testing.T) {
		bidDeadline := time.Now().Add(24 * time.Hour)
		req := models.CreateStoreItemRequest{
			Title:           "Vintage Guitar",
			Description:     "Classic acoustic guitar in excellent condition",
			PriceType:       "bidding",
			StartingBid:     500.0,
			MinBidIncrement: 25.0,
			BidDeadline:     &bidDeadline,
			Category:        "music",
			Condition:       "good",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, req.Title, response.Title)
		assert.Equal(t, req.PriceType, response.PriceType)
		assert.Equal(t, req.StartingBid, response.StartingBid)

		itemID = response.ID
	})

	var secondBidID uint

	t.Run("Place first bid", func(t *testing.T) {
		bidReq := models.CreateBidRequest{
			Amount:  525.0,
			Message: "Great looking guitar!",
		}

		jsonData, _ := json.Marshal(bidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/bids", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.Bid
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, bidReq.Amount, response.Amount)
		assert.Equal(t, bidReq.Message, response.Message)
		assert.Equal(t, uint(2), response.BidderID)
	})

	t.Run("Place higher bid", func(t *testing.T) {
		bidReq := models.CreateBidRequest{
			Amount:  575.0,
			Message: "I really want this guitar!",
		}

		jsonData, _ := json.Marshal(bidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/bids", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "3")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.Bid
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, bidReq.Amount, response.Amount)
		assert.Equal(t, uint(3), response.BidderID)

		secondBidID = response.ID
	})

	t.Run("Get item bids", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/bids", itemID), nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.Bid
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)

		// Should be ordered by amount DESC
		assert.True(t, response[0].Amount >= response[1].Amount)
	})

	t.Run("Accept winning bid", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/bids/%d/accept", itemID, secondBidID), nil)
		httpReq.Header.Set("X-User-ID", "1") // Item owner

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Verify item is sold to winning bidder", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d", itemID), nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "sold", response.Status)
		assert.Equal(t, uint(3), *response.BuyerID)
	})

	t.Run("Cannot place bid on sold item", func(t *testing.T) {
		bidReq := models.CreateBidRequest{
			Amount: 600.0,
		}

		jsonData, _ := json.Marshal(bidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/bids", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestIntegration_BookingLifecycle(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	t.Run("Create bookable item", func(t *testing.T) {
		req := models.CreateStoreItemRequest{
			Title:       "Camera Equipment",
			Description: "Professional DSLR camera with lenses",
			PriceType:   "fixed",
			FixedPrice:  200.0,
			Category:    "electronics",
			Condition:   "good",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		itemID = response.ID
	})

	var requestID uint

	t.Run("Create booking request", func(t *testing.T) {
		bookingReq := models.CreateBookingRequestRequest{
			Message: "I need this camera for a wedding shoot this weekend",
		}

		jsonData, _ := json.Marshal(bookingReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, bookingReq.Message, response.Message)
		assert.Equal(t, uint(2), response.RequesterID)
		assert.Equal(t, "pending", response.Status)

		requestID = response.ID
	})

	t.Run("Get booking request as owner", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "1") // Item owner

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, requestID, response.ID)
	})

	t.Run("Get booking request as requester", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, requestID, response.ID)
	})

	t.Run("Approve booking request", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/booking-requests/%d/approve", requestID), nil)
		httpReq.Header.Set("X-User-ID", "1") // Item owner

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Verify booking request is approved", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "approved", response.Status)
	})

	t.Run("Get user booking requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/booking-requests", nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, requestID, response[0].ID)
	})
}

func TestIntegration_BookingRequestEdgeCases(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	// Create a test item
	t.Run("Create item for edge case testing", func(t *testing.T) {
		item := models.CreateStoreItemRequest{
			Title:       "Edge Case Test Item",
			Description: "Testing edge cases",
			PriceType:   "fixed",
			FixedPrice:  50.0,
			Category:    "electronics",
			Condition:   "new",
		}

		jsonData, _ := json.Marshal(item)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1") // Seller

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		itemID = response.ID
	})

	t.Run("Get booking request when none exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Different user

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Nil(t, response["booking_request"])
	})

	t.Run("Cannot create booking request for own item", func(t *testing.T) {
		bookingReq := models.CreateBookingRequestRequest{
			Message: "I want to book my own item",
		}

		jsonData, _ := json.Marshal(bookingReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1") // Same as seller

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Duplicate booking request should fail", func(t *testing.T) {
		// First booking request
		bookingReq := models.CreateBookingRequestRequest{
			Message: "First booking request",
		}

		jsonData, _ := json.Marshal(bookingReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Second booking request from same user should fail
		bookingReq2 := models.CreateBookingRequestRequest{
			Message: "Duplicate booking request",
		}

		jsonData2, _ := json.Marshal(bookingReq2)
		w2 := httptest.NewRecorder()
		httpReq2, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), bytes.NewBuffer(jsonData2))
		httpReq2.Header.Set("Content-Type", "application/json")
		httpReq2.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w2, httpReq2)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("Get booking request after creation", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["booking_request"])

		bookingRequest := response["booking_request"].(map[string]interface{})
		assert.Equal(t, "First booking request", bookingRequest["message"])
		assert.Equal(t, "pending", bookingRequest["status"])
	})

	t.Run("Invalid item ID for booking request", func(t *testing.T) {
		bookingReq := models.CreateBookingRequestRequest{
			Message: "Invalid item booking",
		}

		jsonData, _ := json.Marshal(bookingReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/99999/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestIntegration_ItemFiltering(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	// Create multiple test items
	testItems := []models.CreateStoreItemRequest{
		{
			Title:       "Expensive Electronics",
			Description: "High-end gadget",
			PriceType:   "fixed",
			FixedPrice:  1000.0,
			Category:    "electronics",
			Condition:   "new",
		},
		{
			Title:       "Cheap Book",
			Description: "Interesting novel",
			PriceType:   "fixed",
			FixedPrice:  15.0,
			Category:    "books",
			Condition:   "good",
		},
		{
			Title:       "Electronics Auction",
			Description: "Bidding item",
			PriceType:   "bidding",
			StartingBid: 100.0,
			Category:    "electronics",
			Condition:   "fair",
		},
	}

	t.Run("Create test items", func(t *testing.T) {
		for _, item := range testItems {
			jsonData, _ := json.Marshal(item)
			w := httptest.NewRecorder()
			httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("X-User-ID", "1")

			router.ServeHTTP(w, httpReq)
			assert.Equal(t, http.StatusCreated, w.Code)
		}
	})

	t.Run("Get all items", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(3), response["total"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 3)
	})

	t.Run("Filter by category", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?category=electronics", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	t.Run("Filter by price range", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?min_price=50&max_price=500", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["total"].(float64) >= 1)
	})

	t.Run("Search items", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?search=electronics", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.True(t, response["total"].(float64) >= 1)
	})

	t.Run("Filter by price type", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?price_type=bidding", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), response["total"])
	})

	t.Run("Pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?page=1&per_page=2", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(3), response["total"])
		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(2), response["per_page"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})
}

func TestIntegration_UserSpecificEndpoints(t *testing.T) {
	db, err := setupIntegrationTestDB()
	assert.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	// Create test items and transactions
	t.Run("Setup test data", func(t *testing.T) {
		// User 1 creates items
		items := []models.CreateStoreItemRequest{
			{
				Title:      "User 1 Item 1",
				PriceType:  "fixed",
				FixedPrice: 100.0,
			},
			{
				Title:      "User 1 Item 2",
				PriceType:  "fixed",
				FixedPrice: 200.0,
			},
		}

		for _, item := range items {
			jsonData, _ := json.Marshal(item)
			w := httptest.NewRecorder()
			httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("X-User-ID", "1")

			router.ServeHTTP(w, httpReq)
			assert.Equal(t, http.StatusCreated, w.Code)
		}

		// User 2 creates item for bidding
		bidItem := models.CreateStoreItemRequest{
			Title:       "Auction Item",
			PriceType:   "bidding",
			StartingBid: 50.0,
		}

		jsonData, _ := json.Marshal(bidItem)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Get user listings", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/listings", nil)
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 2)

		for _, item := range response {
			assert.Equal(t, uint(1), item.SellerID)
		}
	})

	t.Run("Place bid and get user bids", func(t *testing.T) {
		// First, place a bid
		bidReq := models.CreateBidRequest{
			Amount: 75.0,
		}

		jsonData, _ := json.Marshal(bidReq)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/3/bids", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusCreated, w.Code)

		// Now get user bids
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("GET", "/api/v1/user/bids", nil)
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.Bid
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, uint(1), response[0].BidderID)
		assert.Equal(t, 75.0, response[0].Amount)
	})

	t.Run("Purchase item and get user purchases", func(t *testing.T) {
		// Purchase an item
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/purchase", nil)
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)
		assert.Equal(t, http.StatusOK, w.Code)

		// Get user purchases
		w = httptest.NewRecorder()
		httpReq, _ = http.NewRequest("GET", "/api/v1/user/purchases", nil)
		httpReq.Header.Set("X-User-ID", "2")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, uint(2), *response[0].BuyerID)
		assert.Equal(t, "sold", response[0].Status)
	})
}
