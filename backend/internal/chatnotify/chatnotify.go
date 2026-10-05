// Package chatnotify posts system messages into task conversations in the
// chat service, so task events (new application, accepted, declined) reach
// people in Messages like any other message.
package chatnotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Notifier posts to the chat service's internal endpoint.
type Notifier struct {
	url    string
	apiKey string
	client *http.Client
}

// New returns a Notifier for the chat API at chatAPIURL (e.g.
// http://chat-websocket-service:8082/api/v1), authenticated with the shared
// INTERNAL_API_KEY.
func New(chatAPIURL, apiKey string) *Notifier {
	return &Notifier{
		url:    chatAPIURL + "/internal/task-message",
		apiKey: apiKey,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

type taskMessage struct {
	TaskID      uint   `json:"task_id"`
	SenderID    uint   `json:"sender_id"`
	RecipientID uint   `json:"recipient_id"`
	Content     string `json:"content"`
}

// TaskMessage delivers content in the background as a system message from
// senderID to recipientID in their conversation about taskID. Best effort:
// failures are logged, never returned, so a chat outage can't block the task
// action that triggered it.
func (n *Notifier) TaskMessage(taskID, senderID, recipientID uint, content string) {
	go func() {
		if err := n.send(taskMessage{TaskID: taskID, SenderID: senderID, RecipientID: recipientID, Content: content}); err != nil {
			log.Printf("chat notification for task %d failed: %v", taskID, err)
		}
	}()
}

func (n *Notifier) send(msg taskMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", n.apiKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("chat service responded %d", resp.StatusCode)
	}
	return nil
}
