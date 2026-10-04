package models

import (
	"time"
)

// LoginCode is a one-time code emailed to a user for passwordless sign-in.
// Only an HMAC of the code is stored, never the code itself.
type LoginCode struct {
	ID         uint      `gorm:"primaryKey"`
	Email      string    `gorm:"index;not null"`
	CodeHash   string    `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	Attempts   int       `gorm:"not null;default:0"`
	ConsumedAt *time.Time
	CreatedAt  time.Time
}
