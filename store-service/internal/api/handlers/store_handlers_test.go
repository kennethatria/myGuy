package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"store-service/internal/models"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock StoreService
type MockStoreService struct {
	mock.Mock
}

func (m *MockStoreService) CreateItem(userID uint, req models.CreateStoreItemRequest) (*models.StoreItem, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoreItem), args.Error(1)
}

func (m *MockStoreService) GetItem(id uint) (*models.StoreItem, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoreItem), args.Error(1)
}

func (m *MockStoreService) GetItems(filter models.StoreItemFilter) ([]models.StoreItem, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.StoreItem), args.Get(1).(int64), args.Error(2)
}

func (m *MockStoreService) UpdateItem(id uint, userID uint, req models.UpdateStoreItemRequest) (*models.StoreItem, error) {
	args := m.Called(id, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.StoreItem), args.Error(1)
}

func (m *MockStoreService) DeleteItem(id uint, userID uint) error {
	args := m.Called(id, userID)
	return args.Error(0)
}

func (m *MockStoreService) PlaceBid(itemID uint, userID uint, req models.CreateBidRequest) (*models.Bid, error) {
	args := m.Called(itemID, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Bid), args.Error(1)
}

func (m *MockStoreService) GetItemBids(itemID uint) ([]models.Bid, error) {
	args := m.Called(itemID)
	return args.Get(0).([]models.Bid), args.Error(1)
}

func (m *MockStoreService) AcceptBid(itemID uint, bidID uint, sellerID uint) error {
	args := m.Called(itemID, bidID, sellerID)
	return args.Error(0)
}

func (m *MockStoreService) PurchaseItem(itemID uint, buyerID uint) error {
	args := m.Called(itemID, buyerID)
	return args.Error(0)
}

func (m *MockStoreService) GetUserListings(userID uint) ([]models.StoreItem, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.StoreItem), args.Error(1)
}

func (m *MockStoreService) GetUserPurchases(userID uint) ([]models.StoreItem, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.StoreItem), args.Error(1)
}

func (m *MockStoreService) GetUserBids(userID uint) ([]models.Bid, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Bid), args.Error(1)
}

func (m *MockStoreService) CreateBookingRequest(itemID uint, requesterID uint, message string) (*models.BookingRequest, error) {
	args := m.Called(itemID, requesterID, message)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) GetBookingRequestByItem(itemID uint, userID uint) (*models.BookingRequest, error) {
	args := m.Called(itemID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) GetAllBookingRequestsByItem(itemID uint, userID uint) ([]models.BookingRequest, error) {
	args := m.Called(itemID, userID)
	return args.Get(0).([]models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) ApproveBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error) {
	args := m.Called(requestID, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) RejectBookingRequest(requestID uint, ownerID uint) (*models.BookingRequest, error) {
	args := m.Called(requestID, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) GetUserBookingRequests(userID uint) ([]models.BookingRequest, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) ConfirmItemReceived(requestID uint, buyerID uint) (*models.BookingRequest, error) {
	args := m.Called(requestID, buyerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) ConfirmDelivery(requestID uint, sellerID uint) (*models.BookingRequest, error) {
	args := m.Called(requestID, sellerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) SubmitBuyerRating(requestID uint, buyerID uint, rating int, review string) (*models.BookingRequest, error) {
	args := m.Called(requestID, buyerID, rating, review)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func (m *MockStoreService) SubmitSellerRating(requestID uint, sellerID uint, rating int, review string) (*models.BookingRequest, error) {
	args := m.Called(requestID, sellerID, rating, review)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.BookingRequest), args.Error(1)
}

func setupTestRouter(handler *StoreHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Middleware to set user context (simulating JWT middleware)
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Set("username", "testuser")
		c.Set("userEmail", "test@example.com")
		c.Set("userName", "Test User")
		c.Next()
	})

	return setupTestRouterRoutes(router, handler)
}

// setupTestRouterWithUserID creates a router that reads userID from X-User-ID header
func setupTestRouterWithUserID(handler *StoreHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")
		userID := uint(1)
		if userIDStr != "" {
			if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
				userID = uint(id)
			}
		}
		c.Set("userID", userID)
		c.Set("username", "testuser")
		c.Set("userEmail", "test@example.com")
		c.Set("userName", "Test User")
		c.Next()
	})

	return setupTestRouterRoutes(router, handler)
}

