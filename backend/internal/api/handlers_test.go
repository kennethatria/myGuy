package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/services"
	"myguy/tests"
)

func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

func setupTestRouter() (*gin.Engine, *Handler, *tests.MockUserRepository, *tests.MockTaskRepository, *tests.MockReviewRepository, *tests.MockApplicationRepository) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	mockUserRepo := new(tests.MockUserRepository)
	mockTaskRepo := new(tests.MockTaskRepository)
	mockReviewRepo := new(tests.MockReviewRepository)
	mockAppRepo := new(tests.MockApplicationRepository)

	userService := services.NewUserService(mockUserRepo)
	taskService := services.NewTaskService(mockTaskRepo, mockAppRepo)
	reviewService := services.NewReviewService(mockReviewRepo, mockTaskRepo, mockUserRepo)
	authMiddleware := middleware.NewJWTAuthMiddleware("test-secret")

	handler := NewHandler(userService, taskService, reviewService, authMiddleware)

	return router, handler, mockUserRepo, mockTaskRepo, mockReviewRepo, mockAppRepo
}

func TestHandler_Register(t *testing.T) {
	router, handler, mockUserRepo, _, _, _ := setupTestRouter()
	router.POST("/register", handler.Register)

	t.Run("successful registration", func(t *testing.T) {
		reqBody := registerRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
			FullName: "Test User",
		}

		mockUserRepo.On("GetByEmail", mock.Anything, reqBody.Email).Return(nil, gorm.ErrRecordNotFound)
		mockUserRepo.On("GetByUsername", mock.Anything, reqBody.Username).Return(nil, gorm.ErrRecordNotFound)
		mockUserRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil).Run(func(args mock.Arguments) {
			user := args.Get(1).(*models.User)
			user.ID = 1
		})

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		var user models.UserResponse
		err := json.Unmarshal(resp.Body.Bytes(), &user)
		assert.NoError(t, err)
		assert.Equal(t, reqBody.Username, user.Username)
		assert.Equal(t, uint(1), user.ID)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("email already exists", func(t *testing.T) {
		reqBody := registerRequest{
			Username: "testuser",
			Email:    "exists@example.com",
			Password: "password123",
			FullName: "Test User",
		}

		mockUserRepo.On("GetByEmail", mock.Anything, reqBody.Email).Return(&models.User{ID: 1}, nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestHandler_Login(t *testing.T) {
	router, handler, mockUserRepo, _, _, _ := setupTestRouter()
	router.POST("/login", handler.Login)

	t.Run("successful login", func(t *testing.T) {
		reqBody := loginRequest{
			Email:    "test@example.com",
			Password: "password123",
		}

		user := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    reqBody.Email,
			Password: hashPassword(reqBody.Password),
			FullName: "Test User",
		}

		mockUserRepo.On("GetByEmail", mock.Anything, reqBody.Email).Return(user, nil)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var result map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &result)
		assert.NotNil(t, result["token"])
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("invalid credentials", func(t *testing.T) {
		reqBody := loginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		mockUserRepo.On("GetByEmail", mock.Anything, reqBody.Email).Return(nil, gorm.ErrRecordNotFound)

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
	})
}

func TestHandler_GetProfile(t *testing.T) {
	router, handler, mockUserRepo, _, _, _ := setupTestRouter()
	
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.GET("/profile", handler.GetProfile)

	t.Run("successful get profile", func(t *testing.T) {
		user := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
		}

		mockUserRepo.On("GetByID", mock.Anything, uint(1)).Return(user, nil)

		req, _ := http.NewRequest(http.MethodGet, "/profile", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var result models.UserResponse
		json.Unmarshal(resp.Body.Bytes(), &result)
		assert.Equal(t, user.Username, result.Username)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestHandler_CreateTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.POST("/tasks", handler.CreateTask)

	t.Run("successful task creation", func(t *testing.T) {
		deadline := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
		reqBody := createTaskRequest{
			Title:       "Test Task",
			Description: "Description",
			Fee:         100.0,
			Deadline:    deadline,
		}

		mockTaskRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil).Run(func(args mock.Arguments) {
			task := args.Get(1).(*models.Task)
			task.ID = 1
		})

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("invalid deadline", func(t *testing.T) {
		reqBody := createTaskRequest{
			Title:       "Test Task",
			Description: "Description",
			Fee:         100.0,
			Deadline:    "invalid-date",
		}

		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestHandler_GetTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.GET("/tasks/:id", handler.GetTask)

	t.Run("successful get task", func(t *testing.T) {
		task := &models.Task{
			ID:    1,
			Title: "Test Task",
		}

		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)

		req, _ := http.NewRequest(http.MethodGet, "/tasks/1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		mockTaskRepo.On("GetByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodGet, "/tasks/999", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestHandler_UpdateTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.PUT("/tasks/:id", handler.UpdateTask)

	t.Run("successful update", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1, Title: "Old Title"}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockTaskRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

		deadline := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
		reqBody := createTaskRequest{Title: "New Title", Description: "New Desc", Fee: 200, Deadline: deadline}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 2, Title: "Old Title"}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)

		deadline := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
		reqBody := createTaskRequest{Title: "New Title", Description: "New Desc", Fee: 200, Deadline: deadline}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
	})
}

func TestHandler_DeleteTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.DELETE("/tasks/:id", handler.DeleteTask)

	t.Run("successful delete", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockTaskRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/tasks/1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(nil, gorm.ErrRecordNotFound)

		req, _ := http.NewRequest(http.MethodDelete, "/tasks/1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestHandler_UpdateTaskStatus(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.PATCH("/tasks/:id/status", handler.UpdateTaskStatus)

	t.Run("successful status update", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockTaskRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

		reqBody := UpdateTaskStatusRequest{Status: "in_progress"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPatch, "/tasks/1/status", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("invalid status", func(t *testing.T) {
		reqBody := gin.H{"status": "invalid"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPatch, "/tasks/1/status", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestHandler_ListTasks(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.GET("/tasks", handler.ListTasks)

	t.Run("successful list tasks", func(t *testing.T) {
		tasks := []models.Task{{ID: 1, Title: "Task 1"}}

		mockTaskRepo.On("ListWithPagination", mock.Anything, mock.Anything).Return(tasks, nil)
		mockTaskRepo.On("Count", mock.Anything, mock.Anything).Return(int64(1), nil)

		req, _ := http.NewRequest(http.MethodGet, "/tasks", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})
}

func TestHandler_ApplyForTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(2))
		c.Next()
	})
	router.POST("/tasks/:id/apply", handler.ApplyForTask)

	t.Run("successful application", func(t *testing.T) {
		task := &models.Task{ID: 1, Status: "open", CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Application")).Return(nil)

		reqBody := applyForTaskRequest{ProposedFee: 100, Message: "I'm interested"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/apply", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockAppRepo.AssertExpectations(t)
	})
}

func TestHandler_CreateReview(t *testing.T) {
	router, handler, _, mockTaskRepo, mockReviewRepo, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.POST("/tasks/:id/reviews", handler.CreateReview)

	t.Run("successful review creation", func(t *testing.T) {
		task := &models.Task{ID: 1, Status: "completed", CreatedBy: 1, AssignedTo: func(u uint) *uint { return &u }(2)}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockReviewRepo.On("GetTaskReview", mock.Anything, uint(1), uint(1)).Return(nil, gorm.ErrRecordNotFound)
		mockReviewRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Review")).Return(nil).Run(func(args mock.Arguments) {
			review := args.Get(1).(*models.Review)
			review.ID = 1
		})

		reqBody := createReviewRequest{
			Rating:  5,
			Comment: "Excellent!",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/reviews", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockReviewRepo.AssertExpectations(t)
	})
}

func TestHandler_GetServerTime(t *testing.T) {
	router, handler, _, _, _, _ := setupTestRouter()
	router.GET("/time", handler.GetServerTime)

	t.Run("successful get server time", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/time", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var result map[string]interface{}
		json.Unmarshal(resp.Body.Bytes(), &result)
		assert.NotNil(t, result["current_time"])
	})
}

func TestHandler_GetTaskApplications(t *testing.T) {
	router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.GET("/tasks/:id/applications", handler.GetTaskApplications)

	t.Run("successful get applications", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("ListByTask", mock.Anything, uint(1)).Return([]models.Application{}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/tasks/1/applications", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockAppRepo.AssertExpectations(t)
	})
}

func TestHandler_GetUserTasks(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.GET("/user/tasks", handler.GetUserTasks)

	t.Run("successful get user tasks", func(t *testing.T) {
		mockTaskRepo.On("ListByUser", mock.Anything, uint(1), "creator").Return([]models.Task{}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/user/tasks", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})
}

func TestHandler_RespondToApplication(t *testing.T) {
	router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.POST("/tasks/:id/applications/:applicationId/respond", handler.RespondToApplication)

	t.Run("successful respond - accept", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("GetByID", mock.Anything, uint(10)).Return(&models.Application{ID: 10, TaskID: 1}, nil)
		mockTaskRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)
		mockAppRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Application")).Return(nil)

		reqBody := respondToApplicationRequest{Status: "accepted"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/applications/10/respond", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("successful respond - decline", func(t *testing.T) {
		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("GetByID", mock.Anything, uint(10)).Return(&models.Application{ID: 10, TaskID: 1}, nil)
		mockAppRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Application")).Return(nil)

		reqBody := respondToApplicationRequest{Status: "declined"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/applications/10/respond", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}

func TestHandler_UpdateProfile(t *testing.T) {
	router, handler, mockUserRepo, _, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.PUT("/user/profile", handler.UpdateProfile)

	t.Run("successful update profile", func(t *testing.T) {
		user := &models.User{ID: 1, FullName: "Old Name", Email: "old@example.com"}
		mockUserRepo.On("GetByID", mock.Anything, uint(1)).Return(user, nil)
		mockUserRepo.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, gorm.ErrRecordNotFound)
		mockUserRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

		reqBody := updateProfileRequest{FullName: "New Name", Email: "new@example.com"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestHandler_GetAssignedTasks(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.GET("/user/assigned-tasks", handler.GetAssignedTasks)

	t.Run("successful get assigned tasks", func(t *testing.T) {
		mockTaskRepo.On("ListByUser", mock.Anything, uint(1), "assigned").Return([]models.Task{}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/user/assigned-tasks", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("exclude self assigned", func(t *testing.T) {
		tasks := []models.Task{
			{ID: 1, CreatedBy: 1}, // Self-assigned
			{ID: 2, CreatedBy: 2}, // Assigned from others
		}
		mockTaskRepo.On("ListByUser", mock.Anything, uint(1), "assigned").Return(tasks, nil)

		req, _ := http.NewRequest(http.MethodGet, "/user/assigned-tasks?exclude_self_assigned=true", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var result []models.Task
		json.Unmarshal(resp.Body.Bytes(), &result)
		assert.Equal(t, 1, len(result))
		assert.Equal(t, uint(2), result[0].ID)
	})
}

func TestHandler_GetUserByID(t *testing.T) {
	router, handler, mockUserRepo, _, _, _ := setupTestRouter()
	router.GET("/users/:id", handler.GetUserByID)

	t.Run("successful get user", func(t *testing.T) {
		user := &models.User{ID: 1, Username: "testuser"}
		mockUserRepo.On("GetByID", mock.Anything, uint(1)).Return(user, nil)

		req, _ := http.NewRequest(http.MethodGet, "/users/1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("user not found", func(t *testing.T) {
		mockUserRepo.On("GetByID", mock.Anything, uint(99)).Return(nil, services.ErrUserNotFound)

		req, _ := http.NewRequest(http.MethodGet, "/users/99", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusNotFound, resp.Code)
	})
}

func TestHandler_GetUserReviews(t *testing.T) {
	router, handler, _, _, mockReviewRepo, _ := setupTestRouter()
	router.GET("/users/:id/reviews", handler.GetUserReviews)

	t.Run("successful get reviews", func(t *testing.T) {
		mockReviewRepo.On("ListByUser", mock.Anything, uint(1)).Return([]models.Review{}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/users/1/reviews", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})
}
