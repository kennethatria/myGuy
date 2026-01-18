package tests

import (
	"context"

	"myguy/internal/models"

	"github.com/stretchr/testify/mock"
)

// MockTaskRepository
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id uint) (*models.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Task), args.Error(1)
}

func (m *MockTaskRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]models.Task), args.Error(1)
}

func (m *MockTaskRepository) ListWithPagination(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]models.Task), args.Error(1)
}

func (m *MockTaskRepository) Count(ctx context.Context, filters map[string]interface{}) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockTaskRepository) Update(ctx context.Context, task *models.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTaskRepository) ListByUser(ctx context.Context, userID uint, role string) ([]models.Task, error) {
	args := m.Called(ctx, userID, role)
	return args.Get(0).([]models.Task), args.Error(1)
}

// MockApplicationRepository
type MockApplicationRepository struct {
	mock.Mock
}

func (m *MockApplicationRepository) Create(ctx context.Context, application *models.Application) error {
	args := m.Called(ctx, application)
	return args.Error(0)
}

func (m *MockApplicationRepository) GetByID(ctx context.Context, id uint) (*models.Application, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Application), args.Error(1)
}

func (m *MockApplicationRepository) ListByTask(ctx context.Context, taskID uint) ([]models.Application, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).([]models.Application), args.Error(1)
}

func (m *MockApplicationRepository) ListByUser(ctx context.Context, userID uint) ([]models.Application, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Application), args.Error(1)
}

func (m *MockApplicationRepository) Update(ctx context.Context, application *models.Application) error {
	args := m.Called(ctx, application)
	return args.Error(0)
}

// MockReviewRepository
type MockReviewRepository struct {
	mock.Mock
}

func (m *MockReviewRepository) Create(ctx context.Context, review *models.Review) error {
	args := m.Called(ctx, review)
	return args.Error(0)
}

func (m *MockReviewRepository) ListByUser(ctx context.Context, userID uint) ([]models.Review, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]models.Review), args.Error(1)
}

func (m *MockReviewRepository) GetTaskReview(ctx context.Context, taskID uint, reviewerID uint) (*models.Review, error) {
	args := m.Called(ctx, taskID, reviewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Review), args.Error(1)
}
