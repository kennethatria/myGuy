package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestClientIPComesFromNginxOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	TrustNginx(r)
	var got string
	r.GET("/ip", func(c *gin.Context) { got = c.ClientIP() })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.Header.Set("X-Forwarded-For", "6.6.6.6") // sent by the client, passed on by nginx
	req.Header.Set("X-Real-IP", "203.0.113.7")   // set by nginx
	r.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "203.0.113.7", got)
}
