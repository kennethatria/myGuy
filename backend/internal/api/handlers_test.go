package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/services"
	"myguy/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func setupTestRouter() (*gin.Engine, *Handler, *tests.MockUserRepository, *tests.MockTaskRepository, *tests.MockReviewRepository, *tests.MockApplicationRepository) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	mockUserRepo := new(tests.MockUserRepository)
	mockTaskRepo := new(tests.MockTaskRepository)
	mockReviewRepo := new(tests.MockReviewRepository)
	mockAppRepo := new(tests.MockApplicationRepository)

	userService := services.NewUserService(mockUserRepo)
	taskService := services.NewTaskService(mockTaskRepo, mockAppRepo, nil)
	reviewService := services.NewReviewService(mockReviewRepo, mockTaskRepo, mockUserRepo)
	authMiddleware := middleware.NewJWTAuthMiddleware("test-secret")

	authService := services.NewAuthService(mockUserRepo, nil, nil, "test-secret")

	handler := NewHandler(authService, userService, taskService, reviewService, authMiddleware)

	return router, handler, mockUserRepo, mockTaskRepo, mockReviewRepo, mockAppRepo
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
		reqBody := createTaskRequest{Title: "Paint my fence", Description: "White paint provided"}

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

	t.Run("contact details are refused with a reason", func(t *testing.T) {
		body, _ := json.Marshal(createTaskRequest{Title: "Paint fence", Description: "call 0772 123 456"})
		req, _ := http.NewRequest(http.MethodPost, "/tasks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "share them in chat")
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
	t.Run("successful update", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.PUT("/tasks/:id", handler.UpdateTask)

		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockTaskRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

		reqBody := createTaskRequest{Title: "New Title", Description: "New Desc"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.PUT("/tasks/:id", handler.UpdateTask)

		task := &models.Task{ID: 1, CreatedBy: 2}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)

		reqBody := createTaskRequest{Title: "New Title", Description: "New Desc"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPut, "/tasks/1", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusForbidden, resp.Code)
	})
}

func TestHandler_DeleteTask(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.DELETE("/tasks/:id", handler.DeleteTask)

		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("ListByTask", mock.Anything, uint(1)).Return([]models.Application{}, nil)
		mockTaskRepo.On("Delete", mock.Anything, uint(1)).Return(nil)

		req, _ := http.NewRequest(http.MethodDelete, "/tasks/1", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.DELETE("/tasks/:id", handler.DeleteTask)

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
		assignee := uint(2)
		task := &models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignee, Status: "in_progress"}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockTaskRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

		reqBody := UpdateTaskStatusRequest{Status: "completed"}
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

// Browsing shows only gigs within their 24 hours; your own list shows all
func TestHandler_ListTasksBoardIsLiveOnly(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(1))
		c.Next()
	})
	router.GET("/tasks", handler.ListTasks)
	live := func(want bool) interface{} {
		return mock.MatchedBy(func(f map[string]interface{}) bool {
			at, ok := f["deadline_after"].(time.Time)
			return ok == want && (!ok || time.Since(at) < time.Minute)
		})
	}
	mockTaskRepo.On("ListWithPagination", mock.Anything, live(true)).Return([]models.Task{}, nil).Once()
	mockTaskRepo.On("Count", mock.Anything, live(true)).Return(int64(0), nil).Once()
	mockTaskRepo.On("ListWithPagination", mock.Anything, live(false)).Return([]models.Task{}, nil).Once()
	mockTaskRepo.On("Count", mock.Anything, live(false)).Return(int64(0), nil).Once()

	for _, path := range []string{"/tasks?status=open&exclude_created_by=1", "/tasks?created=true"} {
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(resp, req)
		assert.Equal(t, http.StatusOK, resp.Code, path)
	}
	mockTaskRepo.AssertExpectations(t)
}

func TestHandler_ApplyForTask(t *testing.T) {
	router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(2))
		c.Next()
	})
	router.POST("/tasks/:id/apply", handler.ApplyForTask)

	t.Run("successful application", func(t *testing.T) {
		task := &models.Task{ID: 1, Status: "open", CreatedBy: 1, Deadline: time.Now().Add(time.Hour)}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("ListByTask", mock.Anything, uint(1)).Return([]models.Application{}, nil)
		mockAppRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Application")).Return(nil)

		reqBody := applyForTaskRequest{Message: "I'm interested"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/apply", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
		mockAppRepo.AssertExpectations(t)
	})

	t.Run("applying without a message", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/apply", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusCreated, resp.Code)
	})
}

