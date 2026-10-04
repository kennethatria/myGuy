package models

import (
	"time"
)

type User struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"unique;not null"`
	Email        string    `json:"email" gorm:"unique;not null"`
	Password     string    `json:"-" gorm:"not null"`
	FullName     string    `json:"full_name"`
	PhoneNumber  string    `json:"phone_number"`
	Bio          string    `json:"bio"`
	AverageRating float64  `json:"average_rating"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserResponse struct {
	ID           uint      `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	FullName     string    `json:"full_name"`
	PhoneNumber  string    `json:"phone_number"`
	Bio          string    `json:"bio"`
	AverageRating float64  `json:"average_rating"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		FullName:      u.FullName,
		PhoneNumber:   u.PhoneNumber,
		Bio:           u.Bio,
		AverageRating: u.AverageRating,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}
