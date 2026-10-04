package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"myguy/internal/models"
	"myguy/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserService() (*UserService, *tests.MockUserRepository) {
	userRepo := new(tests.MockUserRepository)
	service := NewUserService(userRepo)
	return service, userRepo
}

// ==================== GetProfile Tests ====================

func TestGetProfile(t *testing.T) {
	t.Run("successful get profile", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:            1,
			Username:      "testuser",
			Email:         "test@example.com",
			FullName:      "Test User",
			Bio:           "Developer",
			AverageRating: 4.8,
			CreatedAt:     time.Now(),
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)

		result, err := service.GetProfile(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existingUser.ID, result.ID)
		assert.Equal(t, existingUser.Username, result.Username)
		assert.Equal(t, existingUser.Bio, result.Bio)
		userRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		userRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.GetProfile(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})
}

// ==================== UpdateProfile Tests ====================

func TestUpdateProfile(t *testing.T) {
	t.Run("successful update profile", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			FullName: "Old Name",
			Bio:      "Old bio",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("Update", ctx, mock.MatchedBy(func(user *models.User) bool {
			return user.FullName == "New Name" && user.Bio == "New bio"
		})).Return(nil)

		result, err := service.UpdateProfile(ctx, 1, "New Name", "New bio")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Name", result.FullName)
		assert.Equal(t, "New bio", result.Bio)
		userRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		userRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.UpdateProfile(ctx, 999, "Name", "Bio")

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})

	t.Run("repository update error", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:       1,
			Username: "testuser",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("Update", ctx, mock.Anything).Return(errors.New("database error"))

		result, err := service.UpdateProfile(ctx, 1, "Name", "Bio")

		assert.Error(t, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})
}

// ==================== GetUser Tests ====================

func TestGetUser(t *testing.T) {
	t.Run("successful get user", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:          1,
			Username:    "testuser",
			Email:       "test@example.com",
			FullName:    "Test User",
			PhoneNumber: "1234567890",
			Bio:         "Developer",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)

		result, err := service.GetUser(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, existingUser.ID, result.ID)
		assert.Equal(t, existingUser.PhoneNumber, result.PhoneNumber)
		userRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		userRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.GetUser(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})
}
