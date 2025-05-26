package repositories

import (
	"context"
	"fmt"
	"strings"
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
	// Preload applications and related user data to provide complete task information
	err := r.db.WithContext(ctx).
		Preload("Applications.Applicant").
		Preload("Creator").
		Preload("Assignee").
		First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormTaskRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	var tasks []models.Task
	query := r.buildTaskQuery(ctx, filters)

	// Order by most recent first and preload related data
	err := query.
		Preload("Applications.Applicant").
		Preload("Creator").
		Preload("Assignee").
		Order("created_at DESC").
		Find(&tasks).Error
	
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *GormTaskRepository) ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	var tasks []models.Task
	query := r.buildTaskQuery(ctx, filters)
	
	// Extract pagination
	page := 1
	perPage := 20
	if p, ok := filters["page"].(int); ok {
		page = p
	}
	if pp, ok := filters["per_page"].(int); ok {
		perPage = pp
	}
	
	// Apply sorting
	sortBy := "created_at"
	sortOrder := "DESC"
	if sb, ok := filters["sort_by"].(string); ok {
		switch sb {
		case "fee", "deadline", "created_at":
			sortBy = sb
		}
	}
	if so, ok := filters["sort_order"].(string); ok && (so == "asc" || so == "desc") {
		sortOrder = strings.ToUpper(so)
	}
	
	offset := (page - 1) * perPage
	err := query.
		Preload("Applications.Applicant").
		Preload("Creator").
		Preload("Assignee").
		Order(fmt.Sprintf("%s %s", sortBy, sortOrder)).
		Offset(offset).
		Limit(perPage).
		Find(&tasks).Error
	
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *GormTaskRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	var count int64
	query := r.buildTaskQuery(ctx, filters)
	err := query.Model(&models.Task{}).Count(&count).Error
	return count, err
}

func (r *GormTaskRepository) buildTaskQuery(ctx context.Context, filters map[string]interface{}) *gorm.DB {
	query := r.db.WithContext(ctx)
	
	// Apply filters
	for key, value := range filters {
		switch key {
		case "search":
			// Search in title and description
			searchTerm := fmt.Sprintf("%%%s%%", value)
			query = query.Where("(title ILIKE ? OR description ILIKE ?)", searchTerm, searchTerm)
		case "min_fee":
			query = query.Where("fee >= ?", value)
		case "max_fee":
			query = query.Where("fee <= ?", value)
		case "deadline_before":
			query = query.Where("deadline <= ?", value)
		case "exclude_created_by":
			query = query.Where("created_by != ?", value)
		case "status", "created_by", "assigned_to":
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		// Skip pagination and sorting params
		case "page", "per_page", "sort_by", "sort_order":
			// These are handled separately
		default:
			// For any other filters, apply them directly
			if key != "" {
				query = query.Where(fmt.Sprintf("%s = ?", key), value)
			}
		}
	}
	
	return query
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
		// Make sure assigned_to is not NULL and equals the user ID
		// The IS NOT NULL check is important to avoid returning tasks where assigned_to is NULL
		query = query.Where("assigned_to IS NOT NULL AND assigned_to = ?", userID)
	default:
		return nil, nil
	}

	// Preload related data and order by most recent first
	err := query.
		Preload("Applications.Applicant").
		Preload("Creator").
		Preload("Assignee").
		Order("created_at DESC").
		Find(&tasks).Error
	
	if err != nil {
		return nil, err
	}
	return tasks, nil
}
