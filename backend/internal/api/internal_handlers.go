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
	blocks      *services.BlockService
	suspensions *services.SuspensionService
	pauses      *services.CodePauseService
}

func NewInternalHandler(blocks *services.BlockService, suspensions *services.SuspensionService) *InternalHandler {
	return &InternalHandler{blocks: blocks, suspensions: suspensions}
}

// WithPauses serves /internal/v1/code-pauses.
func (h *InternalHandler) WithPauses(pauses *services.CodePauseService) *InternalHandler {
	h.pauses = pauses
	return h
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
// and chat refuse (user_ids) or keep read-only (suspended); they poll it.
func (h *InternalHandler) BlockedUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user_ids": h.blocks.BlockedUserIDs(), "suspended": h.suspensions.SuspendedUsers()})
}

type suspendRequest struct {
	Email  string `json:"email" binding:"required"`
	Hours  int    `json:"hours"` // 0: the next step on the ladder
	Reason string `json:"reason"`
	By     string `json:"by"`
}

type suspensionResponse struct {
	Email     string     `json:"email"`
	Suspended bool       `json:"suspended"`
	Until     *time.Time `json:"until,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	Hours     int        `json:"hours,omitempty"`
	Step      int        `json:"step,omitempty"`       // this one's place in a row
	InRow     int        `json:"in_row"`               // suspensions in a row so far
	NextHours int        `json:"next_hours,omitempty"` // what a bare suspend gives now
}

func hours(d time.Duration) int { return int(d / time.Hour) }

// Suspend: POST /internal/v1/suspensions {email, hours (0: the ladder), reason, by}.
func (h *InternalHandler) Suspend(c *gin.Context) {
	var req suspendRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Hours < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	s, err := h.suspensions.Suspend(c.Request.Context(), req.Email, time.Duration(req.Hours)*time.Hour, req.Reason, req.By)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	length := s.EndsAt.Sub(s.StartsAt)
	log.Printf("suspension event=suspended email=%q user=%d hours=%d step=%d by=%q", s.Email, s.UserID, hours(length), s.Step, req.By)
	until := s.EndsAt
	c.JSON(http.StatusOK, suspensionResponse{Email: s.Email, Suspended: true, Until: &until, Reason: s.Reason,
		Hours: hours(length), Step: s.Step, InRow: s.Step})
}

// Unsuspend: DELETE /internal/v1/suspensions/:email.
func (h *InternalHandler) Unsuspend(c *gin.Context) {
	email := services.NormalizeEmail(c.Param("email"))
	lifted, err := h.suspensions.Unsuspend(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if !lifted {
		c.JSON(http.StatusNotFound, gin.H{"error": "that account isn't suspended"})
		return
	}
	log.Printf("suspension event=lifted email=%q", email)
	c.JSON(http.StatusOK, suspensionResponse{Email: email})
}

// SuspensionStatus: GET /internal/v1/suspensions/:email.
func (h *InternalHandler) SuspensionStatus(c *gin.Context) {
	email := services.NormalizeEmail(c.Param("email"))
	status, err := h.suspensions.Status(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	out := suspensionResponse{Email: email, InRow: status.InRow, NextHours: hours(status.Next)}
	if a := status.Active; a != nil {
		end := a.End()
		out.Suspended, out.Until, out.Reason = true, &end, a.Reason
	}
	c.JSON(http.StatusOK, out)
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
	if h.pauses != nil {
		internal.POST("/code-pauses", h.PauseCodes)
		internal.GET("/code-pauses/:email", h.CodePauseStatus)
		internal.DELETE("/code-pauses/:email", h.UnpauseCodes)
	}
	internal.POST("/suspensions", h.Suspend)
	internal.GET("/suspensions/:email", h.SuspensionStatus)
	internal.DELETE("/suspensions/:email", h.Unsuspend)
}

type pauseRequest struct {
	Email string `json:"email" binding:"required"`
	Hours int    `json:"hours"` // 1 to 72
	By    string `json:"by"`
}

type pauseResponse struct {
	Email  string     `json:"email"`
	Paused bool       `json:"paused"`
	Until  *time.Time `json:"until,omitempty"`
}

// PauseCodes: POST /internal/v1/code-pauses {email, hours, by}.
func (h *InternalHandler) PauseCodes(c *gin.Context) {
	var req pauseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	p, err := h.pauses.Pause(c.Request.Context(), req.Email, time.Duration(req.Hours)*time.Hour, req.By)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	log.Printf("codes event=paused email=%q hours=%d by=%q", p.Email, req.Hours, req.By)
	c.JSON(http.StatusOK, pauseResponse{Email: p.Email, Paused: true, Until: &p.Until})
}

// UnpauseCodes: DELETE /internal/v1/code-pauses/:email.
func (h *InternalHandler) UnpauseCodes(c *gin.Context) {
	email := services.NormalizeEmail(c.Param("email"))
	removed, err := h.pauses.Unpause(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}
	if !removed {
		c.JSON(http.StatusNotFound, gin.H{"error": "codes to that address aren't paused"})
		return
	}
	log.Printf("codes event=unpaused email=%q", email)
	c.JSON(http.StatusOK, pauseResponse{Email: email})
}

// CodePauseStatus: GET /internal/v1/code-pauses/:email.
func (h *InternalHandler) CodePauseStatus(c *gin.Context) {
	email := services.NormalizeEmail(c.Param("email"))
	p, err := h.pauses.Status(c.Request.Context(), email)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}
	out := pauseResponse{Email: email}
	if p != nil {
		out.Paused, out.Until = true, &p.Until
	}
	c.JSON(http.StatusOK, out)
}
