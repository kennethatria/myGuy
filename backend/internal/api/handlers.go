package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/proximity"
	"myguy/internal/services"
)

type Handler struct {
	authService    *services.AuthService
	userService    *services.UserService
	taskService    *services.TaskService
	reviewService  *services.ReviewService
	authMiddleware *middleware.JWTAuthMiddleware
}

func NewHandler(
	authService *services.AuthService,
	userService *services.UserService,
	taskService *services.TaskService,
	reviewService *services.ReviewService,
	authMiddleware *middleware.JWTAuthMiddleware,
) *Handler {
	return &Handler{
		authService:    authService,
		userService:    userService,
		taskService:    taskService,
		reviewService:  reviewService,
		authMiddleware: authMiddleware,
	}
}

type requestCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestLoginCode emails a one-time sign-in code. The response is the same
// whether or not an account exists for the address.
func (h *Handler) RequestLoginCode(c *gin.Context) {
	var req requestCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enter a valid email address"})
		return
	}

	err := h.authService.RequestCode(c.Request.Context(), req.Email)
	if errors.Is(err, services.ErrTooManyRequests) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("request login code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send code"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message": "code sent"})
}

type verifyCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

// VerifyLoginCode signs in an existing user, or returns a short-lived
// signup token when the verified email has no account yet.
func (h *Handler) VerifyLoginCode(c *gin.Context) {
	var req verifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	result, err := h.authService.VerifyCode(c.Request.Context(), req.Email, req.Code)
	if errors.Is(err, services.ErrInvalidCode) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		log.Printf("verify login code: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify code"})
		return
	}

	if result.NewAccount {
		signupToken, err := h.authMiddleware.GenerateSignupToken(result.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"signup_token": signupToken})
		return
	}

	h.respondWithSession(c, http.StatusOK, result.User)
}

type completeSignupRequest struct {
	SignupToken string `json:"signup_token" binding:"required"`
	FullName    string `json:"full_name" binding:"required"`
}

// CompleteSignup creates the account for a verified email and signs it in.
func (h *Handler) CompleteSignup(c *gin.Context) {
	var req completeSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email, err := h.authMiddleware.ValidateSignupToken(req.SignupToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "sign-up session expired, request a new code"})
		return
	}

	user, err := h.authService.CompleteSignup(c.Request.Context(), email, req.FullName)
	switch {
	case errors.Is(err, services.ErrFullNameRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	case errors.Is(err, services.ErrEmailExists):
		c.JSON(http.StatusConflict, gin.H{"error": "an account already exists for this email, sign in instead"})
		return
	case err != nil:
		log.Printf("complete signup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	h.respondWithSession(c, http.StatusCreated, user)
}

func (h *Handler) respondWithSession(c *gin.Context, status int, user *models.UserResponse) {
	token, err := h.authMiddleware.GenerateToken(user.ID, user.Username, user.Email, user.FullName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(status, gin.H{
		"user":  user,
		"token": token,
	})
}

// createTaskRequest is a sticky note: a short headline and body. There is no
// fee or deadline; price is agreed in chat and every gig runs 24 hours.
type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	// Optional rough location, already snapped to a cell by the browser
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

func (h *Handler) CreateTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	location, err := proximity.Parse(req.Lat, req.Lng)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	userID := c.GetUint("userID")
	task, err := h.taskService.CreateTask(c.Request.Context(), services.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		CreatedBy:   userID,
		Location:    location,
	})

	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, task)
}

func (h *Handler) GetTask(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.taskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	c.JSON(http.StatusOK, taskForViewer(*task, c.GetUint("userID")))
}

// UpdateTask updates a task with new details
func (h *Handler) UpdateTask(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.GetUint("userID")
	input := services.UpdateTaskInput{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		UpdatedBy:   userID,
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTaskNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case errors.Is(err, services.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this task"})
		case services.IsGigTextError(err):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		}
		return
	}

	c.JSON(http.StatusOK, taskForViewer(*task, userID))
}

