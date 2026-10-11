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
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"myguy/internal/services"
)

const testInternalKey = "internal-test-key"

type blockingApp struct {
	router *gin.Engine
	db     *gorm.DB
	sender *capturingSender
	jwt    *middleware.JWTAuthMiddleware
}

// setupBlockingApp wires sign-in, the session check, the gig board and the
// internal routes over in-memory SQLite, as main.go does.
func setupBlockingApp(t *testing.T) *blockingApp {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Task{}, &models.Application{}, &models.LoginCode{}, &models.BlockedEmail{}); err != nil {
		t.Fatal(err)
	}
	userRepo := repositories.NewGormUserRepository(db)
	blocks := services.NewBlockService(repositories.NewGormBlockedEmailRepository(db))
	sender := &capturingSender{codes: map[string]string{}}
	jwt := middleware.NewJWTAuthMiddleware("test-secret").WithBlocked(blocks.UserBlocked)
	auth := services.NewAuthService(userRepo, repositories.NewGormLoginCodeRepository(db), sender, "test-secret").WithBlocks(blocks)
	tasks := services.NewTaskService(repositories.NewGormTaskRepository(db), repositories.NewGormApplicationRepository(db), nil)
	handler := NewHandler(auth, services.NewUserService(userRepo), tasks, nil, jwt)

	router := gin.New()
	router.POST("/auth/request-code", handler.RequestLoginCode)
	router.POST("/auth/verify-code", handler.VerifyLoginCode)
	protected := router.Group("/api/v1", jwt.AuthRequired())
	protected.GET("/tasks", handler.ListTasks)
	NewInternalHandler(blocks).Register(router, testInternalKey)
	return &blockingApp{router: router, db: db, sender: sender, jwt: jwt}
}

func (a *blockingApp) call(method, path, key, token string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("X-Internal-API-Key", key)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, req)
	var out map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func (a *blockingApp) user(t *testing.T, name string) (*models.User, string) {
	t.Helper()
	u := &models.User{Username: name, Email: name + "@example.com", FullName: name}
	if err := a.db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	token, _ := a.jwt.GenerateToken(u.ID, u.Username, u.Email, u.FullName)
	return u, token
}

func TestInternalRoutesNeedTheKey(t *testing.T) {
	app := setupBlockingApp(t)
	for _, key := range []string{"", "wrong"} {
		w, _ := app.call(http.MethodGet, "/internal/v1/blocked-users", key, "", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}
	// No key configured: nothing gets in, not even an empty key
	router := gin.New()
	NewInternalHandler(nil).Register(router, "")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/internal/v1/blocked-users", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestBlockingAnEmail(t *testing.T) {
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
	assert.Equal(t, 1, board())

	w, out := app.call(http.MethodPost, "/internal/v1/email-blocks", testInternalKey, "",
		gin.H{"email": "Jane@Example.com", "reason": "spam", "days": 0, "by": "telegram"})
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, true, out["blocked"])
	assert.Equal(t, "jane@example.com", out["email"])

	_, out = app.call(http.MethodGet, "/internal/v1/blocked-users", testInternalKey, "", nil)
	assert.Equal(t, []interface{}{float64(jane.ID)}, out["user_ids"])

	// Her session is refused, with a code the app understands
	w, out = app.call(http.MethodGet, "/api/v1/tasks", "", janeToken, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, middleware.AccountUnavailable, out["code"])
	// Her gig is off the board; Bob is unaffected
	assert.Equal(t, 0, board())

	// Asking for a code looks the same, but nothing is sent
	w, _ = app.call(http.MethodPost, "/auth/request-code", "", "", gin.H{"email": "jane@example.com"})
	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Empty(t, app.sender.codes)

	_, out = app.call(http.MethodGet, "/internal/v1/email-blocks/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, true, out["blocked"])
	assert.Equal(t, "spam", out["reason"])
	_, out = app.call(http.MethodGet, "/internal/v1/email-blocks", testInternalKey, "", nil)
	assert.Len(t, out["blocks"], 1)

	// Unblocked: all back
	w, _ = app.call(http.MethodDelete, "/internal/v1/email-blocks/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	w, _ = app.call(http.MethodGet, "/api/v1/tasks", "", janeToken, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, board())
	w, _ = app.call(http.MethodDelete, "/internal/v1/email-blocks/jane@example.com", testInternalKey, "", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestBlockingRefusesBadInput(t *testing.T) {
	app := setupBlockingApp(t)
	w, out := app.call(http.MethodPost, "/internal/v1/email-blocks", testInternalKey, "", gin.H{"email": "nope"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, services.ErrInvalidEmail.Error(), out["error"])
	w, _ = app.call(http.MethodPost, "/internal/v1/email-blocks", testInternalKey, "", gin.H{"email": "a@example.com", "days": -2})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestBlockedCodeAlreadySentIsRefused(t *testing.T) {
	app := setupBlockingApp(t)
	app.user(t, "jane")
	app.call(http.MethodPost, "/auth/request-code", "", "", gin.H{"email": "jane@example.com"})
	code := app.sender.codes["jane@example.com"]
	app.call(http.MethodPost, "/internal/v1/email-blocks", testInternalKey, "", gin.H{"email": "jane@example.com"})

	w, out := app.call(http.MethodPost, "/auth/verify-code", "", "", gin.H{"email": "jane@example.com", "code": code})
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, middleware.AccountUnavailable, out["code"])
}
