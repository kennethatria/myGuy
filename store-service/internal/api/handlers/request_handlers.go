package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"store-service/internal/models"
	"store-service/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequestHandler serves "wanted" notes: items people ask sellers for.
type RequestHandler struct {
	service services.RequestServiceInterface
}

func NewRequestHandler(service services.RequestServiceInterface) *RequestHandler {
	return &RequestHandler{service: service}
}

// requestError answers with 404 for a missing request, else as
// respondError does (the service's reason for users, a generic 500 otherwise).
func requestError(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	respondError(c, http.StatusBadRequest, err)
}

func (h *RequestHandler) CreateRequest(c *gin.Context) {
	var req models.CreateItemRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "headline is required"})
		return
	}
	request, err := h.service.CreateRequest(c.GetUint("userID"), req)
	if err != nil {
		requestError(c, err)
		return
	}
	c.JSON(http.StatusCreated, request)
}

func (h *RequestHandler) GetRequests(c *gin.Context) {
	filter := models.ItemRequestFilter{
		Search:    c.Query("search"),
		// The board: a request past its 24 hours leaves it even while a
		// listing made for it waits
		LiveAt:    time.Now().UTC(),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Page:      1,
		PerPage:   20,
	}
	if id, err := parseID(c.Query("exclude_requester_id")); err == nil {
		filter.ExcludeRequesterID = id
	}
	if p, err := strconv.Atoi(c.Query("page")); err == nil && p > 0 {
		filter.Page = p
	}
	if pp, err := strconv.Atoi(c.Query("per_page")); err == nil && pp > 0 && pp <= 100 {
		filter.PerPage = pp
	}

	near, err := parseNear(c.Query("near"))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	var requests []models.ItemRequest
	var total int64
	if near != nil && filter.SortBy == "distance" {
		requests, total, err = h.service.GetRequestsNear(filter, *near)
	} else {
		requests, total, err = h.service.GetRequests(filter)
		if err == nil && near != nil {
			h.service.TagRequestDistances(requests, *near)
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"requests": requests, "total": total, "page": filter.Page, "per_page": filter.PerPage})
}

func (h *RequestHandler) GetRequest(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	request, err := h.service.GetRequest(id)
	if err != nil {
		requestError(c, err)
		return
	}
	c.JSON(http.StatusOK, request)
}

// GetRequestListings lists the live listings made for a request
func (h *RequestHandler) GetRequestListings(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	items, err := h.service.GetRequestListings(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve listings"})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *RequestHandler) GetUserRequests(c *gin.Context) {
	requests, err := h.service.GetUserRequests(c.GetUint("userID"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve requests"})
		return
	}
	c.JSON(http.StatusOK, requests)
}

func (h *RequestHandler) RepostRequest(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	request, err := h.service.RepostRequest(id, c.GetUint("userID"))
	if err != nil {
		requestError(c, err)
		return
	}
	c.JSON(http.StatusOK, request)
}

func (h *RequestHandler) UpdateRequest(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	var req models.UpdateItemRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "headline is required"})
		return
	}
	request, err := h.service.UpdateRequest(id, c.GetUint("userID"), req)
	if err != nil {
		requestError(c, err)
		return
	}
	c.JSON(http.StatusOK, request)
}

func (h *RequestHandler) DeleteRequest(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request id"})
		return
	}
	if err := h.service.DeleteRequest(id, c.GetUint("userID")); err != nil {
		requestError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "request removed"})
}
