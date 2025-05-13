package repositories

import (
	"context"
	"myguy/internal/models"
	"gorm.io/gorm"
)

type GormTaskRepository struct {
	db *gorm.DB
}

func NewGormTaskRepository(db *gorm.DB) *GormTaskRepository {
	return &GormTaskRepository{db: db}
}

func (r *GormTaskRepository) Create(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *GormTaskRepository) GetByID(ctx context.Context, id uint) (*models.Task, error) {
	var task models.Task
	err := r.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormTaskRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	var tasks []models.Task
	query := r.db.WithContext(ctx)

	// Apply filters
	for key, value := range filters {
		query = query.Where(key, value)
	}

	err := query.Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *GormTaskRepository) Update(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *GormTaskRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Task{}, id).Error
}

func (r *GormTaskRepository) ListByUser(ctx context.Context, userID uint, role string) ([]models.Task, error) {
	var tasks []models.Task
	query := r.db.WithContext(ctx)

	switch role {
	case "creator":
		query = query.Where("created_by = ?", userID)
	case "assigned":
		query = query.Where("assigned_to = ?", userID)
	default:
		return nil, nil
	}

	err := query.Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}
