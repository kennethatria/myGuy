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

// ListAccepted returns every accepted application with its task: each pair
// of poster and applicant who agreed to work together.
func (r *GormApplicationRepository) ListAccepted(ctx context.Context) ([]models.Application, error) {
	var applications []models.Application
	err := r.db.WithContext(ctx).
		Preload("Task").
		Where("status = ?", "accepted").
		Find(&applications).Error
	if err != nil {
		return nil, err
	}
	return applications, nil
}

// ListByUser returns userID's applications, newest first, with each task and
// its poster loaded for the "My applications" list.
func (r *GormApplicationRepository) ListByUser(ctx context.Context, userID uint) ([]models.Application, error) {
	var applications []models.Application
	err := r.db.WithContext(ctx).
		Preload("Task").
		Preload("Task.Creator").
		Where("applicant_id = ?", userID).
		Order("created_at DESC").
		Find(&applications).Error
	if err != nil {
		return nil, err
	}
	return applications, nil
}

func (r *GormApplicationRepository) Update(ctx context.Context, application *models.Application) error {
	return r.db.WithContext(ctx).Save(application).Error
}

// DeclinePending declines every still-pending application for the task except
// exceptID (0 declines all), so applicants are never left waiting.
func (r *GormApplicationRepository) DeclinePending(ctx context.Context, taskID, exceptID uint) error {
	return r.db.WithContext(ctx).Model(&models.Application{}).
		Where("task_id = ? AND id <> ? AND status = ?", taskID, exceptID, "pending").
		Update("status", "declined").Error
}
