package repositories

import (
	"context"
	"myguy/internal/models"
	"gorm.io/gorm"
)

type GormReviewRepository struct {
	db *gorm.DB
}

func NewGormReviewRepository(db *gorm.DB) *GormReviewRepository {
	return &GormReviewRepository{db: db}
}

func (r *GormReviewRepository) Create(ctx context.Context, review *models.Review) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the review
		if err := tx.Create(review).Error; err != nil {
			return err
		}

		// Update user's average rating
		var avgRating float64
		err := tx.Model(&models.Review{}).
			Select("COALESCE(AVG(rating), 0)").
			Where("reviewed_user_id = ?", review.ReviewedUserID).
			Scan(&avgRating).Error
		if err != nil {
			return err
		}

		return tx.Model(&models.User{}).
			Where("id = ?", review.ReviewedUserID).
			Update("average_rating", avgRating).Error
	})

	return err
}

func (r *GormReviewRepository) ListByUser(ctx context.Context, userID uint) ([]models.Review, error) {
	var reviews []models.Review
	err := r.db.WithContext(ctx).
		Where("reviewed_user_id = ?", userID).
		Order("created_at DESC").
		Find(&reviews).Error
	if err != nil {
		return nil, err
	}
	return reviews, nil
}

func (r *GormReviewRepository) GetTaskReview(ctx context.Context, taskID uint, reviewerID uint) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).
		Where("task_id = ? AND reviewer_id = ?", taskID, reviewerID).
		First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}
