package models

import (
	"time"
)

type Task struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description" gorm:"type:text"`
	Status      string    `json:"status" gorm:"default:'open'"`
	CreatedBy   uint      `json:"created_by" gorm:"not null"`
	AssignedTo  *uint     `json:"assigned_to"`
	Fee         float64   `json:"fee"`
	Deadline    time.Time `json:"deadline"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Define relationships for preloading
	Creator   User          `json:"creator" gorm:"foreignKey:CreatedBy"`
	Assignee  *User         `json:"assignee" gorm:"foreignKey:AssignedTo"`
	Applications []Application `json:"applications" gorm:"foreignKey:TaskID"`
}

type Application struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"not null"`
	ApplicantID uint      `json:"applicant_id" gorm:"not null"`
	ProposedFee float64   `json:"proposed_fee"`
	Status      string    `json:"status" gorm:"default:'pending'"`
	Message     string    `json:"message" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	
	// Define relationship for preloading
	Applicant User `json:"applicant" gorm:"foreignKey:ApplicantID"`
}

type Message struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"not null"`
	SenderID    uint      `json:"sender_id" gorm:"not null"`
	RecipientID uint      `json:"recipient_id" gorm:"not null"`
	Content     string    `json:"content" gorm:"type:text;not null"`
	CreatedAt   time.Time `json:"created_at"`
}

type Review struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	TaskID         uint      `json:"task_id" gorm:"not null"`
	ReviewerID     uint      `json:"reviewer_id" gorm:"not null"`
	ReviewedUserID uint      `json:"reviewed_user_id" gorm:"not null"`
	Rating         int       `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment        string    `json:"comment" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
}
