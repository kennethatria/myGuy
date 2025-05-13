package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"myguy/internal/middleware"
	"myguy/internal/services"
)

type Handler struct {
	userService    *services.UserService
	taskService    *services.TaskService
	messageService *services.MessageService
	reviewService  *services.ReviewService
	authMiddleware *middleware.JWTAuthMiddleware
}

func NewHandler(
	userService *services.UserService,
	taskService *services.TaskService,
	messageService *services.MessageService,
	reviewService *services.ReviewService,
	authMiddleware *middleware.JWTAuthMiddleware,
) *Handler {
	return &Handler{
		userService:    userService,
		taskService:    taskService,
		messageService: messageService,
		reviewService:  reviewService,
		authMiddleware: authMiddleware,
	}
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Register(c.Request.Context(), services.RegisterUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := h.authMiddleware.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":  user,
		"token": token,
	})
}

type createTaskRequest struct {
	Title       string  `json:"title" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Fee         float64 `json:"fee" binding:"required"`
	Deadline    string  `json:"deadline" binding:"required"`
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Parse the deadline string to time.Time
	deadline, err := time.Parse(time.RFC3339, req.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deadline format. Must be RFC3339 format (e.g., 2025-05-15T12:00:00Z)"})
		return
	}

	userID := c.GetUint("userID")
	task, err := h.taskService.CreateTask(c.Request.Context(), services.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Fee:         req.Fee,
		Deadline:    deadline,
		CreatedBy:   userID,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *Handler) GetTask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}

// UpdateTask updates a task with new details
func (h *Handler) UpdateTask(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Parse the deadline string to time.Time
	deadline, err := time.Parse(time.RFC3339, req.Deadline)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deadline format. Must be RFC3339 format (e.g., 2025-05-15T12:00:00Z)"})
		return
	}

	userID := c.GetUint("userID")
	input := services.UpdateTaskInput{
		ID:          uint(id),
		Title:       req.Title,
		Description: req.Description,
		Fee:         req.Fee,
		Deadline:    deadline,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Deadline must be at least one day (24 hours) in the future"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

// ListTasks returns all tasks with optional filtering
func (h *Handler) ListTasks(c *gin.Context) {
	// Create filters from query parameters
	filters := make(map[string]interface{})
	
	// Add status filter if provided
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	
	// Check for specific filters
	userID := c.GetUint("userID")
	
	// Filter for user's created tasks
	if created := c.Query("created"); created == "true" {
		filters["created_by"] = userID
	} else if assigned := c.Query("assigned"); assigned == "true" {
		// Filter for tasks assigned to the user
		filters["assigned_to"] = userID
	} else if createdBy := c.Query("created_by"); createdBy != "" {
		// Add created_by filter if explicitly provided
		userID, err := strconv.ParseUint(createdBy, 10, 64)
		if err == nil {
			filters["created_by"] = uint(userID)
		}
	}
	
	// Get tasks with the provided filters
	tasks, err := h.taskService.ListTasks(c.Request.Context(), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	
	c.JSON(http.StatusOK, tasks)
}

// GetUserTasks returns tasks created by the current user
func (h *Handler) GetUserTasks(c *gin.Context) {
	userID := c.GetUint("userID")
	
	tasks, err := h.taskService.ListUserTasks(c.Request.Context(), userID, "creator")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user tasks"})
		return
	}
	
	c.JSON(http.StatusOK, tasks)
}

// GetAssignedTasks returns tasks assigned to the current user
func (h *Handler) GetAssignedTasks(c *gin.Context) {
	userID := c.GetUint("userID")
	
	tasks, err := h.taskService.ListUserTasks(c.Request.Context(), userID, "assigned")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve assigned tasks"})
		return
	}
	
	c.JSON(http.StatusOK, tasks)
}

type applyForTaskRequest struct {
	ProposedFee float64 `json:"proposed_fee" binding:"required"`
	Message     string  `json:"message" binding:"required"`
}

func (h *Handler) ApplyForTask(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req applyForTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")
	err = h.taskService.ApplyForTask(c.Request.Context(), uint(taskID), userID, req.ProposedFee, req.Message)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

type createMessageRequest struct {
	Content     string `json:"content" binding:"required"`
	RecipientID uint   `json:"recipient_id" binding:"required"`
}

func (h *Handler) CreateMessage(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req createMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	senderID := c.GetUint("userID")
	message, err := h.messageService.CreateMessage(c.Request.Context(), services.CreateMessageInput{
		TaskID:      uint(taskID),
		SenderID:    senderID,
		RecipientID: req.RecipientID,
		Content:     req.Content,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *Handler) GetTaskMessages(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	messages, err := h.messageService.GetTaskMessages(c.Request.Context(), uint(taskID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

type createReviewRequest struct {
	Rating   int    `json:"rating" binding:"required,min=1,max=5"`
	Comment  string `json:"comment"`
}

func (h *Handler) CreateReview(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req createReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID := c.GetUint("userID")
	review, err := h.reviewService.CreateReview(c.Request.Context(), services.CreateReviewInput{
		TaskID:     uint(taskID),
		ReviewerID: reviewerID,
		Rating:     req.Rating,
		Comment:    req.Comment,
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *Handler) GetUserReviews(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	reviews, err := h.reviewService.GetUserReviews(c.Request.Context(), uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reviews)
}

// GetUserByID handles retrieving a user by their ID
func (h *Handler) GetUserByID(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), uint(userID))
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

type updateProfileRequest struct {
	FullName    string `json:"full_name"`
	Email       string `json:"email" binding:"email"`
	PhoneNumber string `json:"phone_number"`
	Bio         string `json:"bio"`
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetUint("userID") // Set by JWT middleware
	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID") // Set by JWT middleware
	user, err := h.userService.UpdateUser(c.Request.Context(), services.UpdateUserInput{
		ID:          userID,
		FullName:    req.FullName,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Bio:         req.Bio,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateTaskStatusRequest contains the data for updating a task's status
type UpdateTaskStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open in_progress completed cancelled"`
}

// UpdateTaskStatus updates the status of a task
func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")
	task, err := h.taskService.UpdateTaskStatus(c.Request.Context(), uint(id), req.Status, userID)
	if err != nil {
		switch err {
		case services.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this task's status"})
		case services.ErrInvalidStatus:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status transition"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task status"})
		}
		return
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask deletes a task
func (h *Handler) DeleteTask(c *gin.Context) {
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

// GetServerTime returns the current server time and a valid deadline
func (h *Handler) GetServerTime(c *gin.Context) {
	now := time.Now().UTC()
	
	// Create a response with current time and valid deadlines
	response := gin.H{
		"current_time": now.Format(time.RFC3339),
		"valid_deadline_examples": []string{
			now.AddDate(0, 0, 2).Format(time.RFC3339),  // 2 days from now
			now.AddDate(0, 0, 7).Format(time.RFC3339),  // 1 week from now
			now.AddDate(0, 1, 0).Format(time.RFC3339),  // 1 month from now
		},
		"minimum_valid_deadline": now.AddDate(0, 0, 1).Format(time.RFC3339), // 1 day from now
	}
	
	c.JSON(http.StatusOK, response)
}
