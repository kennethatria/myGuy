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
		itemImage = item.Images[0]
	}

	payload := ChatNotificationPayload{
		BookingID:  booking.ID,
		ItemID:     item.ID,
		ItemTitle:  item.Title,
		ItemImage:  itemImage,
		BuyerID:    booking.RequesterID,
		SellerID:   item.SellerID,
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
