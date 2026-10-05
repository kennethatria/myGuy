package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestInternalKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	call := func(key, sent string) int {
		r := gin.New()
		r.GET("/x", InternalKey(key), func(c *gin.Context) { c.Status(http.StatusNoContent) })
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/x", nil)
		if sent != "" {
			req.Header.Set("X-Internal-API-Key", sent)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusNoContent, call("secret", "secret"))
	assert.Equal(t, http.StatusUnauthorized, call("secret", "wrong"))
	assert.Equal(t, http.StatusUnauthorized, call("secret", ""))
	assert.Equal(t, http.StatusUnauthorized, call("", ""), "no key configured refuses everyone")
}
