package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestUserRepository(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)

	repo := NewGormUserRepository(db)
	ctx := context.Background()

	t.Run("Create and GetByID", func(t *testing.T) {
		user := &models.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password",
			FullName: "Test User",
		}

		err := repo.Create(ctx, user)
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		fetched, err := repo.GetByID(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.Username, fetched.Username)
		assert.Equal(t, user.Email, fetched.Email)
	})

	t.Run("GetByEmail", func(t *testing.T) {
		user := &models.User{
			Username: "emailuser",
			Email:    "email@example.com",
			Password: "password",
		}
		repo.Create(ctx, user)

		fetched, err := repo.GetByEmail(ctx, user.Email)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)

		_, err = repo.GetByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
	})

	t.Run("GetByUsername", func(t *testing.T) {
		user := &models.User{
			Username: "usernameuser",
			Email:    "username@example.com",
			Password: "password",
		}
		repo.Create(ctx, user)

		fetched, err := repo.GetByUsername(ctx, user.Username)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)
	})

	t.Run("Update", func(t *testing.T) {
		user := &models.User{
			Username: "updateuser",
			Email:    "update@example.com",
			Password: "password",
			FullName: "Old Name",
		}
		repo.Create(ctx, user)

		user.FullName = "New Name"
		err := repo.Update(ctx, user)
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, user.ID)
		assert.Equal(t, "New Name", fetched.FullName)
	})

	t.Run("UpdateRating", func(t *testing.T) {
		user := &models.User{
			Username: "rateduser",
			Email:    "rated@example.com",
		}
		repo.Create(ctx, user)

		// Create unique tasks and reviewers to satisfy unique constraints
		db.Create(&models.Review{
			TaskID:         1,
			ReviewerID:     101,
			ReviewedUserID: user.ID,
			Rating:         4,
		})
		db.Create(&models.Review{
			TaskID:         2,
			ReviewerID:     102,
			ReviewedUserID: user.ID,
			Rating:         5,
		})

		err := repo.UpdateRating(ctx, user.ID)
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, user.ID)
		assert.Equal(t, 4.5, fetched.AverageRating)
	})
}
