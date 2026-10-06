package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"store-service/internal/models"
	"store-service/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockRequestService struct {
	mock.Mock
}

func (m *MockRequestService) CreateRequest(userID uint, req models.CreateItemRequestRequest) (*models.ItemRequest, error) {
	args := m.Called(userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ItemRequest), args.Error(1)
}

func (m *MockRequestService) GetRequest(id uint) (*models.ItemRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ItemRequest), args.Error(1)
}

func (m *MockRequestService) GetRequests(filter models.ItemRequestFilter) ([]models.ItemRequest, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]models.ItemRequest), args.Get(1).(int64), args.Error(2)
}

func (m *MockRequestService) GetRequestListings(id uint) ([]models.StoreItem, error) {
	args := m.Called(id)
	return args.Get(0).([]models.StoreItem), args.Error(1)
}

func (m *MockRequestService) GetUserRequests(userID uint) ([]models.ItemRequest, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.ItemRequest), args.Error(1)
}

func (m *MockRequestService) RepostRequest(id uint, userID uint) (*models.ItemRequest, error) {
	args := m.Called(id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ItemRequest), args.Error(1)
}

func (m *MockRequestService) DeleteRequest(id uint, userID uint) error {
	return m.Called(id, userID).Error(0)
}

func setupRequestRouter(service *MockRequestService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewRequestHandler(service)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", uint(1)); c.Next() })
	r.GET("/requests", h.GetRequests)
	r.GET("/requests/:id", h.GetRequest)
	r.GET("/requests/:id/listings", h.GetRequestListings)
	r.POST("/requests", h.CreateRequest)
	r.POST("/requests/:id/repost", h.RepostRequest)
	r.DELETE("/requests/:id", h.DeleteRequest)
	r.GET("/user/requests", h.GetUserRequests)
	return r
}

func call(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestRequestHandlers_Create(t *testing.T) {
	s := new(MockRequestService)
	r := setupRequestRouter(s)
	s.On("CreateRequest", uint(1), models.CreateItemRequestRequest{Title: "Printer wanted", Description: "Any"}).Return(&models.ItemRequest{ID: 3}, nil)
	s.On("CreateRequest", uint(1), models.CreateItemRequestRequest{Title: "Printer", Description: "call 0772123456"}).Return(nil, services.NewUserError("remove phone numbers"))

	assert.Equal(t, http.StatusCreated, call(r, "POST", "/requests", `{"title":"Printer wanted","description":"Any"}`).Code)
	bad := call(r, "POST", "/requests", `{"title":"Printer","description":"call 0772123456"}`)
	assert.Equal(t, http.StatusBadRequest, bad.Code)
	assert.Contains(t, bad.Body.String(), "remove phone numbers")
	assert.Equal(t, http.StatusBadRequest, call(r, "POST", "/requests", `{}`).Code)
}

func TestRequestHandlers_Reads(t *testing.T) {
	s := new(MockRequestService)
	r := setupRequestRouter(s)
	s.On("GetRequests", models.ItemRequestFilter{Search: "printer", SortBy: "deadline", SortOrder: "asc", ExcludeRequesterID: 1, Page: 2, PerPage: 10}).
		Return([]models.ItemRequest{{ID: 3}}, int64(11), nil)
	s.On("GetRequest", uint(3)).Return(&models.ItemRequest{ID: 3}, nil)
	s.On("GetRequest", uint(4)).Return(nil, gorm.ErrRecordNotFound)
	s.On("GetRequestListings", uint(3)).Return([]models.StoreItem{{ID: 9}}, nil)
	s.On("GetUserRequests", uint(1)).Return([]models.ItemRequest{{ID: 3}}, nil)

	list := call(r, "GET", "/requests?search=printer&sort_by=deadline&sort_order=asc&exclude_requester_id=1&page=2&per_page=10", "")
	assert.Equal(t, http.StatusOK, list.Code)
	assert.Contains(t, list.Body.String(), `"total":11`)
	assert.Equal(t, http.StatusOK, call(r, "GET", "/requests/3", "").Code)
	assert.Equal(t, http.StatusNotFound, call(r, "GET", "/requests/4", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "GET", "/requests/x", "").Code)
	assert.Equal(t, http.StatusOK, call(r, "GET", "/requests/3/listings", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "GET", "/requests/x/listings", "").Code)
	assert.Equal(t, http.StatusOK, call(r, "GET", "/user/requests", "").Code)
}

func TestRequestHandlers_ReadFailures(t *testing.T) {
	s := new(MockRequestService)
	r := setupRequestRouter(s)
	s.On("GetRequests", mock.Anything).Return([]models.ItemRequest{}, int64(0), services.NewUserError("db"))
	s.On("GetRequestListings", uint(3)).Return([]models.StoreItem{}, services.NewUserError("db"))
	s.On("GetUserRequests", uint(1)).Return([]models.ItemRequest{}, services.NewUserError("db"))

	assert.Equal(t, http.StatusInternalServerError, call(r, "GET", "/requests", "").Code)
	assert.Equal(t, http.StatusInternalServerError, call(r, "GET", "/requests/3/listings", "").Code)
	assert.Equal(t, http.StatusInternalServerError, call(r, "GET", "/user/requests", "").Code)
}

func TestRequestHandlers_RepostAndDelete(t *testing.T) {
	s := new(MockRequestService)
	r := setupRequestRouter(s)
	s.On("RepostRequest", uint(3), uint(1)).Return(&models.ItemRequest{ID: 3, Status: "active"}, nil)
	s.On("RepostRequest", uint(4), uint(1)).Return(nil, services.NewUserError("only an expired request can be reposted"))
	s.On("DeleteRequest", uint(3), uint(1)).Return(nil)
	s.On("DeleteRequest", uint(4), uint(1)).Return(gorm.ErrRecordNotFound)

	assert.Equal(t, http.StatusOK, call(r, "POST", "/requests/3/repost", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "POST", "/requests/4/repost", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "POST", "/requests/x/repost", "").Code)
	assert.Equal(t, http.StatusOK, call(r, "DELETE", "/requests/3", "").Code)
	assert.Equal(t, http.StatusNotFound, call(r, "DELETE", "/requests/4", "").Code)
	assert.Equal(t, http.StatusBadRequest, call(r, "DELETE", "/requests/x", "").Code)
}

func TestUnexpectedErrorsDontLeak(t *testing.T) {
	s := new(MockRequestService)
	r := setupRequestRouter(s)
	dbErr := errors.New(`pq: duplicate key value violates unique constraint "idx_item_requests_requester_id" (SQLSTATE 23505)`)
	s.On("CreateRequest", uint(1), mock.Anything).Return(nil, dbErr)
	s.On("RepostRequest", uint(3), uint(1)).Return(nil, dbErr)

	for _, w := range []*httptest.ResponseRecorder{
		call(r, "POST", "/requests", `{"title":"Printer","description":"Any"}`),
		call(r, "POST", "/requests/3/repost", ""),
	} {
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), "SQLSTATE")
		assert.NotContains(t, w.Body.String(), "idx_item_requests")
		assert.Contains(t, w.Body.String(), "something went wrong")
	}
}
