package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"store-service/internal/models"
	"store-service/internal/repositories"
)

// ChatNotificationPayload represents the data sent to chat service
type ChatNotificationPayload struct {
	BookingID  uint   `json:"bookingId"`
	ItemID     uint   `json:"itemId"`
	ItemTitle  string `json:"itemTitle"`
	ItemImage  string `json:"itemImage,omitempty"`
	BuyerID    uint   `json:"buyerId"`
	SellerID   uint   `json:"sellerId"`
	Message    string `json:"message,omitempty"`
}

// NotifyChatServiceAboutBooking sends a notification to the chat service about a new booking request
func NotifyChatServiceAboutBooking(booking *models.BookingRequest, item *models.StoreItem, bookingRepo repositories.BookingRequestRepository) {
	chatAPIURL := os.Getenv("CHAT_API_URL")
	if chatAPIURL == "" {
		chatAPIURL = "http://localhost:8082/api/v1"
	}

	internalAPIKey := os.Getenv("INTERNAL_API_KEY")
	if internalAPIKey == "" {
		log.Printf("⚠️ INTERNAL_API_KEY not set, skipping chat notification for booking %d", booking.ID)
		markNotificationFailed(booking.ID, bookingRepo)
		return
	}

	// Get first image if available
	var itemImage string
	if len(item.Images) > 0 {
		        itemImage = item.Images[0].URL	}

	payload := ChatNotificationPayload{
		BookingID:  booking.ID,
		ItemID:     item.ID,
		ItemTitle:  item.Title,
		ItemImage:  itemImage,
		BuyerID:    booking.RequesterID,
		SellerID:   item.SellerID,
		Message:    booking.Message,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling chat notification payload: %v", err)
		markNotificationFailed(booking.ID, bookingRepo)
		return
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/internal/booking-created", chatAPIURL),
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		log.Printf("Error creating chat notification request: %v", err)
		markNotificationFailed(booking.ID, bookingRepo)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", internalAPIKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error notifying chat service: %v", err)
		markNotificationFailed(booking.ID, bookingRepo)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Chat service returned non-OK status: %d", resp.StatusCode)
		markNotificationFailed(booking.ID, bookingRepo)
		return
	}

	// Mark as successfully notified
	log.Printf("✅ Chat service notified successfully for booking %d", booking.ID)
	markNotificationSuccess(booking.ID, bookingRepo)
}

func markNotificationSuccess(bookingID uint, bookingRepo repositories.BookingRequestRepository) {
	bookingRepo.UpdateChatNotificationStatus(bookingID, true, 0)
}

func markNotificationFailed(bookingID uint, bookingRepo repositories.BookingRequestRepository) {
	bookingRepo.IncrementNotificationAttempts(bookingID)
}

// ChatNotifier posts store events into the conversation between two people
// about an item. Implementations must not block or fail the caller (chat is
// best effort).
type ChatNotifier interface {
	StoreMessage(itemID, senderID, recipientID uint, content string)
	// BookingClosed marks a booking's message in chat as declined by the
	// seller, with note telling the buyer why.
	BookingClosed(bookingID, sellerID uint, note string)
}

type noopChatNotifier struct{}

func (noopChatNotifier) StoreMessage(uint, uint, uint, string) {}
func (noopChatNotifier) BookingClosed(uint, uint, string)     {}

// HTTPChatNotifier posts to chat's /internal/store-message in the background.
type HTTPChatNotifier struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewHTTPChatNotifier reads CHAT_API_URL and INTERNAL_API_KEY; without the
// key it returns a notifier that does nothing (and says so once).
func NewHTTPChatNotifier() ChatNotifier {
	apiKey := os.Getenv("INTERNAL_API_KEY")
	if apiKey == "" {
		log.Println("⚠️ INTERNAL_API_KEY not set; request matches won't reach Messages")
		return noopChatNotifier{}
	}
	baseURL := os.Getenv("CHAT_API_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082/api/v1"
	}
	return &HTTPChatNotifier{baseURL: baseURL, apiKey: apiKey, client: &http.Client{Timeout: 5 * time.Second}}
}

func (n *HTTPChatNotifier) StoreMessage(itemID, senderID, recipientID uint, content string) {
	go func() {
		if err := n.post(itemID, senderID, recipientID, content); err != nil {
			log.Printf("⚠️ store message for item %d not delivered: %v", itemID, err)
		}
	}()
}

func (n *HTTPChatNotifier) BookingClosed(bookingID, sellerID uint, note string) {
	go func() {
		err := n.postJSON("/internal/booking-status", map[string]interface{}{
			"booking_id": bookingID, "actor_id": sellerID, "status": "rejected", "note": note,
		})
		if err != nil {
			log.Printf("⚠️ booking %d closure not delivered: %v", bookingID, err)
		}
	}()
}

func (n *HTTPChatNotifier) post(itemID, senderID, recipientID uint, content string) error {
	return n.postJSON("/internal/store-message", map[string]interface{}{
		"store_item_id": itemID, "sender_id": senderID, "recipient_id": recipientID, "content": content,
	})
}

func (n *HTTPChatNotifier) postJSON(path string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, n.baseURL+path, bytes.NewReader(body))
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
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("chat returned %d", resp.StatusCode)
	}
	return nil
}
