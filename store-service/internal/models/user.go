package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents the user data needed by the store service
// This is a simplified version that will be populated from JWT claims
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null"`
	Name      string         `json:"name"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}