func TestHandler_CreateReview(t *testing.T) {
	router, handler, mockUserRepo, mockTaskRepo, mockReviewRepo, _ := setupTestRouter()
	mockUserRepo.On("UpdateRating", mock.Anything, uint(2)).Return(nil)
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
	t.Run("successful respond - accept", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.POST("/tasks/:id/applications/:applicationId/respond", handler.RespondToApplication)

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("GetByID", mock.Anything, uint(10)).Return(&models.Application{ID: 10, TaskID: 1, ApplicantID: 2, Status: "pending"}, nil)
		mockAppRepo.On("ListByTask", mock.Anything, uint(1)).Return([]models.Application{}, nil)
		mockTaskRepo.On("AssignIfOpen", mock.Anything, uint(1), uint(2)).Return(true, nil)
		mockAppRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.Application")).Return(nil)
		mockAppRepo.On("DeclinePending", mock.Anything, uint(1), uint(10)).Return(nil)

		reqBody := respondToApplicationRequest{Status: "accepted"}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest(http.MethodPost, "/tasks/1/applications/10/respond", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("successful respond - decline", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.POST("/tasks/:id/applications/:applicationId/respond", handler.RespondToApplication)

		task := &models.Task{ID: 1, CreatedBy: 1}
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)
		mockAppRepo.On("GetByID", mock.Anything, uint(10)).Return(&models.Application{ID: 10, TaskID: 1, ApplicantID: 2, Status: "pending"}, nil)
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

	t.Run("updates name and can clear the bio; email is never changed", func(t *testing.T) {
		mockUserRepo.On("GetByID", mock.Anything, uint(1)).Return(&models.User{ID: 1, Email: "old@example.com", Bio: "old bio"}, nil).Once()
		mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(u *models.User) bool {
			return u.FullName == "New Name" && u.Bio == "" && u.Email == "old@example.com"
		})).Return(nil).Once()

		// An email in the body is ignored rather than applied unverified.
		body := []byte(`{"full_name":"  New Name ","bio":"","email":"attacker@example.com"}`)
		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("full name is required", func(t *testing.T) {
		body := []byte(`{"full_name":"   ","bio":"hi"}`)
		req, _ := http.NewRequest(http.MethodPut, "/user/profile", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
	})
}

func TestHandler_GetAssignedTasks(t *testing.T) {
	t.Run("successful get assigned tasks", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.GET("/user/assigned-tasks", handler.GetAssignedTasks)

		mockTaskRepo.On("ListByUser", mock.Anything, uint(1), "assigned").Return([]models.Task{}, nil)

		req, _ := http.NewRequest(http.MethodGet, "/user/assigned-tasks", nil)
		resp := httptest.NewRecorder()

		router.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		mockTaskRepo.AssertExpectations(t)
	})

	t.Run("exclude self assigned", func(t *testing.T) {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) {
			c.Set("userID", uint(1))
			c.Next()
		})
		router.GET("/user/assigned-tasks", handler.GetAssignedTasks)

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
		mockUserRepo.On("GetByID", mock.Anything, uint(99)).Return(nil, gorm.ErrRecordNotFound)

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

func TestHandler_GetMyTaskReview(t *testing.T) {
	check := func(found *models.Review, err error) map[string]interface{} {
		router, handler, _, _, mockReviewRepo, _ := setupTestRouter()
		router.Use(func(c *gin.Context) { c.Set("userID", uint(3)); c.Next() })
		router.GET("/tasks/:id/reviews/mine", handler.GetMyTaskReview)
		mockReviewRepo.On("GetTaskReview", mock.Anything, uint(1), uint(3)).Return(found, err)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/tasks/1/reviews/mine", nil))
		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		return body
	}

	assert.Equal(t, true, check(&models.Review{ID: 8, TaskID: 1, ReviewerID: 3, Rating: 5}, nil)["reviewed"])
	assert.Equal(t, false, check(nil, gorm.ErrRecordNotFound)["reviewed"])
}


func TestParseID(t *testing.T) {
	id, err := parseID("42")
	assert.NoError(t, err)
	assert.Equal(t, uint(42), id)

	for _, bad := range []string{"", "-1", "abc", "1.5", "4294967296"} {
		_, err := parseID(bad)
		assert.Error(t, err, bad)
	}
}
