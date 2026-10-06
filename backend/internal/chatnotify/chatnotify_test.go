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

// capture starts a chat server that hands each posted message to the test.
func capture(t *testing.T) (*httptest.Server, chan taskMessage) {
	got := make(chan taskMessage, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg taskMessage
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&msg))
		got <- msg
		w.WriteHeader(http.StatusCreated)
	}))
	return server, got
}

func TestPostTagsTheEvent(t *testing.T) {
	server, got := capture(t)
	defer server.Close()

	New(server.URL, "secret").Post(Message{
		TaskID: 1, SenderID: 2, RecipientID: 3, Content: "accepted",
		Event: EventAccepted, ApplicationID: 7, UnlockContacts: true,
	})

	assert.Equal(t, taskMessage{
		TaskID: 1, SenderID: 2, RecipientID: 3, Content: "accepted", UnlockContacts: true,
		Metadata: &metadata{Event: "accepted", ApplicationID: 7},
	}, <-got)
}

func TestPostWithoutEventSendsNoMetadata(t *testing.T) {
	server, got := capture(t)
	defer server.Close()

	New(server.URL, "secret").Post(Message{TaskID: 1, SenderID: 2, RecipientID: 3, Content: "hi"})

	assert.Nil(t, (<-got).Metadata)
}

func TestUnlockRecordsTheMatchOnly(t *testing.T) {
	server, got := capture(t)
	defer server.Close()

	assert.NoError(t, New(server.URL, "secret").Unlock(1, 9, 2))
	assert.Equal(t, taskMessage{TaskID: 1, SenderID: 9, RecipientID: 2, UnlockContacts: true}, <-got)
}
