package repositories

import (
	"context"
	"myguy/internal/models"
	"gorm.io/gorm"
)

type GormApplicationRepository struct {
	db *gorm.DB
}

func NewGormApplicationRepository(db *gorm.DB) *GormApplicationRepository {
	return &GormApplicationRepository{db: db}
}

func (r *GormApplicationRepository) Create(ctx context.Context, application *models.Application) error {
	return r.db.WithContext(ctx).Create(application).Error
}

func (r *GormApplicationRepository) GetByID(ctx context.Context, id uint) (*models.Application, error) {
	var application models.Application
	err := r.db.WithContext(ctx).
		Preload("Applicant").
		Preload("Task").
		First(&application, id).Error
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *GormApplicationRepository) ListByTask(ctx context.Context, taskID uint) ([]models.Application, error) {
	var applications []models.Application
	err := r.db.WithContext(ctx).
		Preload("Applicant").
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&applications).Error
	if err != nil {
		return nil, err
	}
	return applications, nil
}

func (r *GormApplicationRepository) ListByUser(ctx context.Context, userID uint) ([]models.Application, error) {
	var applications []models.Application
	err := r.db.WithContext(ctx).Where("applicant_id = ?", userID).Find(&applications).Error
	if err != nil {
		return nil, err
	}
	return applications, nil
}

func (r *GormApplicationRepository) Update(ctx context.Context, application *models.Application) error {
	return r.db.WithContext(ctx).Save(application).Error
}
