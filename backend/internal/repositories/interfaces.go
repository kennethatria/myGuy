package repositories

import (
	"context"
	"myguy/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateRating(ctx context.Context, userID uint) error
}

type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uint) (*models.Task, error)
	List(ctx context.Context, filters map[string]interface{}) ([]models.Task, error)
	ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]models.Task, error)
	Count(ctx context.Context, filters map[string]interface{}) (int64, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id uint) error
	ListByUser(ctx context.Context, userID uint, role string) ([]models.Task, error)
}

type ApplicationRepository interface {
	Create(ctx context.Context, application *models.Application) error
	GetByID(ctx context.Context, id uint) (*models.Application, error)
	ListByTask(ctx context.Context, taskID uint) ([]models.Application, error)
	ListByUser(ctx context.Context, userID uint) ([]models.Application, error)
	Update(ctx context.Context, application *models.Application) error
}

type ReviewRepository interface {
	Create(ctx context.Context, review *models.Review) error
	ListByUser(ctx context.Context, userID uint) ([]models.Review, error)
	GetTaskReview(ctx context.Context, taskID uint, reviewerID uint) (*models.Review, error)
}
