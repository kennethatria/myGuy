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

// Events name what a message is about, so the app can show the matching
// action on it (accept an application, mark a gig done, approve it, review).
const (
	EventApplication = "application" // applicant → poster: accept or decline
	EventAccepted    = "accepted"    // poster → applicant: mark as done when finished
	EventDeclined    = "declined"
	EventCancelled   = "cancelled"
	EventDone        = "done"      // assignee → poster: approve, or not yet
	EventNotDone     = "not_done"  // poster → assignee: mark as done again later
	EventCompleted   = "completed" // both may now review each other
)

// Message is one event in the conversation between two people about a gig.
type Message struct {
	TaskID, SenderID, RecipientID uint
	Content                       string
	Event                         string
	ApplicationID                 uint
	// UnlockContacts lets the two share contact details, and chat with each
	// other, from now on: they agreed to work together.
	UnlockContacts bool
}

type metadata struct {
	Event         string `json:"event,omitempty"`
	ApplicationID uint   `json:"application_id,omitempty"`
}

type taskMessage struct {
	TaskID         uint      `json:"task_id"`
	SenderID       uint      `json:"sender_id"`
	RecipientID    uint      `json:"recipient_id"`
	Content        string    `json:"content,omitempty"`
	UnlockContacts bool      `json:"unlock_contacts,omitempty"`
	Metadata       *metadata `json:"metadata,omitempty"`
}

// Post sends m to the chat service in the background; failures are logged.
func (n *Notifier) Post(m Message) {
	msg := taskMessage{
		TaskID: m.TaskID, SenderID: m.SenderID, RecipientID: m.RecipientID,
		Content: m.Content, UnlockContacts: m.UnlockContacts,
	}
	if m.Event != "" {
		msg.Metadata = &metadata{Event: m.Event, ApplicationID: m.ApplicationID}
	}
	go func() {
		if err := n.send(msg); err != nil {
			log.Printf("chat notification for task %d failed: %v", msg.TaskID, err)
		}
	}()
}

// Unlock records that two people were matched on a gig, without posting a
// message. Used to catch up matches made before chat recorded them.
func (n *Notifier) Unlock(taskID, userA, userB uint) error {
	return n.send(taskMessage{TaskID: taskID, SenderID: userA, RecipientID: userB, UnlockContacts: true})
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
