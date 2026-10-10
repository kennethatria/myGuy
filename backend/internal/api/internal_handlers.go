package api

import (
	"crypto/subtle"

	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"myguy/internal/models"
	"myguy/internal/services"
)

// InternalHandler serves /internal/v1: calls from the other services and
// from myguy-admin on the server, never from browsers (nginx routes only
// /api/v1 here, and every call needs INTERNAL_API_KEY).
type InternalHandler struct {
	blocks *services.BlockService
}

func NewInternalHandler(blocks *services.BlockService) *InternalHandler {
	return &InternalHandler{blocks: blocks}
}

// InternalKeyRequired admits requests carrying key in X-Internal-API-Key.
// With no key configured, nothing is admitted.
func InternalKeyRequired(key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		got := c.GetHeader("X-Internal-API-Key")
		if key == "" || subtle.ConstantTimeCompare([]byte(got), []byte(key)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}

type blockRequest struct {
	Email  string `json:"email" binding:"required"`
	Reason string `json:"reason"`
	Days   int    `json:"days"`
	By     string `json:"by"`
}

type blockResponse struct {
	Email     string     `json:"email"`
	Blocked   bool       `json:"blocked"`
	Reason    string     `json:"reason,omitempty"`
	Until     *time.Time `json:"until,omitempty"`
	CreatedBy string     `json:"created_by,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

func toBlockResponse(email string, block *models.BlockedEmail) blockResponse {
	if block == nil {
		return blockResponse{Email: services.NormalizeEmail(email)}
	}
	created := block.CreatedAt
	return blockResponse{Email: block.Email, Blocked: true, Reason: block.Reason, Until: block.Until,
		CreatedBy: block.CreatedBy, CreatedAt: &created}
}

// BlockEmail blocks an address: POST /internal/v1/email-blocks
// {email, reason, days (0: for good), by}.
func (h *InternalHandler) BlockEmail(c *gin.Context) {
	var req blockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	block, err := h.blocks.Block(c.Request.Context(), req.Email, req.Reason, req.Days, req.By)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	log.Printf("block event=blocked email=%q days=%d by=%q", block.Email, req.Days, req.By)
	c.JSON(http.StatusOK, toBlockResponse(block.Email, block))
}

// UnblockEmail lifts a block: DELETE /internal/v1/email-blocks/:email.
func (h *InternalHandler) UnblockEmail(c *gin.Context) {
	email := c.Param("email")
	removed, err := h.blocks.Unblock(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if !removed {
		c.JSON(http.StatusNotFound, gin.H{"error": "that address isn't blocked"})
		return
	}
	log.Printf("block event=unblocked email=%q", services.NormalizeEmail(email))
	c.JSON(http.StatusOK, toBlockResponse(email, nil))
}

// EmailBlockStatus: GET /internal/v1/email-blocks/:email.
func (h *InternalHandler) EmailBlockStatus(c *gin.Context) {
	email := c.Param("email")
	block, err := h.blocks.Status(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusOK, toBlockResponse(email, block))
}

// ListEmailBlocks: GET /internal/v1/email-blocks, the blocks in force.
func (h *InternalHandler) ListEmailBlocks(c *gin.Context) {
	blocks, err := h.blocks.List(c.Request.Context())
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	out := make([]blockResponse, 0, len(blocks))
	for i := range blocks {
		out = append(out, toBlockResponse(blocks[i].Email, &blocks[i]))
	}
	c.JSON(http.StatusOK, gin.H{"blocks": out})
}

// BlockedUsers: GET /internal/v1/blocked-users, the accounts store-service
// and chat refuse (they poll it).
func (h *InternalHandler) BlockedUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user_ids": h.blocks.BlockedUserIDs()})
}

// Register adds the internal routes to r under key.
func (h *InternalHandler) Register(r gin.IRouter, key string) {
	internal := r.Group("/internal/v1")
	internal.Use(InternalKeyRequired(key))
	internal.POST("/email-blocks", h.BlockEmail)
	internal.GET("/email-blocks", h.ListEmailBlocks)
	internal.GET("/email-blocks/:email", h.EmailBlockStatus)
	internal.DELETE("/email-blocks/:email", h.UnblockEmail)
	internal.GET("/blocked-users", h.BlockedUsers)
}
