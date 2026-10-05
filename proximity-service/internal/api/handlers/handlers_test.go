package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"proximity-service/internal/middleware"
	"proximity-service/internal/repositories"
	"proximity-service/internal/services"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

const key = "test-key"

func setup(t *testing.T) (*gin.Engine, *miniredis.Miniredis) {
	gin.SetMode(gin.TestMode)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	r := gin.New()
	NewHandler(services.NewProximityService(repositories.NewRedisLocationRepository(rdb))).Register(r, middleware.InternalKey(key))
	return r, mr
}

func call(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", key)
	r.ServeHTTP(w, req)
	return w
}

func TestSaveDistancesDelete(t *testing.T) {
	r, _ := setup(t)

	assert.Equal(t, http.StatusNoContent, call(r, "PUT", "/internal/locations/task/41", `{"lat":0.3476,"lng":32.5842}`).Code)
	assert.Equal(t, http.StatusNoContent, call(r, "PUT", "/internal/locations/task/57", `{"lat":0.3476,"lng":32.63}`).Code)

	w := call(r, "POST", "/internal/distances/task", `{"lat":0.3476,"lng":32.5842,"ids":[41,42,57]}`)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"results":[{"id":41,"bucket":0},{"id":57,"bucket":2}]}`, w.Body.String())

	assert.Equal(t, http.StatusNoContent, call(r, "DELETE", "/internal/locations/task/41", "").Code)
	w = call(r, "POST", "/internal/distances/task", `{"from":{"kind":"task","id":57},"ids":[41,57]}`)
	assert.JSONEq(t, `{"results":[{"id":57,"bucket":0}]}`, w.Body.String())
}

func TestBadRequests(t *testing.T) {
	r, _ := setup(t)
	cases := []struct{ method, path, body string }{
		{"PUT", "/internal/locations/user/1", `{"lat":0,"lng":0}`},
		{"PUT", "/internal/locations/task/0", `{"lat":0,"lng":0}`},
		{"PUT", "/internal/locations/task/x", `{"lat":0,"lng":0}`},
		{"PUT", "/internal/locations/task/1", `{"lat":0}`},
		{"PUT", "/internal/locations/task/1", `not json`},
		{"PUT", "/internal/locations/task/1", `{"lat":95,"lng":0}`},
		{"DELETE", "/internal/locations/task/x", ``},
		{"DELETE", "/internal/locations/user/1", ``},
		{"POST", "/internal/distances/task", `not json`},
		{"POST", "/internal/distances/task", `{"ids":[1]}`},
		{"POST", "/internal/distances/user", `{"lat":0,"lng":0}`},
	}
	for _, tc := range cases {
		assert.Equal(t, http.StatusBadRequest, call(r, tc.method, tc.path, tc.body).Code, "%s %s %s", tc.method, tc.path, tc.body)
	}
}

func TestKeyRequired(t *testing.T) {
	r, _ := setup(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/internal/distances/task", bytes.NewBufferString(`{"lat":0,"lng":0}`))
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHealthAndRedisDown(t *testing.T) {
	r, mr := setup(t)
	assert.Equal(t, http.StatusOK, call(r, "GET", "/health", "").Code)
	mr.Close()

	assert.Equal(t, http.StatusServiceUnavailable, call(r, "GET", "/health", "").Code)
	assert.Equal(t, http.StatusServiceUnavailable, call(r, "PUT", "/internal/locations/task/1", `{"lat":0,"lng":0}`).Code)
	assert.Equal(t, http.StatusServiceUnavailable, call(r, "DELETE", "/internal/locations/task/1", ``).Code)
	assert.Equal(t, http.StatusServiceUnavailable, call(r, "POST", "/internal/distances/task", `{"lat":0,"lng":0,"ids":[1]}`).Code)
}