// ListTasks returns all tasks with optional filtering, search, sorting, and pagination
func (h *Handler) ListTasks(c *gin.Context) {
	// Create filters from query parameters
	filters := make(map[string]interface{})
	
	// Add status filter if provided
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	
	// Add search query if provided
	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}
	
	// Add deadline filter (tasks due before a certain date)
	if deadline := c.Query("deadline_before"); deadline != "" {
		filters["deadline_before"] = deadline
	}
	
	// Sorting
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy // deadline (expiring soonest), created_at
	}
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		filters["sort_order"] = sortOrder // asc, desc
	}
	
	// Pagination
	page := 1
	perPage := 20
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if perPageNum, err := strconv.Atoi(pp); err == nil && perPageNum > 0 && perPageNum <= 100 {
			perPage = perPageNum
		}
	}
	filters["page"] = page
	filters["per_page"] = perPage
	
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
		userID, err := parseID(createdBy)
		if err == nil {
			filters["created_by"] = userID
		}
	} else if excludeCreatedBy := c.Query("exclude_created_by"); excludeCreatedBy != "" {
		// Exclude tasks created by a specific user (useful for browsing)
		userID, err := parseID(excludeCreatedBy)
		if err == nil {
			filters["exclude_created_by"] = userID
		}
	}
	
	// The viewer's rough position ("lat,lng"), for distance sort and tags
	near, err := parseNear(c.Query("near"))
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	// Get tasks with the provided filters
	var result *services.PaginatedTasksResult
	if near != nil && c.Query("sort_by") == "distance" {
		result, err = h.taskService.ListTasksNear(c.Request.Context(), filters, *near)
	} else {
		result, err = h.taskService.ListTasksWithPagination(c.Request.Context(), filters)
		if err == nil && near != nil {
			h.taskService.TagDistances(result.Tasks, *near)
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks"})
		return
	}
	result.Tasks = tasksForViewer(result.Tasks, userID)

	c.JSON(http.StatusOK, result)
}

// GetUserTasks returns tasks created by the current user
func (h *Handler) GetUserTasks(c *gin.Context) {
	userID := c.GetUint("userID")
	
	tasks, err := h.taskService.ListUserTasks(c.Request.Context(), userID, "creator")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user tasks"})
		return
	}

	c.JSON(http.StatusOK, tasksForViewer(tasks, userID))
}

// GetAssignedTasks returns tasks assigned to the current user
// Excludes tasks the user created themselves (only shows tasks from other users)
func (h *Handler) GetAssignedTasks(c *gin.Context) {
	userID := c.GetUint("userID")
	
	// Get all tasks assigned to the user
	tasks, err := h.taskService.ListUserTasks(c.Request.Context(), userID, "assigned")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve assigned tasks"})
		return
	}
	
	// Check if we should exclude self-assigned tasks (tasks the user both created and is assigned to)
	if excludeSelf := c.Query("exclude_self_assigned"); excludeSelf == "true" {
		// Filter out tasks where createdBy == current user
		filteredTasks := make([]models.Task, 0, len(tasks))
		for _, task := range tasks {
			if task.CreatedBy != userID {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}

	c.JSON(http.StatusOK, tasksForViewer(tasks, userID))
}

type applyForTaskRequest struct {
	Message string `json:"message" binding:"required"`
}

func (h *Handler) ApplyForTask(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req applyForTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.GetUint("userID")
	err = h.taskService.ApplyForTask(c.Request.Context(), taskID, userID, req.Message)
	switch {
	case errors.Is(err, services.ErrTaskNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	case errors.Is(err, services.ErrAlreadyApplied), errors.Is(err, services.ErrTaskNotOpen):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	case err != nil:
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusCreated)
}

type respondToApplicationRequest struct {
	Status string `json:"status" binding:"required,oneof=accepted declined"`
}

func (h *Handler) RespondToApplication(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	applicationID, err := parseID(c.Param("applicationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	var req respondToApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.GetUint("userID")
	
	// Verify the user is the task creator
	task, err := h.taskService.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	
	if task.CreatedBy != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only task creator can respond to applications"})
		return
	}

	// Update application status
	var updatedTask *models.Task
	if req.Status == "accepted" {
		updatedTask, err = h.taskService.AssignTask(c.Request.Context(), taskID, applicationID)
		if err != nil {
			respondApplicationError(c, err)
			return
		}
	} else {
		// For declined, just update the application status
		err = h.taskService.DeclineApplication(c.Request.Context(), taskID, applicationID)
		if err != nil {
			respondApplicationError(c, err)
			return
		}
	}

	if updatedTask != nil {
		// Return simple success response to avoid serialization issues
		c.JSON(http.StatusOK, gin.H{
			"message": "Application accepted successfully",
			"task_id": updatedTask.ID,
			"status": updatedTask.Status,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "Application declined"})
	}
}


type createReviewRequest struct {
	Rating   int    `json:"rating" binding:"required,min=1,max=5"`
	Comment  string `json:"comment"`
}

func (h *Handler) CreateReview(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	var req createReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	reviewerID := c.GetUint("userID")
	
	// Fetch the task to determine who should be reviewed
	task, err := h.taskService.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	
	// Determine who is being reviewed based on the reviewer's role
	var reviewedUserID uint
	if task.CreatedBy == reviewerID {
		// Task creator is reviewing the assignee
		if task.AssignedTo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "task has no assignee to review"})
			return
		}
		reviewedUserID = *task.AssignedTo
	} else if task.AssignedTo != nil && *task.AssignedTo == reviewerID {
		// Assignee is reviewing the task creator
		reviewedUserID = task.CreatedBy
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not a participant in this task"})
		return
	}
	
	review, err := h.reviewService.CreateReview(c.Request.Context(), services.CreateReviewInput{
		TaskID:         taskID,
		ReviewerID:     reviewerID,
		ReviewedUserID: reviewedUserID,
		Rating:         req.Rating,
		Comment:        req.Comment,
	})

	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, review)
}