func setupTestRouterRoutes(router *gin.Engine, handler *StoreHandler) *gin.Engine {

	api := router.Group("/api/v1")
	{
		api.POST("/items", handler.CreateItem)
		api.GET("/items/:id", handler.GetItem)
		api.GET("/items", handler.GetItems)
		api.PUT("/items/:id", handler.UpdateItem)
		api.DELETE("/items/:id", handler.DeleteItem)
		api.POST("/items/:id/bids", handler.PlaceBid)
		api.GET("/items/:id/bids", handler.GetItemBids)
		api.POST("/items/:id/bids/:bidId/accept", handler.AcceptBid)
		api.POST("/items/:id/purchase", handler.PurchaseItem)
		api.GET("/user/listings", handler.GetUserListings)
		api.GET("/user/purchases", handler.GetUserPurchases)
		api.GET("/user/bids", handler.GetUserBids)
		api.POST("/items/:id/booking-request", handler.CreateBookingRequest)
		api.GET("/items/:id/booking-request", handler.GetBookingRequest)
		api.GET("/items/:id/booking-requests", handler.GetAllBookingRequests)
		api.POST("/booking-requests/:requestId/approve", handler.ApproveBookingRequest)
		api.POST("/booking-requests/:requestId/reject", handler.RejectBookingRequest)
		api.GET("/user/booking-requests", handler.GetUserBookingRequests)
		api.POST("/booking-requests/:requestId/confirm-received", handler.ConfirmItemReceived)
		api.POST("/booking-requests/:requestId/confirm-delivery", handler.ConfirmDelivery)
		api.POST("/booking-requests/:requestId/buyer-rating", handler.SubmitBuyerRating)
		api.POST("/booking-requests/:requestId/seller-rating", handler.SubmitSellerRating)
	}

	return router
}

