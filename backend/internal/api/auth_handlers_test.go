package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"myguy/internal/middleware"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"myguy/internal/services"
)

type capturingSender struct {
	codes map[string]string
}

func (s *capturingSender) SendLoginCode(_ context.Context, email, code string) error {
	s.codes[email] = code
	return nil
}

// setupAuthRouter wires the real auth stack over in-memory SQLite.
func setupAuthRouter(t *testing.T) (*gin.Engine, *capturingSender, *repositories.GormUserRepository, *middleware.JWTAuthMiddleware) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.LoginCode{}); err != nil {
		t.Fatal(err)
	}

	userRepo := repositories.NewGormUserRepository(db)
	sender := &capturingSender{codes: map[string]string{}}
	jwt := middleware.NewJWTAuthMiddleware("test-secret")
	authService := services.NewAuthService(userRepo, repositories.NewGormLoginCodeRepository(db), sender, "test-secret")
	handler := NewHandler(authService, services.NewUserService(userRepo), nil, nil, jwt)

	router := gin.New()
	router.POST("/auth/request-code", handler.RequestLoginCode)
	router.POST("/auth/verify-code", handler.VerifyLoginCode)
	router.POST("/auth/complete-signup", handler.CompleteSignup)
	return router, sender, userRepo, jwt
}

func postJSON(router *gin.Engine, path string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var out map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

func TestAuthHandlers_NewUserSignup(t *testing.T) {
	router, sender, _, jwt := setupAuthRouter(t)

	w, _ := postJSON(router, "/auth/request-code", gin.H{"email": "new@example.com"})
	assert.Equal(t, http.StatusAccepted, w.Code)

	w, body := postJSON(router, "/auth/verify-code", gin.H{"email": "new@example.com", "code": sender.codes["new@example.com"]})
	assert.Equal(t, http.StatusOK, w.Code)
	signupToken, _ := body["signup_token"].(string)
	assert.NotEmpty(t, signupToken)
	assert.Nil(t, body["token"], "no session before sign-up completes")

	w, body = postJSON(router, "/auth/complete-signup", gin.H{"signup_token": signupToken, "full_name": "New Person"})
	assert.Equal(t, http.StatusCreated, w.Code)
	user := body["user"].(map[string]interface{})
	assert.Equal(t, "new", user["username"])
	assert.Equal(t, "New Person", user["full_name"])

	claims, err := jwt.ValidateToken(body["token"].(string))
	assert.NoError(t, err)
	assert.Equal(t, "new@example.com", claims.Email)

	// Replaying the signup token cannot create a second account.
	w, _ = postJSON(router, "/auth/complete-signup", gin.H{"signup_token": signupToken, "full_name": "Again"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAuthHandlers_ExistingUserLogin(t *testing.T) {
	router, sender, userRepo, jwt := setupAuthRouter(t)
	assert.NoError(t, userRepo.Create(context.Background(), &models.User{Username: "jane", Email: "jane@example.com", FullName: "Jane"}))

	postJSON(router, "/auth/request-code", gin.H{"email": "jane@example.com"})
	w, body := postJSON(router, "/auth/verify-code", gin.H{"email": "jane@example.com", "code": sender.codes["jane@example.com"]})

	assert.Equal(t, http.StatusOK, w.Code)
	claims, err := jwt.ValidateToken(body["token"].(string))
	assert.NoError(t, err)
	assert.Equal(t, "jane", claims.Username)
}

func TestAuthHandlers_Errors(t *testing.T) {
	router, sender, _, _ := setupAuthRouter(t)

	t.Run("invalid email", func(t *testing.T) {
		w, _ := postJSON(router, "/auth/request-code", gin.H{"email": "not-an-email"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("malformed code", func(t *testing.T) {
		w, _ := postJSON(router, "/auth/verify-code", gin.H{"email": "a@example.com", "code": "12ab"})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("wrong code", func(t *testing.T) {
		postJSON(router, "/auth/request-code", gin.H{"email": "b@example.com"})
		code := "000000"
		if sender.codes["b@example.com"] == code {
			code = "111111"
		}
		w, _ := postJSON(router, "/auth/verify-code", gin.H{"email": "b@example.com", "code": code})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("forged signup token", func(t *testing.T) {
		w, _ := postJSON(router, "/auth/complete-signup", gin.H{"signup_token": "forged", "full_name": "X"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rate limited", func(t *testing.T) {
		var w *httptest.ResponseRecorder
		for i := 0; i < 6; i++ {
			w, _ = postJSON(router, "/auth/request-code", gin.H{"email": "c@example.com"})
		}
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})
}