func (h *Handler) GetUserReviews(c *gin.Context) {
	userID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	reviews, err := h.reviewService.GetUserReviews(c.Request.Context(), userID)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	viewerID := c.GetUint("userID")
	for i := range reviews {
		reviews[i].Reviewer = publicUser(reviews[i].Reviewer, viewerID)
		reviews[i].ReviewedUser = publicUser(reviews[i].ReviewedUser, viewerID)
		reviews[i].Task = taskForViewer(reviews[i].Task, viewerID)
	}

	c.JSON(http.StatusOK, reviews)
}

// GetUserByID handles retrieving a user by their ID
func (h *Handler) GetUserByID(c *gin.Context) {
	userID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), userID)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		}
		return
	}

	// Another user's profile never includes their contact details.
	if user.ID != c.GetUint("userID") {
		user.Email = ""
		user.PhoneNumber = ""
	}

	c.JSON(http.StatusOK, user)
}

// Profile edits change the display name and bio only. The email is the
// sign-in identity (codes are sent to it), so it can't be changed here
// without verifying the new address.
type updateProfileRequest struct {
	FullName string `json:"full_name" binding:"required,max=100"`
	Bio      string `json:"bio" binding:"max=500"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "full name is required"})
		return
	}

	userID := c.GetUint("userID") // Set by JWT middleware
	// Bio is saved as given, so clearing it works.
	user, err := h.userService.UpdateProfile(c.Request.Context(), userID, fullName, strings.TrimSpace(req.Bio))
	if errors.Is(err, services.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateTaskStatusRequest contains the data for updating a task's status
type UpdateTaskStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=open in_progress completed cancelled"`
	// Optional fresh rough location when reposting (status open)
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

// UpdateTaskStatus updates the status of a task
func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var req UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	location, err := proximity.Parse(req.Lat, req.Lng)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	userID := c.GetUint("userID")
	task, err := h.taskService.UpdateTaskStatus(c.Request.Context(), id, req.Status, userID)
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

	if req.Status == "open" {
		h.taskService.SaveLocation(task.ID, location)
	}

	c.JSON(http.StatusOK, taskForViewer(*task, userID))
}

