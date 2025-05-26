package repositories

import (
	"context"
	"myguy/internal/models"
	"gorm.io/gorm"
)

type GormMessageRepository struct {
	db *gorm.DB
}

func NewGormMessageRepository(db *gorm.DB) *GormMessageRepository {
	return &GormMessageRepository{db: db}
}

func (r *GormMessageRepository) Create(ctx context.Context, message *models.Message) error {
	err := r.db.WithContext(ctx).Create(message).Error
	if err != nil {
		return err
	}
	
	// Reload message with related data
	return r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		First(message, message.ID).Error
}

func (r *GormMessageRepository) ListByTask(ctx context.Context, taskID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) ListByUser(ctx context.Context, userID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("sender_id = ? OR recipient_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) ListByApplication(ctx context.Context, applicationID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("application_id = ?", applicationID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) GetApplication(ctx context.Context, applicationID uint) (*models.Application, error) {
	var application models.Application
	err := r.db.WithContext(ctx).First(&application, applicationID).Error
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *GormMessageRepository) GetTask(ctx context.Context, taskID uint) (*models.Task, error) {
	var task models.Task
	err := r.db.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}
