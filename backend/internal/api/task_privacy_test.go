package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"myguy/internal/models"
)

func TestHandler_GetTask_LimitsWhatEachViewerSees(t *testing.T) {
	task := &models.Task{
		ID: 1, CreatedBy: 1, Status: "open",
		Creator: models.User{ID: 1, Email: "owner@example.com", PhoneNumber: "+256700000001"},
		Applications: []models.Application{
			{ID: 10, ApplicantID: 2, ProposedFee: 80, Message: "pick me", Applicant: models.User{ID: 2, Email: "a2@example.com", PhoneNumber: "+256700000002"}},
			{ID: 11, ApplicantID: 3, ProposedFee: 90, Message: "or me", Applicant: models.User{ID: 3, Email: "a3@example.com"}},
		},
	}

	get := func(viewer uint) models.Task {
		router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) { c.Set("userID", viewer); c.Next() })
		router.GET("/tasks/:id", handler.GetTask)
		mockTaskRepo.On("GetByID", mock.Anything, uint(1)).Return(task, nil)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/tasks/1", nil))
		assert.Equal(t, http.StatusOK, resp.Code)

		var got models.Task
		assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &got))
		return got
	}

	t.Run("stranger sees no applications and no contact details", func(t *testing.T) {
		got := get(99)
		assert.Empty(t, got.Applications)
		assert.Empty(t, got.Creator.Email)
		assert.Empty(t, got.Creator.PhoneNumber)
	})

	t.Run("applicant sees only their own application", func(t *testing.T) {
		got := get(2)
		assert.Len(t, got.Applications, 1)
		assert.Equal(t, uint(10), got.Applications[0].ID)
		assert.Equal(t, "a2@example.com", got.Applications[0].Applicant.Email, "own details stay visible")
		assert.Empty(t, got.Creator.Email)
	})

	t.Run("owner sees every application without applicants' contact details", func(t *testing.T) {
		got := get(1)
		assert.Len(t, got.Applications, 2)
		for _, app := range got.Applications {
			assert.Empty(t, app.Applicant.Email)
			assert.Empty(t, app.Applicant.PhoneNumber)
		}
		assert.Equal(t, "owner@example.com", got.Creator.Email)
	})

	// The handler must not have mutated the shared task.
	assert.Equal(t, "a2@example.com", task.Applications[0].Applicant.Email)
}

func TestHandler_UserContactDetailsAreOnlyShownToThatUser(t *testing.T) {
	user := &models.User{ID: 5, Username: "sam", Email: "sam@example.com", PhoneNumber: "+256700000005"}

	getUser := func(viewer uint) map[string]interface{} {
		router, handler, mockUserRepo, _, _, _ := setupTestRouter()
		router.Use(func(c *gin.Context) { c.Set("userID", viewer); c.Next() })
		router.GET("/users/:id", handler.GetUserByID)
		mockUserRepo.On("GetByID", mock.Anything, uint(5)).Return(user, nil)

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/users/5", nil))
		assert.Equal(t, http.StatusOK, resp.Code)
		var body map[string]interface{}
		assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
		return body
	}

	other := getUser(9)
	assert.Equal(t, "sam", other["username"])
	assert.Empty(t, other["email"])
	assert.Empty(t, other["phone_number"])

	self := getUser(5)
	assert.Equal(t, "sam@example.com", self["email"])
}

func TestHandler_GetUserReviews_HidesContactDetails(t *testing.T) {
	router, handler, _, _, mockReviewRepo, _ := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(9)); c.Next() })
	router.GET("/users/:id/reviews", handler.GetUserReviews)
	mockReviewRepo.On("ListByUser", mock.Anything, uint(5)).Return([]models.Review{{
		ID: 1, ReviewerID: 6, ReviewedUserID: 5, Rating: 5,
		Reviewer:     models.User{ID: 6, Username: "ann", Email: "ann@example.com", PhoneNumber: "+256700000006"},
		ReviewedUser: models.User{ID: 5, Username: "sam", Email: "sam@example.com"},
		Task:         models.Task{ID: 3, CreatedBy: 6, Creator: models.User{ID: 6, Email: "ann@example.com"}},
	}}, nil)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/users/5/reviews", nil))

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotContains(t, resp.Body.String(), "@example.com")
	assert.NotContains(t, resp.Body.String(), "+2567")
	assert.Contains(t, resp.Body.String(), `"username":"ann"`)
}

