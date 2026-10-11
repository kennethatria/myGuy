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
