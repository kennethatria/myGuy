package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

var (
	ErrInvalidRating     = errors.New("rating must be between 1 and 5")
	ErrTaskNotCompleted  = errors.New("cannot review an incomplete task")
	ErrAlreadyReviewed   = errors.New("you have already reviewed this task")
	ErrNotTaskParticipant = errors.New("you must be a participant in the task to leave a review")
)

type ReviewService struct {
	reviewRepo repositories.ReviewRepository
	taskRepo   repositories.TaskRepository
	userRepo   repositories.UserRepository
}

func NewReviewService(
	reviewRepo repositories.ReviewRepository,
	taskRepo repositories.TaskRepository,
	userRepo repositories.UserRepository,
) *ReviewService {
	return &ReviewService{
		reviewRepo: reviewRepo,
		taskRepo:   taskRepo,
		userRepo:   userRepo,
	}
}

type CreateReviewInput struct {
	TaskID         uint
	ReviewerID     uint
	ReviewedUserID uint
	Rating         int
	Comment        string
}

func (s *ReviewService) CreateReview(ctx context.Context, input CreateReviewInput) (*models.Review, error) {
	// Validate rating
	if input.Rating < 1 || input.Rating > 5 {
		return nil, ErrInvalidRating
	}

	// Get task to verify it's completed and the reviewer is a participant
	task, err := s.taskRepo.GetByID(ctx, input.TaskID)
	if err != nil {
		return nil, err
	}

	if task.Status != "completed" {
		return nil, ErrTaskNotCompleted
	}

	// Verify reviewer is either the task creator or assignee
	if task.CreatedBy != input.ReviewerID && *task.AssignedTo != input.ReviewerID {
		return nil, ErrNotTaskParticipant
	}

	// Check if reviewer has already reviewed this task
	existingReview, err := s.reviewRepo.GetTaskReview(ctx, input.TaskID, input.ReviewerID)
	if err == nil && existingReview != nil {
		return nil, ErrAlreadyReviewed
	}

	review := &models.Review{
		TaskID:         input.TaskID,
		ReviewerID:     input.ReviewerID,
		ReviewedUserID: input.ReviewedUserID,
		Rating:         input.Rating,
		Comment:        input.Comment,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

func (s *ReviewService) GetUserReviews(ctx context.Context, userID uint) ([]models.Review, error) {
	return s.reviewRepo.ListByUser(ctx, userID)
}

func (s *ReviewService) GetTaskReview(ctx context.Context, taskID, reviewerID uint) (*models.Review, error) {
	return s.reviewRepo.GetTaskReview(ctx, taskID, reviewerID)
}
