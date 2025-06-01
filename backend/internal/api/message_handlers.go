package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// EditMessage handles message editing
func (h *Handler) EditMessage(c *gin.Context) {
	userID := c.GetUint("userID")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.messageService.EditMessage(c.Request.Context(), uint(messageID), userID, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

// DeleteMessage handles soft message deletion
func (h *Handler) DeleteMessage(c *gin.Context) {
	userID := c.GetUint("userID")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	err = h.messageService.DeleteMessage(c.Request.Context(), uint(messageID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "message deleted successfully"})
}

// MarkMessageAsRead handles marking a message as read
func (h *Handler) MarkMessageAsRead(c *gin.Context) {
	userID := c.GetUint("userID")
	messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	message, err := h.messageService.MarkAsRead(c.Request.Context(), uint(messageID), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, message)
}

// GetUserConversations handles getting all user conversations
func (h *Handler) GetUserConversations(c *gin.Context) {
	userID := c.GetUint("userID")

	conversations, err := h.messageService.GetUserConversations(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, conversations)