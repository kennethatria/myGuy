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
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *GormMessageRepository) ListByTask(ctx context.Context, taskID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
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
		Where("sender_id = ? OR recipient_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}
