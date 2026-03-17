package middleware

import (
	"net/http"
	"net/http/httptest"
	"store-service/internal/models"
	"store-service/internal/repositories"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Mock UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id uint) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) UpsertFromJWT(userID uint, username, email, name string) (*models.User, error) {
	args := m.Called(userID, username, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) UpdateRating(userID uint, newRating float64) error {
	args := m.Called(userID, newRating)
	return args.Error(0)
}

func TestJWTAuthMiddleware_ValidateToken(t *testing.T) {
	secretKey := "test-secret-key"
	middleware := NewJWTAuthMiddleware(secretKey, nil)

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

	t.Run("successful authentication with user sync", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		middleware := NewJWTAuthMiddleware(secretKey, mockUserRepo)

		// Mock the UpsertFromJWT call
		expectedUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			Name:     "Test User",
		}
		mockUserRepo.On("UpsertFromJWT", uint(1), "testuser", "test@example.com", "Test User").Return(expectedUser, nil)

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

		// Setup test route
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			userID := c.GetUint("userID")
			username := c.GetString("username")
			email := c.GetString("userEmail")
			name := c.GetString("userName")

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
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		middleware := NewJWTAuthMiddleware(secretKey, nil)

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
		middleware := NewJWTAuthMiddleware(secretKey, nil)

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
		middleware := NewJWTAuthMiddleware(secretKey, nil)

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

	t.Run("user sync fails but request continues", func(t *testing.T) {
		mockUserRepo := new(MockUserRepository)
		middleware := NewJWTAuthMiddleware(secretKey, mockUserRepo)

		// Mock the UpsertFromJWT call to fail
		mockUserRepo.On("UpsertFromJWT", uint(1), "testuser", "test@example.com", "Test User").Return(nil, gorm.ErrInvalidData)

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

		// Setup test route
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(middleware.AuthRequired())
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		// Make request with valid token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Request should still succeed even if user sync fails
		assert.Equal(t, http.StatusOK, w.Code)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestJWTAuthMiddleware_NewJWTAuthMiddleware(t *testing.T) {
	secretKey := "test-secret"

	t.Run("without user repository", func(t *testing.T) {
		middleware := NewJWTAuthMiddleware(secretKey, nil)
		assert.NotNil(t, middleware)
		assert.Equal(t, secretKey, middleware.secretKey)
		assert.Nil(t, middleware.userRepo)
	})

	t.Run("with user repository", func(t *testing.T) {
		db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		userRepo := repositories.NewUserRepository(db)
		middleware := NewJWTAuthMiddleware(secretKey, userRepo)
		assert.NotNil(t, middleware)
		assert.Equal(t, secretKey, middleware.secretKey)
		assert.NotNil(t, middleware.userRepo)
	})
}
