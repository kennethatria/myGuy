package chatnotify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendPostsTaskMessageWithInternalKey(t *testing.T) {
	var got taskMessage
	var key string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/internal/task-message", r.URL.Path)
		key = r.Header.Get("X-Internal-API-Key")
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	n := New(server.URL+"/api/v1", "secret")
	err := n.send(taskMessage{TaskID: 1, SenderID: 2, RecipientID: 3, Content: "hi"})

	assert.NoError(t, err)
	assert.Equal(t, "secret", key)
	assert.Equal(t, taskMessage{TaskID: 1, SenderID: 2, RecipientID: 3, Content: "hi"}, got)
}

func TestSendReportsChatErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	assert.Error(t, New(server.URL, "wrong").send(taskMessage{TaskID: 1}))
}
