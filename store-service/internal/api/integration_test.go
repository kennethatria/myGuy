package api

import (
	"bytes"
	"database/sql"
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
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// keepAliveDB is a global handle to keep the shared in-memory database alive
var keepAliveDB *sql.DB

func setupIntegrationTestDB(t *testing.T) (*gorm.DB, error) {
	// Use a unique name for each test function to avoid interference
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Get underlying sql.DB to keep it alive
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	// Limit to a single connection to prevent SQLite "table is locked" errors
	// when test functions run concurrently within the same package.
	sqlDB.SetMaxOpenConns(1)

	// We don't close it until the end of the test function
	t.Cleanup(func() {
		sqlDB.Close()
	})

	// Auto migrate the schema
	err = db.AutoMigrate(&models.StoreItem{}, &models.ItemImage{}, &models.BookingRequest{}, &models.User{}, &models.ItemRequest{})
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
	bookingRepo := repositories.NewBookingRequestRepository(db)
	userRepo := repositories.NewUserRepository(db)

	// Initialize service
	storeService := services.NewStoreService(db, itemRepo, bookingRepo, userRepo)

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
		api.GET("/user/listings", storeHandler.GetUserListings)
		api.POST("/items/:id/booking-request", storeHandler.CreateBookingRequest)
		api.GET("/items/:id/booking-request", storeHandler.GetBookingRequest)
		api.POST("/booking-requests/:requestId/approve", storeHandler.ApproveBookingRequest)
		api.POST("/booking-requests/:requestId/reject", storeHandler.RejectBookingRequest)
		api.GET("/user/booking-requests", storeHandler.GetUserBookingRequests)
	}

	return router
}

// seedListing stores a live listing straight in the database
func seedListing(t *testing.T, db *gorm.DB, item models.StoreItem) uint {
	t.Helper()
	deadline := time.Now().UTC().Add(24 * time.Hour)
	item.Status = "active"
	item.Deadline = &deadline
	require.NoError(t, db.Create(&item).Error)
	return item.ID
}

