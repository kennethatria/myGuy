package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type JWTAuthMiddleware struct {
	secretKey string
}

func NewJWTAuthMiddleware(secretKey string) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		secretKey: secretKey,
	}
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	jwt.RegisteredClaims
}

func (m *JWTAuthMiddleware) GenerateToken(userID uint, username, email, name string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Name:     name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secretKey))
}

func (m *JWTAuthMiddleware) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidToken
}

// signupTokenTTL bounds how long a verified email may take to finish sign-up.
const signupTokenTTL = 15 * time.Minute

type signupClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// signupKey is derived from the session secret so signup tokens can never
// pass ValidateToken here, or the session checks in store and chat services.
func (m *JWTAuthMiddleware) signupKey() []byte {
	return []byte(m.secretKey + ":signup")
}

// GenerateSignupToken proves email was verified by login code, for an email
// that has no account yet.
func (m *JWTAuthMiddleware) GenerateSignupToken(email string) (string, error) {
	claims := signupClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(signupTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.signupKey())
}

// ValidateSignupToken returns the verified email carried by a signup token.
func (m *JWTAuthMiddleware) ValidateSignupToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &signupClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.signupKey(), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*signupClaims)
	if !ok || !token.Valid || claims.Email == "" {
		return "", ErrInvalidToken
	}
	return claims.Email, nil
}

func (m *JWTAuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := m.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Next()
	}
}