func TestCreateItem(t *testing.T) {
	t.Run("successful JSON creation", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateStoreItemRequest{
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "fixed",
			FixedPrice:  100.0,
			Category:    "electronics",
			Condition:   "new",
		}

		expectedItem := &models.StoreItem{
			ID:          1,
			Title:       req.Title,
			Description: req.Description,
			PriceType:   req.PriceType,
			FixedPrice:  req.FixedPrice,
			Category:    req.Category,
			Condition:   req.Condition,
			SellerID:    1,
			Status:      "active",
		}

		mockService.On("CreateItem", uint(1), mock.AnythingOfType("models.CreateStoreItemRequest")).Return(expectedItem, nil)

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer([]byte("invalid json")))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateStoreItemRequest{
			Title:      "Test Item",
			PriceType:  "fixed",
			FixedPrice: 100.0,
			Condition:  "new",
		}

		mockService.On("CreateItem", uint(1), mock.AnythingOfType("models.CreateStoreItemRequest")).Return(nil, errors.New("service error"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("form data creation", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		expectedItem := &models.StoreItem{
			ID:          1,
			Title:       "Test Item",
			Description: "Test Description",
			PriceType:   "fixed",
			FixedPrice:  100.0,
			SellerID:    1,
			Status:      "active",
		}

		mockService.On("CreateItem", uint(1), mock.AnythingOfType("models.CreateStoreItemRequest")).Return(expectedItem, nil)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("title", "Test Item")
		writer.WriteField("description", "Test Description")
		writer.WriteField("price", "100.0")
		writer.WriteField("category", "electronics")
		writer.WriteField("condition", "new")
		writer.Close()

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items", body)
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetItem(t *testing.T) {
	mockService := new(MockStoreService)
	handler := NewStoreHandler(mockService)
	router := setupTestRouter(handler)

	t.Run("successful get", func(t *testing.T) {
		item := &models.StoreItem{
			ID:       1,
			Title:    "Test Item",
			SellerID: 1,
			Status:   "active",
		}

		mockService.On("GetItem", uint(1)).Return(item, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/invalid", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("item not found", func(t *testing.T) {
		mockService.On("GetItem", uint(999)).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/999", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetItems(t *testing.T) {
	t.Run("successful get items", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		items := []models.StoreItem{
			{ID: 1, Title: "Item 1", SellerID: 1, Status: "active"},
			{ID: 2, Title: "Item 2", SellerID: 2, Status: "active"},
		}

		mockService.On("GetItems", mock.AnythingOfType("models.StoreItemFilter")).Return(items, int64(2), nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("with filters", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		items := []models.StoreItem{
			{ID: 1, Title: "Electronics Item", SellerID: 1, Status: "active"},
		}

		mockService.On("GetItems", mock.AnythingOfType("models.StoreItemFilter")).Return(items, int64(1), nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items?category=electronics&min_price=50&max_price=500", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetItems", mock.AnythingOfType("models.StoreItemFilter")).Return([]models.StoreItem{}, int64(0), errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestUpdateItem(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.UpdateStoreItemRequest{
			Title:       "Updated Item",
			Description: "Updated Description",
		}

		updatedItem := &models.StoreItem{
			ID:          1,
			Title:       req.Title,
			Description: req.Description,
			SellerID:    1,
			Status:      "active",
		}

		mockService.On("UpdateItem", uint(1), uint(1), req).Return(updatedItem, nil)

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", "/api/v1/items/1", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", "/api/v1/items/invalid", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.UpdateStoreItemRequest{
			Title: "Updated Item",
		}

		mockService.On("UpdateItem", uint(1), uint(1), req).Return(nil, errors.New("unauthorized"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("PUT", "/api/v1/items/1", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestDeleteItem(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("DeleteItem", uint(1), uint(1)).Return(nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("DELETE", "/api/v1/items/1", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("DELETE", "/api/v1/items/invalid", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("DeleteItem", uint(1), uint(1)).Return(errors.New("unauthorized"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("DELETE", "/api/v1/items/1", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestPlaceBid(t *testing.T) {
	t.Run("successful bid", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBidRequest{
			Amount:  150.0,
			Message: "My bid",
		}

		bid := &models.Bid{
			ID:       1,
			ItemID:   1,
			BidderID: 1,
			Amount:   req.Amount,
			Message:  req.Message,
			Status:   "active",
		}

		mockService.On("PlaceBid", uint(1), uint(1), req).Return(bid, nil)

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/bids", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid bid amount", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBidRequest{
			Amount: 50.0,
		}

		mockService.On("PlaceBid", uint(1), uint(1), req).Return(nil, errors.New("bid amount too low"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/bids", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetItemBids(t *testing.T) {
	t.Run("successful get bids", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		bids := []models.Bid{
			{ID: 1, ItemID: 1, BidderID: 1, Amount: 150.0, Status: "active"},
			{ID: 2, ItemID: 1, BidderID: 2, Amount: 120.0, Status: "outbid"},
		}

		mockService.On("GetItemBids", uint(1)).Return(bids, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1/bids", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetItemBids", uint(1)).Return([]models.Bid{}, errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1/bids", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestPurchaseItem(t *testing.T) {
	t.Run("successful purchase", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("PurchaseItem", uint(1), uint(1)).Return(nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/purchase", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("cannot purchase own item", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("PurchaseItem", uint(1), uint(1)).Return(errors.New("cannot purchase own item"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/purchase", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetUserListings(t *testing.T) {
	t.Run("successful get user listings", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		items := []models.StoreItem{
			{ID: 1, Title: "My Item 1", SellerID: 1, Status: "active"},
			{ID: 2, Title: "My Item 2", SellerID: 1, Status: "sold"},
		}

		mockService.On("GetUserListings", uint(1)).Return(items, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/listings", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetUserListings", uint(1)).Return([]models.StoreItem{}, errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/listings", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestCreateBookingRequest(t *testing.T) {
	t.Run("successful booking request", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "I'd like to book this item",
		}

		bookingRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
			Message:     req.Message,
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(bookingRequest, nil)

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("cannot book own item", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "I'd like to book this item",
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(nil, errors.New("cannot book your own item"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("duplicate booking request", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "I'd like to book this item",
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(nil, errors.New("you already have a booking request for this item"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusConflict, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("empty message should work", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "",
		}

		bookingRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
			Message:     "",
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(bookingRequest, nil)

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusCreated, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer([]byte("{")))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid item ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "Test message",
		}

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/invalid/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("item not available for booking", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "Test message",
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(nil, errors.New("item is not available for booking"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req := models.CreateBookingRequestRequest{
			Message: "Test message",
		}

		mockService.On("CreateBookingRequest", uint(1), uint(1), req.Message).Return(nil, errors.New("database error"))

		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/booking-request", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestApproveBookingRequest(t *testing.T) {
	t.Run("successful approval", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{Status: "approved"}
		mockService.On("ApproveBookingRequest", uint(1), uint(1)).Return(booking, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/approve", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ApproveBookingRequest", uint(1), uint(1)).Return(nil, errors.New("unauthorized: you are not the owner of this item"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/approve", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/approve", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("booking request not pending", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ApproveBookingRequest", uint(1), uint(1)).Return(nil, errors.New("booking request is not pending"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/approve", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ApproveBookingRequest", uint(1), uint(1)).Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/approve", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestRejectBookingRequest(t *testing.T) {
	t.Run("successful rejection", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{Status: "rejected"}
		mockService.On("RejectBookingRequest", uint(1), uint(1)).Return(booking, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/reject", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("RejectBookingRequest", uint(1), uint(1)).Return(nil, errors.New("unauthorized: you are not the owner of this item"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/reject", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/reject", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("booking request not pending", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("RejectBookingRequest", uint(1), uint(1)).Return(nil, errors.New("booking request is not pending"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/reject", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("RejectBookingRequest", uint(1), uint(1)).Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/reject", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetUserBookingRequests(t *testing.T) {
	t.Run("successful get user booking requests", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		requests := []models.BookingRequest{
			{ID: 1, ItemID: 1, RequesterID: 1, Status: "pending", Message: "Booking request 1"},
			{ID: 2, ItemID: 2, RequesterID: 1, Status: "approved", Message: "Booking request 2"},
		}

		mockService.On("GetUserBookingRequests", uint(1)).Return(requests, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/booking-requests", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetUserBookingRequests", uint(1)).Return([]models.BookingRequest{}, errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/booking-requests", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("empty booking requests", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetUserBookingRequests", uint(1)).Return([]models.BookingRequest{}, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/booking-requests", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []models.BookingRequest
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Empty(t, response)

		mockService.AssertExpectations(t)
	})
}

func TestAcceptBid(t *testing.T) {
	t.Run("successful accept bid", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("AcceptBid", uint(1), uint(1), uint(1)).Return(nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/bids/1/accept", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid item ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/invalid/bids/1/accept", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid bid ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/bids/invalid/accept", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("unauthorized", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("AcceptBid", uint(1), uint(1), uint(1)).Return(errors.New("unauthorized"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/items/1/bids/1/accept", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetUserPurchases(t *testing.T) {
	t.Run("successful get user purchases", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		items := []models.StoreItem{
			{ID: 1, Title: "Purchased Item 1", SellerID: 2, Status: "sold"},
			{ID: 2, Title: "Purchased Item 2", SellerID: 3, Status: "sold"},
		}

		mockService.On("GetUserPurchases", uint(1)).Return(items, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/purchases", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetUserPurchases", uint(1)).Return([]models.StoreItem{}, errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/purchases", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetUserBids(t *testing.T) {
	t.Run("successful get user bids", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		bids := []models.Bid{
			{ID: 1, ItemID: 1, BidderID: 1, Amount: 150.0, Status: "active"},
			{ID: 2, ItemID: 2, BidderID: 1, Amount: 120.0, Status: "outbid"},
		}

		mockService.On("GetUserBids", uint(1)).Return(bids, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/bids", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetUserBids", uint(1)).Return([]models.Bid{}, errors.New("service error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/user/bids", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetBookingRequest(t *testing.T) {
	t.Run("successful get booking request", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		bookingRequest := &models.BookingRequest{
			ID:          1,
			ItemID:      1,
			RequesterID: 1,
			Status:      "pending",
			Message:     "Booking request message",
		}

		mockService.On("GetBookingRequestByItem", uint(1), uint(1)).Return(bookingRequest, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1/booking-request", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["booking_request"])

		mockService.AssertExpectations(t)
	})

	t.Run("booking request not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetBookingRequestByItem", uint(1), uint(1)).Return(nil, gorm.ErrRecordNotFound)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1/booking-request", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Nil(t, response["booking_request"])

		mockService.AssertExpectations(t)
	})

	t.Run("invalid item ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/invalid/booking-request", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("item not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetBookingRequestByItem", uint(999), uint(1)).Return(nil, errors.New("item not found"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/999/booking-request", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetBookingRequestByItem", uint(1), uint(1)).Return(nil, errors.New("database connection error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("GET", "/api/v1/items/1/booking-request", nil)

		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestGetAllBookingRequests(t *testing.T) {
	t.Run("successful retrieval of multiple booking requests", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		requests := []models.BookingRequest{
			{
				ID:          1,
				ItemID:      1,
				RequesterID: 2,
				Status:      "pending",
				Message:     "First request",
				CreatedAt:   time.Now(),
			},
			{
				ID:          2,
				ItemID:      1,
				RequesterID: 3,
				Status:      "approved",
				Message:     "Second request",
				CreatedAt:   time.Now(),
			},
		}

		mockService.On("GetAllBookingRequestsByItem", uint(1), uint(1)).Return(requests, nil)

		req, _ := http.NewRequest("GET", "/api/v1/items/1/booking-requests", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			BookingRequests []models.BookingRequest `json:"booking_requests"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.BookingRequests, 2)
		assert.Equal(t, "pending", response.BookingRequests[0].Status)
		assert.Equal(t, "approved", response.BookingRequests[1].Status)
		mockService.AssertExpectations(t)
	})

	t.Run("unauthorized access - not item owner", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		// Use router that reads userID from X-User-ID header
		router := setupTestRouterWithUserID(handler)

		mockService.On("GetAllBookingRequestsByItem", uint(1), uint(2)).Return([]models.BookingRequest{}, errors.New("unauthorized: you are not the owner of this item"))

		req, _ := http.NewRequest("GET", "/api/v1/items/1/booking-requests", nil)
		req.Header.Set("X-User-ID", "2")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid item ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		req, _ := http.NewRequest("GET", "/api/v1/items/invalid/booking-requests", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("item not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetAllBookingRequestsByItem", uint(999), uint(1)).Return([]models.BookingRequest{}, errors.New("item not found"))

		req, _ := http.NewRequest("GET", "/api/v1/items/999/booking-requests", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("empty booking requests list", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("GetAllBookingRequestsByItem", uint(1), uint(1)).Return([]models.BookingRequest{}, nil)

		req, _ := http.NewRequest("GET", "/api/v1/items/1/booking-requests", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response struct {
			BookingRequests []models.BookingRequest `json:"booking_requests"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Len(t, response.BookingRequests, 0)
		mockService.AssertExpectations(t)
	})
}

func TestConfirmItemReceived(t *testing.T) {
	t.Run("successful confirm receipt", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{ID: 1, Status: "item_received"}
		mockService.On("ConfirmItemReceived", uint(1), uint(1)).Return(booking, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("booking not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmItemReceived", uint(1), uint(1)).Return(nil, errors.New("booking request not found"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("not the buyer", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmItemReceived", uint(1), uint(1)).Return(nil, errors.New("only the buyer can confirm receipt"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("booking not approved", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmItemReceived", uint(1), uint(1)).Return(nil, errors.New("booking must be approved before confirming receipt"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmItemReceived", uint(1), uint(1)).Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-received", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestConfirmDelivery(t *testing.T) {
	t.Run("successful confirm delivery", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{ID: 1, Status: "completed"}
		mockService.On("ConfirmDelivery", uint(1), uint(1)).Return(booking, nil)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("booking not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmDelivery", uint(1), uint(1)).Return(nil, errors.New("booking request not found"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("not the seller", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmDelivery", uint(1), uint(1)).Return(nil, errors.New("only the seller can confirm delivery"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("buyer must confirm first", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmDelivery", uint(1), uint(1)).Return(nil, errors.New("buyer must confirm receipt before seller can confirm delivery"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("ConfirmDelivery", uint(1), uint(1)).Return(nil, errors.New("database error"))

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/confirm-delivery", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestSubmitBuyerRating(t *testing.T) {
	t.Run("successful buyer rating", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{ID: 1, Status: "completed"}
		mockService.On("SubmitBuyerRating", uint(1), uint(1), 5, "Great seller!").Return(booking, nil)

		req := models.SubmitRatingRequest{Rating: 5, Review: "Great seller!"}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/buyer-rating", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid request body", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer([]byte("invalid")))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("booking not found", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitBuyerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("booking request not found"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusNotFound, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("not the buyer", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitBuyerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("only the buyer can rate the seller"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("booking not completed", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitBuyerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("booking must be completed before rating"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("already rated", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitBuyerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("buyer has already rated this transaction"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitBuyerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("database error"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/buyer-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

func TestSubmitSellerRating(t *testing.T) {
	t.Run("successful seller rating", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		booking := &models.BookingRequest{ID: 1, Status: "completed"}
		mockService.On("SubmitSellerRating", uint(1), uint(1), 5, "Great buyer!").Return(booking, nil)

		req := models.SubmitRatingRequest{Rating: 5, Review: "Great buyer!"}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/seller-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusOK, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("invalid request ID", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/invalid/seller-rating", nil)
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not the seller", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitSellerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("only the seller can rate the buyer"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/seller-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusForbidden, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("already rated", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitSellerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("seller has already rated this transaction"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/seller-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		mockService.AssertExpectations(t)
	})

	t.Run("service error", func(t *testing.T) {
		mockService := new(MockStoreService)
		handler := NewStoreHandler(mockService)
		router := setupTestRouter(handler)

		mockService.On("SubmitSellerRating", uint(1), uint(1), 4, "").Return(nil, errors.New("database error"))

		req := models.SubmitRatingRequest{Rating: 4}
		jsonData, _ := json.Marshal(req)
		w := httptest.NewRecorder()
		httpReq, _ := http.NewRequest("POST", "/api/v1/booking-requests/1/seller-rating", bytes.NewBuffer(jsonData))
		httpReq.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, httpReq)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		mockService.AssertExpectations(t)
	})
}

// Clean up test uploads directory
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Clean up
	os.RemoveAll("./uploads")

	os.Exit(code)
}
