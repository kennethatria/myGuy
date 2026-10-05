package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPChatNotifier(t *testing.T) {
	t.Run("without a key it does nothing", func(t *testing.T) {
		t.Setenv("INTERNAL_API_KEY", "")
		_, isNoop := NewHTTPChatNotifier().(noopChatNotifier)
		assert.True(t, isNoop)
	})

	t.Run("posts the store message with the internal key", func(t *testing.T) {
		got := make(chan map[string]interface{}, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/internal/store-message", r.URL.Path)
			assert.Equal(t, "secret", r.Header.Get("X-Internal-API-Key"))
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusCreated)
			got <- body
		}))
		defer server.Close()
		t.Setenv("INTERNAL_API_KEY", "secret")
		t.Setenv("CHAT_API_URL", server.URL)

		NewHTTPChatNotifier().StoreMessage(9, 1, 2, "Listed for you")

		select {
		case body := <-got:
			assert.Equal(t, float64(9), body["store_item_id"])
			assert.Equal(t, float64(2), body["recipient_id"])
			assert.Equal(t, "Listed for you", body["content"])
		case <-time.After(2 * time.Second):
			t.Fatal("no message posted")
		}
	})

	t.Run("reports a refusal", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()
		n := &HTTPChatNotifier{baseURL: server.URL, apiKey: "x", client: server.Client()}
		assert.ErrorContains(t, n.post(9, 1, 2, "hi"), "401")
	})
}
