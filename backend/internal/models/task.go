package models

import (
	"time"
)

type Task struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:'open'"`
	CreatedBy   uint       `json:"created_by" gorm:"not null"`
	AssignedTo  *uint      `json:"assigned_to"`
	Fee         float64    `json:"fee"`
	Deadline    time.Time  `json:"deadline"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	
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
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Define relationships for preloading
	Applicant User      `json:"applicant" gorm:"foreignKey:ApplicantID"`
	Task      Task      `json:"task" gorm:"foreignKey:TaskID"`
	Messages  []Message `json:"messages,omitempty" gorm:"foreignKey:ApplicationID"`
}

type Message struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	TaskID          uint       `json:"task_id" gorm:"not null"`
	ApplicationID   *uint      `json:"application_id"` // Optional: links message to specific application
	SenderID        uint       `json:"sender_id" gorm:"not null"`
	RecipientID     uint       `json:"recipient_id" gorm:"not null"`
	Content         string     `json:"content" gorm:"type:text;not null"`
	OriginalContent string     `json:"-" gorm:"type:text"` // Store original before filtering
	IsRead          bool       `json:"is_read" gorm:"default:false"`
	ReadAt          *time.Time `json:"read_at"`
	IsEdited        bool       `json:"is_edited" gorm:"default:false"`
	EditedAt        *time.Time `json:"edited_at"`
	IsDeleted       bool       `json:"is_deleted" gorm:"default:false"`
	DeletedAt       *time.Time `json:"deleted_at"`
	CreatedAt       time.Time  `json:"created_at"`
	
	// Define relationships for preloading
	Sender      User         `json:"sender" gorm:"foreignKey:SenderID"`
	Recipient   User         `json:"recipient" gorm:"foreignKey:RecipientID"`
	Task        Task         `json:"task,omitempty" gorm:"foreignKey:TaskID"`
	Application *Application `json:"application,omitempty" gorm:"foreignKey:ApplicationID"`
	
	// Non-database fields
	HasRemovedContent bool `json:"has_removed_content,omitempty" gorm:"-"`
}

// ConversationSummary represents a conversation in the user's message list
type ConversationSummary struct {
	TaskID            uint       `json:"task_id"`
	TaskTitle         string     `json:"task_title"`
	TaskDescription   string     `json:"task_description"`
	TaskStatus        string     `json:"task_status"`
	LastMessage       string     `json:"last_message"`
	LastMessageTime   time.Time  `json:"last_message_time"`
	OtherUserID       uint       `json:"other_user_id"`
	OtherUserName     string     `json:"other_user_name"`
	UnreadCount       int        `json:"unread_count"`
}

type Review struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	TaskID         uint      `json:"task_id" gorm:"not null"`
	ReviewerID     uint      `json:"reviewer_id" gorm:"not null"`
	ReviewedUserID uint      `json:"reviewed_user_id" gorm:"not null"`
	Rating         int       `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment        string    `json:"comment" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`
	
	// Define relationships for preloading
	Task         Task `json:"task" gorm:"foreignKey:TaskID"`
	Reviewer     User `json:"reviewer" gorm:"foreignKey:ReviewerID"`
	ReviewedUser User `json:"reviewed_user" gorm:"foreignKey:ReviewedUserID"`
}
