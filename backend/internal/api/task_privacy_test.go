package api

import (
	"encoding/json"
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
