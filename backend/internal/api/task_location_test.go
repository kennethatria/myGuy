package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"myguy/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandler_TaskLocation(t *testing.T) {
	router, handler, _, mockTaskRepo, _, mockAppRepo := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(1)); c.Next() })
	router.POST("/tasks", handler.CreateTask)
	router.PATCH("/tasks/:id/status", handler.UpdateTaskStatus)

	post := func(method, path, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	mockTaskRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Task")).Return(nil)

	assert.Equal(t, http.StatusCreated, post("POST", "/tasks",
		`{"title":"Paint my fence","description":"White paint","lat":0.35,"lng":32.585}`).Code, "a location is optional and accepted")

	for _, body := range []string{
		`{"title":"Paint my fence","description":"White paint","lat":0.35}`,
		`{"title":"Paint my fence","description":"White paint","lat":95,"lng":32.585}`,
	} {
		resp := post("POST", "/tasks", body)
		assert.Equal(t, http.StatusBadRequest, resp.Code, body)
		assert.Contains(t, resp.Body.String(), "location must have a valid lat and lng")
	}

	// Reposting an expired gig can bring a fresh location
	mockTaskRepo.On("GetByID", mock.Anything, uint(5)).Return(&models.Task{ID: 5, CreatedBy: 1, Status: "expired"}, nil)
	mockTaskRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	mockAppRepo.On("ListByTask", mock.Anything, uint(5)).Return([]models.Application{}, nil).Maybe()
	assert.Equal(t, http.StatusOK, post("PATCH", "/tasks/5/status", `{"status":"open","lat":0.35,"lng":32.585}`).Code)
	assert.Equal(t, http.StatusBadRequest, post("PATCH", "/tasks/5/status", `{"status":"open","lng":32.585}`).Code)
}

func TestParseNear(t *testing.T) {
	none, err := parseNear("")
	assert.NoError(t, err)
	assert.Nil(t, none)

	got, err := parseNear("0.3476, 32.5842")
	assert.NoError(t, err)
	assert.InDelta(t, 0.35, got.Lat, 1e-9)

	for _, bad := range []string{"0.35", "a,b", "0.35,32,1", "95,0"} {
		_, err := parseNear(bad)
		assert.Error(t, err, bad)
	}
}

func TestHandler_ListTasksNear(t *testing.T) {
	router, handler, _, mockTaskRepo, _, _ := setupTestRouter()
	router.Use(func(c *gin.Context) { c.Set("userID", uint(1)); c.Next() })
	router.GET("/tasks", handler.ListTasks)

	get := func(q string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodGet, "/tasks?"+q, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	assert.Equal(t, http.StatusBadRequest, get("near=nope").Code)

	// No distancer configured in tests: distance sort falls back to newest first
	mockTaskRepo.On("Count", mock.Anything, mock.Anything).Return(int64(1), nil)
	mockTaskRepo.On("ListWithPagination", mock.Anything, mock.Anything).Return([]models.Task{{ID: 3, Title: "Gig"}}, nil)
	resp := get("near=0.35,32.585&sort_by=distance&status=open")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), `"id":3`)
	assert.Equal(t, http.StatusOK, get("near=0.35,32.585").Code, "other sorts are tagged when possible")
}
