package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/services"
)

func TestSuspendingAnAccount(t *testing.T) {
	app := setupBlockingApp(t)
	jane, janeToken := app.user(t, "jane")
	_, bobToken := app.user(t, "bob")
	assert.NoError(t, app.db.Create(&models.Task{Title: "Jane's gig", CreatedBy: jane.ID, Status: "open",
		Deadline: time.Now().Add(time.Hour)}).Error)
	board := func() int {
		_, out := app.call(http.MethodGet, "/api/v1/tasks", "", bobToken, nil)
		tasks, _ := out["tasks"].([]interface{})
		return len(tasks)
	}

	w, out := app.call(http.MethodPost, "/internal/v1/suspensions", testInternalKey, "",
		gin.H{"email": "Jane@Example.com", "reason": "spam", "by": "telegram"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, out["suspended"])
	assert.Equal(t, float64(12), out["hours"], "the first is 12 hours")
	assert.Equal(t, float64(1), out["step"])

	_, out = app.call(http.MethodGet, "/internal/v1/blocked-users", testInternalKey, "", nil)
	assert.Empty(t, out["user_ids"])
	suspended := out["suspended"].([]interface{})
	assert.Len(t, suspended, 1)
	assert.Equal(t, float64(jane.ID), suspended[0].(map[string]interface{})["user_id"])

	// Read-only: she can look, not post
	w, _ = app.call(http.MethodGet, "/api/v1/tasks", "", janeToken, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w, out = app.call(http.MethodPost, "/api/v1/tasks", "", janeToken, gin.H{"title": "Another", "description": "x"})
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, middleware.AccountSuspended, out["code"])
	assert.NotEmpty(t, out["until"])
	assert.Contains(t, out["error"], "suspended until")
	// Her gig is off the board meanwhile
	assert.Equal(t, 0, board())

	// Everyone sees the mark on her profile, and so does she
	_, out = app.call(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", jane.ID), "", bobToken, nil)
	assert.NotEmpty(t, out["suspension"].(map[string]interface{})["until"])
	assert.Empty(t, out["email"], "still no contact details")
	_, out = app.call(http.MethodGet, "/api/v1/profile", "", janeToken, nil)
	assert.NotEmpty(t, out["suspension"].(map[string]interface{})["until"])

	w, out = app.call(http.MethodPost, "/internal/v1/suspensions", testInternalKey, "", gin.H{"email": "jane@example.com"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, services.ErrAlreadySuspended.Error(), out["error"])

	_, out = app.call(http.MethodGet, "/internal/v1/suspensions/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, true, out["suspended"])
	assert.Equal(t, "spam", out["reason"])
	assert.Equal(t, float64(1), out["in_row"])
	assert.Equal(t, float64(24), out["next_hours"])

	// Lifted: all back, and the profile says it happened
	w, _ = app.call(http.MethodDelete, "/internal/v1/suspensions/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, board())
	w, _ = app.call(http.MethodPost, "/api/v1/tasks", "", janeToken, gin.H{"title": "Another", "description": "x"})
	assert.NotEqual(t, http.StatusForbidden, w.Code)
	_, out = app.call(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", jane.ID), "", bobToken, nil)
	mark := out["suspension"].(map[string]interface{})
	assert.Nil(t, mark["until"])
	assert.NotEmpty(t, mark["ended"])
	w, _ = app.call(http.MethodDelete, "/internal/v1/suspensions/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// The next one is longer, unless a length is given
	_, out = app.call(http.MethodPost, "/internal/v1/suspensions", testInternalKey, "", gin.H{"email": "jane@example.com"})
	assert.Equal(t, float64(24), out["hours"])
}

func TestSuspensionRefusesBadInput(t *testing.T) {
	app := setupBlockingApp(t)
	app.user(t, "jane")
	for _, body := range []gin.H{
		{"email": "nobody@example.com"},
		{"email": "jane@example.com", "hours": 5},
		{"email": "nope"},
	} {
		w, out := app.call(http.MethodPost, "/internal/v1/suspensions", testInternalKey, "", body)
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
		assert.NotContains(t, out["error"], "something went wrong", body)
	}
	app.call(http.MethodPost, "/internal/v1/email-blocks", testInternalKey, "", gin.H{"email": "jane@example.com"})
	_, out := app.call(http.MethodPost, "/internal/v1/suspensions", testInternalKey, "", gin.H{"email": "jane@example.com"})
	assert.Equal(t, services.ErrSuspendBlocked.Error(), out["error"])
}