func TestIntegration_ItemLifecycle(t *testing.T) {
	db, err := setupIntegrationTestDB(t)
	require.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	t.Run("Create fixed price item", func(t *testing.T) {
		req := models.CreateStoreItemRequest{
			Title:       "iPhone 15 Pro",
			Description: "Brand new iPhone 15 Pro in pristine condition",
			Category:    "electronics",
			Condition:   "new",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, req.Title, response.Title)
		// Just a note: the price goes in it or is agreed in chat
		assert.Equal(t, "fixed", response.PriceType)
		assert.Zero(t, response.FixedPrice)
		assert.Equal(t, uint(1), response.SellerID)

		itemID = response.ID
	})

	t.Run("Get created item", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d", itemID), nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
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

		require.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, updateReq.Title, response.Title)
		assert.Equal(t, updateReq.Description, response.Description)
	})

	t.Run("Verify item is sold", func(t *testing.T) {
		// A handover in chat marks it sold (there is no direct purchase)
		buyer := uint(2)
		require.NoError(t, db.Model(&models.StoreItem{}).Where("id = ?", itemID).Updates(map[string]interface{}{"status": "sold", "buyer_id": buyer}).Error)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d", itemID), nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "sold", response.Status)
		assert.Equal(t, uint(2), *response.BuyerID)
	})

	t.Run("Cannot update sold item", func(t *testing.T) {
		updateReq := models.UpdateStoreItemRequest{
			Title: "Should not work",
			Description: "A short note",
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

func TestIntegration_BookingLifecycle(t *testing.T) {
	db, err := setupIntegrationTestDB(t)
	require.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	t.Run("Create bookable item", func(t *testing.T) {
		req := models.CreateStoreItemRequest{
			Title:       "Camera Equipment",
			Description: "Professional DSLR camera with lenses",
			Category:    "electronics",
			Condition:   "good",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
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

		require.Equal(t, http.StatusCreated, w.Code)

		var response models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
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

		require.Equal(t, http.StatusOK, w.Code)

		var wrapper struct {
			BookingRequest models.BookingRequest `json:"booking_request"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &wrapper)
		require.NoError(t, err)
		assert.Equal(t, requestID, wrapper.BookingRequest.ID)
	})

	t.Run("Get booking request as requester", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var wrapper struct {
			BookingRequest models.BookingRequest `json:"booking_request"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &wrapper)
		require.NoError(t, err)
		assert.Equal(t, requestID, wrapper.BookingRequest.ID)
	})

	t.Run("Approve booking request", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/booking-requests/%d/approve", requestID), nil)
		httpReq.Header.Set("X-User-ID", "1") // Item owner

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Verify booking request is approved", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.NotNil(t, response["booking_request"])
		
		bookingRequest := response["booking_request"].(map[string]interface{})
		assert.Equal(t, "approved", bookingRequest["status"])
	})

	t.Run("Get user booking requests", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/booking-requests", nil)
		httpReq.Header.Set("X-User-ID", "2") // Requester

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response []models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, requestID, response[0].ID)
	})
}

func TestIntegration_BookingRequestEdgeCases(t *testing.T) {
	db, err := setupIntegrationTestDB(t)
	require.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	var itemID uint

	// Create a test item
	t.Run("Create item for edge case testing", func(t *testing.T) {
		item := models.CreateStoreItemRequest{
			Title:       "Edge Case Test Item",
			Description: "Testing edge cases",
			Category:    "electronics",
			Condition:   "new",
		}

		jsonData, _ := json.Marshal(item)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-User-ID", "1") // Seller

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusCreated, w.Code)

		var response models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		itemID = response.ID
	})

	t.Run("Get booking request when none exists", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/items/%d/booking-request", itemID), nil)
		httpReq.Header.Set("X-User-ID", "2") // Different user

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
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
		require.Equal(t, http.StatusCreated, w.Code)

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

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		require.NotNil(t, response["booking_request"])

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
	db, err := setupIntegrationTestDB(t)
	require.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	t.Run("Create test items", func(t *testing.T) {
		for _, item := range []models.StoreItem{
			{Title: "Expensive Electronics", Description: "High-end gadget", SellerID: 1, PriceType: "fixed", Category: "electronics", Condition: "new"},
			{Title: "Cheap Book", Description: "Interesting novel", SellerID: 1, PriceType: "fixed", Category: "books", Condition: "good"},
			{Title: "Electronics Lamp", Description: "Desk lamp", SellerID: 1, PriceType: "fixed", Category: "electronics", Condition: "fair"},
		} {
			seedListing(t, db, item)
		}
	})

	t.Run("Get all items", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items", nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(3), response["total"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 3)
	})

	t.Run("Filter by category", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?category=electronics", nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(2), response["total"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	t.Run("Search items", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?search=electronics", nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["total"].(float64) >= 1)
	})

	t.Run("Pagination", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?page=1&per_page=2", nil)

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, float64(3), response["total"])
		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(2), response["per_page"])

		items := response["items"].([]interface{})
		assert.Len(t, items, 2)
	})
}

func TestIntegration_UserSpecificEndpoints(t *testing.T) {
	db, err := setupIntegrationTestDB(t)
	require.NoError(t, err)

	router := setupIntegrationTestRouter(db)

	// Create test items and transactions
	t.Run("Setup test data", func(t *testing.T) {
		// User 1 creates items
		items := []models.CreateStoreItemRequest{
			{
				Title:      "User 1 Item 1",
				Description: "A short note",
				Condition:  "new",
			},
			{
				Title:      "User 1 Item 2",
				Description: "A short note",
				Condition:  "new",
			},
		}

		for _, item := range items {
			jsonData, _ := json.Marshal(item)
			w := httptest.NewRecorder()
			httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
			httpReq.Header.Set("Content-Type", "application/json")
			httpReq.Header.Set("X-User-ID", "1")

			router.ServeHTTP(w, httpReq)
			require.Equal(t, http.StatusCreated, w.Code)
		}
	})

	t.Run("Get user listings", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/listings", nil)
		httpReq.Header.Set("X-User-ID", "1")

		router.ServeHTTP(w, httpReq)

		require.Equal(t, http.StatusOK, w.Code)

		var response []models.StoreItem
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)

		for _, item := range response {
			assert.Equal(t, uint(1), item.SellerID)
		}
	})

}
