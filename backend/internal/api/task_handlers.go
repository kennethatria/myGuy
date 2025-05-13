package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"myguy/internal/services"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService *services.TaskService
}

func NewTaskHandler(taskService *services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

type CreateTaskRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Fee         float64 `json:"fee" binding:"required,min=0"`
	Deadline    string  `json:"deadline" binding:"required"`
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deadline, err := time.Parse(time.RFC3339, req.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deadline format. Use RFC3339 format (e.g., 2025-12-31T00:00:00Z)"})
		return
	}

	// Add debug logging
	fmt.Printf("Parsed deadline: %v, Now: %v\n", deadline.UTC(), time.Now().UTC())

	userID := c.GetUint("userID") // Set by auth middleware
	input := services.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Fee:         req.Fee,
		Deadline:    deadline,
		CreatedBy:   userID,
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), input)
	if err != nil {
		switch err {
		case services.ErrInvalidDeadline:
			c.JSON(http.StatusBadRequest, gin.H{"error": "deadline must be in the future"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		}
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *TaskHandler) GetTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	task, err := h.taskService.GetTaskByID(c.Request.Context(), uint(id))
	if err != nil {
		switch err {
		case services.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	status := c.Query("status")
	createdBy := c.Query("created_by")

	filters := make(map[string]interface{})
	if status != "" {
		filters["status"] = status
	}
	if createdBy != "" {
		userID, err := strconv.ParseUint(createdBy, 10, 64)
		if err == nil {
			filters["created_by"] = uint(userID)
		}
	}

	tasks, err := h.taskService.ListTasks(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")
	input := services.UpdateTaskInput{
		ID:          uint(id),
		Title:       req.Title,
		Description: req.Description,
		Fee:         req.Fee,
		Deadline:    req.Deadline,
		UpdatedBy:   userID,
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), input)
	if err != nil {
		switch err {
		case services.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this task"})
		case services.ErrInvalidDeadline:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Deadline must be in the future"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	userID := c.GetUint("userID")
	err = h.taskService.DeleteTask(c.Request.Context(), uint(id), userID)
	if err != nil {
		switch err {
		case services.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this task"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
}
