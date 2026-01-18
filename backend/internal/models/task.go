package models

import (
	"time"
)

type Task struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	Title       string     `json:"title" gorm:"not null"`
	Description string     `json:"description" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:'open';index:idx_tasks_status"`
	CreatedBy   uint       `json:"created_by" gorm:"not null;index:idx_tasks_created_by"`
	AssignedTo  *uint      `json:"assigned_to" gorm:"index:idx_tasks_assigned_to"`
	Fee                float64    `json:"fee" gorm:"index:idx_tasks_fee"`
	Deadline           time.Time  `json:"deadline" gorm:"index:idx_tasks_deadline"`
	CompletedAt        *time.Time `json:"completed_at"`
	IsMessagesPublic   bool       `json:"is_messages_public" gorm:"default:false"`
	CreatedAt          time.Time  `json:"created_at" gorm:"index:idx_tasks_created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Define relationships for preloading
	Creator   User          `json:"creator" gorm:"foreignKey:CreatedBy"`
	Assignee  *User         `json:"assignee" gorm:"foreignKey:AssignedTo"`
	Applications []Application `json:"applications" gorm:"foreignKey:TaskID"`
}

type Application struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TaskID      uint      `json:"task_id" gorm:"not null;index:idx_applications_task_id"`
	ApplicantID uint      `json:"applicant_id" gorm:"not null;index:idx_applications_applicant_id"`
	ProposedFee float64   `json:"proposed_fee"`
	Status      string    `json:"status" gorm:"default:'pending';index:idx_applications_status"`
	Message     string    `json:"message" gorm:"type:text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Define relationships for preloading
	Applicant User      `json:"applicant" gorm:"foreignKey:ApplicantID"`
	Task      Task      `json:"task" gorm:"foreignKey:TaskID"`
}

type Review struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	TaskID         uint      `json:"task_id" gorm:"not null;index:idx_reviews_task_id;uniqueIndex:idx_reviews_task_reviewer,priority:1"`
	ReviewerID     uint      `json:"reviewer_id" gorm:"not null;index:idx_reviews_reviewer_id;uniqueIndex:idx_reviews_task_reviewer,priority:2"`
	ReviewedUserID uint      `json:"reviewed_user_id" gorm:"not null;index:idx_reviews_reviewed_user_id"`
	Rating         int       `json:"rating" gorm:"not null;check:rating >= 1 AND rating <= 5"`
	Comment        string    `json:"comment" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"`

	// Define relationships for preloading
	Task         Task `json:"task" gorm:"foreignKey:TaskID"`
	Reviewer     User `json:"reviewer" gorm:"foreignKey:ReviewerID"`
	ReviewedUser User `json:"reviewed_user" gorm:"foreignKey:ReviewedUserID"`
}
