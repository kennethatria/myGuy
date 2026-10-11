package ratelimit

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func clocked(rules map[string][]Rule) (*Limiter, *time.Time) {
	l := New(rules)
	clock := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	l.now = func() time.Time { return clock }
	return l, &clock
}

func TestAllowCountsPerAccountAndAction(t *testing.T) {
	l, clock := clocked(map[string][]Rule{"post": {{Limit: 2, Per: time.Hour}}})
	assert.True(t, ok(l.Allow(1, "post")))
	*clock = clock.Add(10 * time.Minute)
	assert.True(t, ok(l.Allow(1, "post")))

	allowed, wait := l.Allow(1, "post")
	assert.False(t, allowed)
	assert.Equal(t, 50*time.Minute, wait, "free again when the first leaves the hour")

	assert.True(t, ok(l.Allow(2, "post")), "another account has its own count")
	assert.True(t, ok(l.Allow(1, "apply")), "an action without rules isn't limited")

	// Refusals don't count: an hour after the first, one is free again
	*clock = clock.Add(50 * time.Minute)
	assert.True(t, ok(l.Allow(1, "post")))
	assert.False(t, ok(l.Allow(1, "post")))
}

func TestEveryRuleMustAllow(t *testing.T) {
	l, clock := clocked(map[string][]Rule{"message": {{Limit: 2, Per: time.Minute}, {Limit: 3, Per: time.Hour}}})
	assert.True(t, ok(l.Allow(1, "message")))
	assert.True(t, ok(l.Allow(1, "message")))
	_, wait := l.Allow(1, "message")
	assert.Equal(t, time.Minute, wait)

	*clock = clock.Add(time.Minute)
	assert.True(t, ok(l.Allow(1, "message")))
	_, wait = l.Allow(1, "message")
	assert.Equal(t, 59*time.Minute, wait, "the hourly rule now")
}

func TestSweepForgetsIdleAccounts(t *testing.T) {
	l, clock := clocked(map[string][]Rule{"post": {{Limit: 5, Per: time.Hour}}})
	l.Allow(1, "post")
	*clock = clock.Add(2 * time.Hour)
	for i := 0; i < 1000; i++ {
		l.Allow(2, "post")
	}
	_, kept := l.hits[key{1, "post"}]
	assert.False(t, kept)
}

func TestMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	l, _ := clocked(map[string][]Rule{"post": {{Limit: 1, Per: time.Hour}}})
	r := gin.New()
	r.POST("/tasks", func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Set("username", "jane")
		c.Set("email", "jane@example.com")
	}, l.Middleware("post"), func(c *gin.Context) { c.Status(http.StatusCreated) })

	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	call := func() *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/tasks", nil))
		return w
	}

	assert.Equal(t, http.StatusCreated, call().Code)
	w := call()
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "3600", w.Header().Get("Retry-After"))
	var body map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, Code, body["code"])
	assert.Equal(t, float64(3600), body["retry_after"])
	assert.Equal(t, "You're doing that a lot. Try again in 60 minutes.", body["error"])
	call()

	// One line per run of refusals, with who it was
	assert.Equal(t, 1, strings.Count(logs.String(), "limit event=hit"))
	assert.Contains(t, logs.String(), `limit event=hit action=post user=7 username="jane" email="jane@example.com"`)
}

func TestInWords(t *testing.T) {
	assert.Equal(t, "a minute", inWords(20*time.Second))
	assert.Equal(t, "1 minute", inWords(time.Minute))
	assert.Equal(t, "2 minutes", inWords(61*time.Second))
}

func ok(allowed bool, _ time.Duration) bool { return allowed }
