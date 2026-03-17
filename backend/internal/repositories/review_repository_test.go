package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestReviewRepository(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)

	repo := NewGormReviewRepository(db)
	ctx := context.Background()

	// Create users and a task
	reviewer := &models.User{Username: "reviewer", Email: "reviewer@example.com"}
	db.Create(reviewer)
	reviewed := &models.User{Username: "reviewed", Email: "reviewed@example.com"}
	db.Create(reviewed)
	task := &models.Task{Title: "Task for Review", CreatedBy: reviewer.ID}
	db.Create(task)

	t.Run("Create and GetTaskReview", func(t *testing.T) {
		review := &models.Review{
			TaskID:         task.ID,
			ReviewerID:     reviewer.ID,
			ReviewedUserID: reviewed.ID,
			Rating:         5,
			Comment:        "Great job!",
		}

		err := repo.Create(ctx, review)
		assert.NoError(t, err)
		assert.NotZero(t, review.ID)

		fetched, err := repo.GetTaskReview(ctx, task.ID, reviewer.ID)
		assert.NoError(t, err)
		assert.Equal(t, review.Comment, fetched.Comment)
		assert.Equal(t, reviewed.ID, fetched.ReviewedUser.ID)

		// Check if user rating was updated
		var user models.User
		db.First(&user, reviewed.ID)
		assert.Equal(t, 5.0, user.AverageRating)
	})

	t.Run("ListByUser", func(t *testing.T) {
		reviews, err := repo.ListByUser(ctx, reviewed.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, reviews)
		assert.Equal(t, reviewed.ID, reviews[0].ReviewedUserID)
	})
}
