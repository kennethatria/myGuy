package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"myguy/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRespondError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	respond := func(err error) *httptest.ResponseRecorder {
		r := gin.New()
		r.GET("/x", func(c *gin.Context) { respondError(c, http.StatusBadRequest, err) })
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/x", nil)
		r.ServeHTTP(w, req)
		return w
	}

	w := respond(services.ErrHeadlineTooLong)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "headline can be at most 5 words")

	w = respond(errors.New(`ERROR: relation "applications" does not exist (SQLSTATE 42P01)`))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NotContains(t, w.Body.String(), "SQLSTATE")
	assert.NotContains(t, w.Body.String(), "applications")
	assert.Contains(t, w.Body.String(), "something went wrong")
}

func TestIsUserFacing(t *testing.T) {
	assert.True(t, services.IsUserFacing(services.ErrContactDetails))
	assert.True(t, services.IsUserFacing(errors.Join(errors.New("ctx"), services.ErrTaskNotFound)))
	assert.False(t, services.IsUserFacing(errors.New("dial tcp: connection refused")))
}