func TestHandler_GetMyReviews(t *testing.T) {
	router, handler, _, _, mockReviewRepo, _ := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(5)); c.Next() })
	router.GET("/user/reviews", handler.GetMyReviews)
	// Always the signed-in user's own reviews, and the other side's contact
	// details stay hidden.
	mockReviewRepo.On("ListInvolving", mock.Anything, uint(5)).Return([]models.Review{{
		ID: 1, ReviewerID: 6, ReviewedUserID: 5, Rating: 4,
		Reviewer:     models.User{ID: 6, Username: "ann", Email: "ann@example.com", PhoneNumber: "+256700000006"},
		ReviewedUser: models.User{ID: 5, Username: "sam"},
	}}, nil)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/user/reviews", nil))

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.NotContains(t, resp.Body.String(), "ann@example.com")
	assert.NotContains(t, resp.Body.String(), "+2567")
	assert.Contains(t, resp.Body.String(), `"username":"ann"`)
	mockReviewRepo.AssertExpectations(t)
}

func TestHandler_GetMyReviews_Error(t *testing.T) {
	router, handler, _, _, mockReviewRepo, _ := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(5)); c.Next() })
	router.GET("/user/reviews", handler.GetMyReviews)
	mockReviewRepo.On("ListInvolving", mock.Anything, uint(5)).Return([]models.Review{}, errors.New("db down"))

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/user/reviews", nil))

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	assert.NotContains(t, resp.Body.String(), "db down")
}

func TestHandler_GetApplicationParticipants(t *testing.T) {
	app := &models.Application{ID: 10, TaskID: 1, ApplicantID: 2, Task: models.Task{ID: 1, CreatedBy: 1}}

	get := func(viewer uint, appID string) *httptest.ResponseRecorder {
		router, handler, _, _, _, mockAppRepo := setupTestRouter()
		router.Use(func(c *gin.Context) { c.Set("userID", viewer); c.Next() })
		router.GET("/applications/:id/participants", handler.GetApplicationParticipants)
		mockAppRepo.On("GetByID", mock.Anything, uint(10)).Return(app, nil)
		mockAppRepo.On("GetByID", mock.Anything, uint(11)).Return(nil, errors.New("not found"))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/applications/"+appID+"/participants", nil))
		return resp
	}

	for _, participant := range []uint{1, 2} {
		resp := get(participant, "10")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"application_id":10,"task_id":1,"applicant_id":2,"task_owner_id":1}`, resp.Body.String())
	}

	// An outsider and a missing id look the same, so ids can't be probed.
	assert.Equal(t, http.StatusNotFound, get(99, "10").Code)
	assert.Equal(t, http.StatusNotFound, get(1, "11").Code)
}

func TestHandler_GetUserApplications(t *testing.T) {
	router, handler, _, _, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(2)); c.Next() })
	router.GET("/user/applications", handler.GetUserApplications)
	mockAppRepo.On("ListByUser", mock.Anything, uint(2)).Return([]models.Application{{
		ID: 10, TaskID: 1, ApplicantID: 2, ProposedFee: 80, Status: "pending",
		Task: models.Task{ID: 1, Title: "Paint fence", Status: "open", CreatedBy: 1,
			Creator: models.User{ID: 1, Username: "ann", Email: "ann@example.com", PhoneNumber: "+256700000001"}},
	}}, nil)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/user/applications", nil))

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), `"title":"Paint fence"`)
	assert.Contains(t, resp.Body.String(), `"username":"ann"`)
	assert.NotContains(t, resp.Body.String(), "ann@example.com", "the poster's contact details stay hidden")
	assert.NotContains(t, resp.Body.String(), "+2567")
}