// DeleteTask deletes a task
func (h *Handler) DeleteTask(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	userID := c.GetUint("userID")
	err = h.taskService.DeleteTask(c.Request.Context(), id, userID)
	if err != nil {
		switch err {
		case services.ErrTaskNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		case services.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this task"})
		case services.ErrTaskWasAssigned:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
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

// GetTaskApplications retrieves all applications for a specific task
func (h *Handler) GetTaskApplications(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	task, err := h.taskService.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	applications, err := h.taskService.GetTaskApplications(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve applications"})
		return
	}

	task.Applications = applications
	c.JSON(http.StatusOK, taskForViewer(*task, c.GetUint("userID")).Applications)
}

// parseID parses a database id from a path or query value. 32 bits is far
// more than any table holds and fits uint on every platform, so the
// conversion can't overflow.
func parseID(s string) (uint, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

func respondApplicationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrApplicationNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrTaskNotOpen):
		c.JSON(http.StatusConflict, gin.H{"error": "this task has already been assigned"})
	case errors.Is(err, services.ErrApplicationNotPending):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		respondError(c, http.StatusBadRequest, err)
	}
}

// taskForViewer limits what a task response reveals to viewerID: the owner
// sees every application, an applicant only their own, anyone else none; and
// nobody's email or phone number is exposed except the viewer's own.
func taskForViewer(task models.Task, viewerID uint) models.Task {
	applications := make([]models.Application, 0, len(task.Applications))
	for _, app := range task.Applications {
		if task.CreatedBy == viewerID || app.ApplicantID == viewerID {
			app.Applicant = publicUser(app.Applicant, viewerID)
			applications = append(applications, app)
		}
	}
	task.Applications = applications

	task.Creator = publicUser(task.Creator, viewerID)
	if task.Assignee != nil {
		assignee := publicUser(*task.Assignee, viewerID)
		task.Assignee = &assignee
	}
	return task
}

func tasksForViewer(tasks []models.Task, viewerID uint) []models.Task {
	visible := make([]models.Task, len(tasks))
	for i, task := range tasks {
		visible[i] = taskForViewer(task, viewerID)
	}
	return visible
}

// respondError answers with err's message when it was written for users,
// and otherwise logs it and answers with a generic 500 so database details
// never reach clients.
func respondError(c *gin.Context, status int, err error) {
	if services.IsUserFacing(err) {
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	log.Printf("%s %s failed: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong; please try again"})
}

// parseNear reads an optional "lat,lng" query value as a rough location.
func parseNear(raw string) (*proximity.Location, error) {
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return nil, proximity.ErrInvalidLocation
	}
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return nil, proximity.ErrInvalidLocation
	}
	return proximity.Parse(&lat, &lng)
}

func publicUser(user models.User, viewerID uint) models.User {
	if user.ID != viewerID {
		user.Email = ""
		user.PhoneNumber = ""
	}
	return user
}

// GetApplicationParticipants lets the chat service check that a user may chat
// about an application, and with whom.
func (h *Handler) GetApplicationParticipants(c *gin.Context) {
	applicationID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	participants, err := h.taskService.GetApplicationParticipants(c.Request.Context(), applicationID, c.GetUint("userID"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	c.JSON(http.StatusOK, participants)
}

// GetMyTaskReview reports whether the current user has already reviewed the
// task, so the UI only offers a review once.
func (h *Handler) GetMyTaskReview(c *gin.Context) {
	taskID, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task ID"})
		return
	}

	review, err := h.reviewService.GetTaskReview(c.Request.Context(), taskID, c.GetUint("userID"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, gin.H{"reviewed": false})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"reviewed": true, "review": review})
}

// GetUserApplications lists the current user's applications with each task's
// status, for the dashboard's "My applications" tab.
func (h *Handler) GetUserApplications(c *gin.Context) {
	userID := c.GetUint("userID")
	applications, err := h.taskService.ListUserApplications(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve applications"})
		return
	}

	for i := range applications {
		applications[i].Task = taskForViewer(applications[i].Task, userID)
		applications[i].Applicant = publicUser(applications[i].Applicant, userID)
	}
	c.JSON(http.StatusOK, applications)
}

