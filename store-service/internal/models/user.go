package models

import "time"

// User represents the user data needed by the store service
// This is a simplified version that will be populated from JWT claims
type User struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}