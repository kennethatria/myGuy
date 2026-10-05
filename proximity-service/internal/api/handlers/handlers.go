// Package handlers is the proximity service's internal HTTP API.
package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"proximity-service/internal/geo"
	"proximity-service/internal/services"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *services.ProximityService
}

func NewHandler(service *services.ProximityService) *Handler {
	return &Handler{service: service}
}

// Register mounts the internal routes behind guard, and /health outside it.
func (h *Handler) Register(r *gin.Engine, guard gin.HandlerFunc) {
	r.GET("/health", h.Health)
	internal := r.Group("/internal", guard)
	internal.PUT("/locations/:kind/:id", h.SaveLocation)
	internal.DELETE("/locations/:kind/:id", h.DeleteLocation)
	internal.POST("/distances/:kind", h.Distances)
}

// respondError maps a rule broken by the caller to 400 and anything else
// (Redis) to 503, so callers fall back instead of retrying a bad request.
func respondError(c *gin.Context, err error) {
	for _, bad := range []error{services.ErrUnknownKind, services.ErrInvalidID, services.ErrTooManyIDs, services.ErrNoOrigin, geo.ErrOutOfRange} {
		if errors.Is(err, bad) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "locations are unavailable"})
}

func parseID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, services.ErrInvalidID
	}
	return id, nil
}

type locationBody struct {
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

func (h *Handler) SaveLocation(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	var body locationBody
	if err := c.ShouldBindJSON(&body); err != nil || body.Lat == nil || body.Lng == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng are required"})
		return
	}
	if err := h.service.SaveLocation(c.Request.Context(), c.Param("kind"), id, *body.Lat, *body.Lng); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteLocation(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.service.DeleteLocation(c.Request.Context(), c.Param("kind"), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type distancesBody struct {
	Lat  *float64      `json:"lat"`
	Lng  *float64      `json:"lng"`
	From *services.Ref `json:"from"`
	IDs  []uint64      `json:"ids"`
}

func (h *Handler) Distances(c *gin.Context) {
	var body distancesBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	results, err := h.service.Distances(c.Request.Context(), services.DistancesQuery{
		Kind: c.Param("kind"), Lat: body.Lat, Lng: body.Lng, From: body.From, IDs: body.IDs,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *Handler) Health(c *gin.Context) {
	if err := h.service.Healthy(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
