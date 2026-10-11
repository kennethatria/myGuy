package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthRequiredRefusesBlockedAccounts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := NewJWTAuthMiddleware("secret", nil).WithBlocked(func(id uint) bool { return id == 7 })
	r := gin.New()
	r.GET("/items", m.AuthRequired(), func(c *gin.Context) { c.Status(http.StatusOK) })

	call := func(userID uint) *httptest.ResponseRecorder {
		token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).
			SignedString([]byte("secret"))
		req := httptest.NewRequest(http.MethodGet, "/items", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	w := call(7)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, AccountUnavailable, body["code"])
	assert.Equal(t, http.StatusOK, call(8).Code)
}

func TestAuthRequiredKeepsSuspendedAccountsReadOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	until := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	m := NewJWTAuthMiddleware("secret", nil).WithSuspended(func(id uint) (time.Time, bool) { return until, id == 7 })
	r := gin.New()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/items", m.AuthRequired(), ok)
	r.POST("/items", m.AuthRequired(), ok)

	call := func(method string, userID uint) *httptest.ResponseRecorder {
		token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{UserID: userID,
			RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}}).
			SignedString([]byte("secret"))
		req := httptest.NewRequest(method, "/items", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusOK, call(http.MethodGet, 7).Code, "they can look")
	w := call(http.MethodPost, 7)
	assert.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, AccountSuspended, body["code"])
	assert.Equal(t, "2026-10-14T12:00:00Z", body["until"])
	assert.Equal(t, "your account is suspended until Wed 14 Oct 12:00 UTC", body["error"])
	assert.Equal(t, http.StatusOK, call(http.MethodPost, 8).Code)
}
