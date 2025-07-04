package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJWTAuthMiddleware_GenerateToken(t *testing.T) {
	secretKey := "test-secret-key"
	middleware := NewJWTAuthMiddleware(secretKey)

	t.Run("successful token generation", func(t *testing.T) {
		userID := uint(123)
		username := "testuser"
		email := "test@example.com"
		name := "Test User"

		tokenString, err := middleware.GenerateToken(userID, username, email, name)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// Validate the generated token
		claims, err := middleware.ValidateToken(tokenString)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, name, claims.Name)
	})

	t.Run("token with empty fields", func(t *testing.T) {
		userID := uint(456)
		username := ""
		email := ""
		name := ""

		tokenString, err := middleware.GenerateToken(userID, username, email, name)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		// Validate the generated token
		claims, err := middleware.ValidateToken(tokenString)
		assert.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, username, claims.Username)
		assert.Equal(t, email, claims.Email)
		assert.Equal(t, name, claims.Name)
	})
}

func TestJWTAuthMiddleware_ValidateToken(t *testing.T) {
	secretKey := "test-secret-key"
	middleware := NewJWTAuthMiddleware(secretKey)

	t.Run("valid token", func(t *testing.T) {
		// Create a valid token
		claims := Claims{
			UserID:   1,
			Username: "testuser",
			Email:    "test@example.com",
			Name:     "Test User",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secretKey))
		assert.NoError(t, err)

		// Validate the token
		validatedClaims, err := middleware.ValidateToken(tokenString)
		assert.NoError(t, err)
		assert.Equal(t, uint(1), validatedClaims.UserID)
		assert.Equal(t, "testuser", validatedClaims.Username)
		assert.Equal(t, "test@example.com", validatedClaims.Email)
		assert.Equal(t, "Test User", validatedClaims.Name)
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired token
		claims := Claims{
			UserID:   1,
			Username: "testuser",
			Email:    "test@example.com",
			Name:     "Test User",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // Expired
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secretKey))
		assert.NoError(t, err)

		// Validate the token
		_, err = middleware.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Equal(t, ErrExpiredToken, err)
	})

	t.Run("invalid token", func(t *testing.T) {
		// Test with invalid token string
		_, err := middleware.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
	})

	t.Run("token with wrong secret", func(t *testing.T) {
		// Create token with different secret
		claims := Claims{
			UserID:   1,
			Username: "testuser",
			Email:    "test@example.com",
			Name:     "Test User",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		assert.NoError(t, err)

		// Validate with correct secret
		_, err = middleware.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
	})
}

func TestJWTAuthMiddleware_AuthRequired(t *testing.T) {
	secretKey := "test-secret-key"
	middleware := NewJWTAuthMiddleware(secretKey)

	t.Run("successful authentication", func(t *testing.T) {
		// Create a valid token
		tokenString, err := middleware.GenerateToken(1, "testuser", "test@example.com", "Test User")
		assert.NoError(t, err)

		// Setup test route
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			userID := c.GetUint("userID")
			username := c.GetString("username")
			email := c.GetString("email")
			name := c.GetString("name")
			
			c.JSON(http.StatusOK, gin.H{
				"userID":   userID,
				"username": username,
				"email":    email,
				"name":     name,
			})
		})

		// Make request with valid token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("invalid token", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestNewJWTAuthMiddleware(t *testing.T) {
	secretKey := "test-secret"
	middleware := NewJWTAuthMiddleware(secretKey)
	
	assert.NotNil(t, middleware)
	assert.Equal(t, secretKey, middleware.secretKey)
}