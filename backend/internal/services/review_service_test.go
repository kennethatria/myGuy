package services

import (
	"context"
	"errors"
	"testing"

	"myguy/internal/models"
	"myguy/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupReviewService() (*ReviewService, *tests.MockReviewRepository, *tests.MockTaskRepository, *tests.MockUserRepository) {
	reviewRepo := new(tests.MockReviewRepository)
	taskRepo := new(tests.MockTaskRepository)
	userRepo := new(tests.MockUserRepository)
	service := NewReviewService(reviewRepo, taskRepo, userRepo)
	return service, reviewRepo, taskRepo, userRepo
}

// ==================== CreateReview Tests ====================

func TestCreateReview(t *testing.T) {
	t.Run("successful review by task creator", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:         1,
			ReviewerID:     1, // Task creator
			ReviewedUserID: 2, // Assigned user
			Rating:         5,
			Comment:        "Great work!",
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(nil, errors.New("not found"))
		reviewRepo.On("Create", ctx, mock.MatchedBy(func(r *models.Review) bool {
			return r.TaskID == 1 &&
				r.ReviewerID == 1 &&
				r.ReviewedUserID == 2 &&
				r.Rating == 5
		})).Return(nil)

		review, err := service.CreateReview(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, 5, review.Rating)
		assert.Equal(t, "Great work!", review.Comment)
		reviewRepo.AssertExpectations(t)
		taskRepo.AssertExpectations(t)
	})

	t.Run("successful review by assignee", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:         1,
			ReviewerID:     2, // Assignee
			ReviewedUserID: 1, // Task creator
			Rating:         4,
			Comment:        "Good task",
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(2)).Return(nil, errors.New("not found"))
		reviewRepo.On("Create", ctx, mock.Anything).Return(nil)

		review, err := service.CreateReview(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, 4, review.Rating)
	})

	t.Run("invalid rating - too low", func(t *testing.T) {
		service, _, _, _ := setupReviewService()
		ctx := context.Background()

		input := CreateReviewInput{
			TaskID:     1,
			ReviewerID: 1,
			Rating:     0, // Invalid
		}

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRating, err)
		assert.Nil(t, review)
	})

	t.Run("invalid rating - too high", func(t *testing.T) {
		service, _, _, _ := setupReviewService()
		ctx := context.Background()

		input := CreateReviewInput{
			TaskID:     1,
			ReviewerID: 1,
			Rating:     6, // Invalid
		}

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidRating, err)
		assert.Nil(t, review)
	})

	t.Run("task not found", func(t *testing.T) {
		service, _, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		input := CreateReviewInput{
			TaskID:     999,
			ReviewerID: 1,
			Rating:     5,
		}

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, review)
	})

	t.Run("task not completed", func(t *testing.T) {
		service, _, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		openTask := &models.Task{
			ID:        1,
			CreatedBy: 1,
			Status:    "in_progress", // Not completed
		}

		input := CreateReviewInput{
			TaskID:     1,
			ReviewerID: 1,
			Rating:     5,
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(openTask, nil)

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotCompleted, err)
		assert.Nil(t, review)
	})

	t.Run("not a task participant", func(t *testing.T) {
		service, _, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:     1,
			ReviewerID: 3, // Neither creator nor assignee
			Rating:     5,
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrNotTaskParticipant, err)
		assert.Nil(t, review)
	})

	t.Run("already reviewed", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		existingReview := &models.Review{
			ID:         1,
			TaskID:     1,
			ReviewerID: 1,
		}

		input := CreateReviewInput{
			TaskID:     1,
			ReviewerID: 1,
			Rating:     5,
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(existingReview, nil)

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrAlreadyReviewed, err)
		assert.Nil(t, review)
	})

	t.Run("repository create error", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:         1,
			ReviewerID:     1,
			ReviewedUserID: 2,
			Rating:         5,
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(nil, errors.New("not found"))
		reviewRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		review, err := service.CreateReview(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, review)
	})

	t.Run("boundary rating - minimum valid (1)", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:         1,
			ReviewerID:     1,
			ReviewedUserID: 2,
			Rating:         1, // Minimum valid
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(nil, errors.New("not found"))
		reviewRepo.On("Create", ctx, mock.Anything).Return(nil)

		review, err := service.CreateReview(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, 1, review.Rating)
	})

	t.Run("boundary rating - maximum valid (5)", func(t *testing.T) {
		service, reviewRepo, taskRepo, _ := setupReviewService()
		ctx := context.Background()

		assignedTo := uint(2)
		completedTask := &models.Task{
			ID:         1,
			CreatedBy:  1,
			AssignedTo: &assignedTo,
			Status:     "completed",
		}

		input := CreateReviewInput{
			TaskID:         1,
			ReviewerID:     1,
			ReviewedUserID: 2,
			Rating:         5, // Maximum valid
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(completedTask, nil)
		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(nil, errors.New("not found"))
		reviewRepo.On("Create", ctx, mock.Anything).Return(nil)

		review, err := service.CreateReview(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, 5, review.Rating)
	})
}

// ==================== GetUserReviews Tests ====================

func TestGetUserReviews(t *testing.T) {
	t.Run("successful get reviews", func(t *testing.T) {
		service, reviewRepo, _, _ := setupReviewService()
		ctx := context.Background()

		expectedReviews := []models.Review{
			{ID: 1, ReviewedUserID: 1, Rating: 5},
			{ID: 2, ReviewedUserID: 1, Rating: 4},
		}

		reviewRepo.On("ListByUser", ctx, uint(1)).Return(expectedReviews, nil)

		reviews, err := service.GetUserReviews(ctx, 1)

		assert.NoError(t, err)
		assert.Len(t, reviews, 2)
		reviewRepo.AssertExpectations(t)
	})

	t.Run("user with no reviews", func(t *testing.T) {
		service, reviewRepo, _, _ := setupReviewService()
		ctx := context.Background()

		reviewRepo.On("ListByUser", ctx, uint(999)).Return([]models.Review{}, nil)

		reviews, err := service.GetUserReviews(ctx, 999)

		assert.NoError(t, err)
		assert.Len(t, reviews, 0)
	})

	t.Run("repository error", func(t *testing.T) {
		service, reviewRepo, _, _ := setupReviewService()
		ctx := context.Background()

		reviewRepo.On("ListByUser", ctx, uint(1)).Return([]models.Review{}, errors.New("database error"))

		reviews, err := service.GetUserReviews(ctx, 1)

		assert.Error(t, err)
		assert.Empty(t, reviews)
	})
}

// ==================== GetTaskReview Tests ====================

func TestGetTaskReview(t *testing.T) {
	t.Run("successful get review", func(t *testing.T) {
		service, reviewRepo, _, _ := setupReviewService()
		ctx := context.Background()

		expectedReview := &models.Review{
			ID:         1,
			TaskID:     1,
			ReviewerID: 1,
			Rating:     5,
		}

		reviewRepo.On("GetTaskReview", ctx, uint(1), uint(1)).Return(expectedReview, nil)

		review, err := service.GetTaskReview(ctx, 1, 1)

		assert.NoError(t, err)
		assert.NotNil(t, review)
		assert.Equal(t, 5, review.Rating)
	})

	t.Run("review not found", func(t *testing.T) {
		service, reviewRepo, _, _ := setupReviewService()
		ctx := context.Background()

		reviewRepo.On("GetTaskReview", ctx, uint(999), uint(1)).Return(nil, errors.New("not found"))

		review, err := service.GetTaskReview(ctx, 999, 1)

		assert.Error(t, err)
		assert.Nil(t, review)
	})
}
