package api

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"myguy/internal/services"
)

func TestPausingCodes(t *testing.T) {
	app := setupBlockingApp(t)
	_, janeToken := app.user(t, "jane")

	w, out := app.call(http.MethodPost, "/internal/v1/code-pauses", testInternalKey, "",
		gin.H{"email": "Jane@Example.com", "hours": 24, "by": "telegram"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, out["paused"])
	assert.Equal(t, "jane@example.com", out["email"])

	// Asking for a code looks the same, but nothing is sent
	w, _ = app.call(http.MethodPost, "/auth/request-code", "", "", gin.H{"email": "jane@example.com"})
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Empty(t, app.sender.codes)
	// Her session is untouched
	w, _ = app.call(http.MethodGet, "/api/v1/tasks", "", janeToken, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	_, out = app.call(http.MethodGet, "/internal/v1/code-pauses/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, true, out["paused"])
	assert.NotEmpty(t, out["until"])

	// Unpaused: codes go again
	w, _ = app.call(http.MethodDelete, "/internal/v1/code-pauses/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	app.call(http.MethodPost, "/auth/request-code", "", "", gin.H{"email": "jane@example.com"})
	assert.NotEmpty(t, app.sender.codes["jane@example.com"])
	w, _ = app.call(http.MethodDelete, "/internal/v1/code-pauses/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	_, out = app.call(http.MethodGet, "/internal/v1/code-pauses/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, false, out["paused"])
}

func TestPausingCodesRefusesBadInput(t *testing.T) {
	app := setupBlockingApp(t)
	for _, body := range []gin.H{{"email": "a@example.com", "hours": 0}, {"email": "a@example.com", "hours": 73}} {
		w, out := app.call(http.MethodPost, "/internal/v1/code-pauses", testInternalKey, "", body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, services.ErrInvalidPause.Error(), out["error"])
	}
	w, out := app.call(http.MethodPost, "/internal/v1/code-pauses", testInternalKey, "", gin.H{"email": "nope", "hours": 2})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, services.ErrInvalidEmail.Error(), out["error"])
}
