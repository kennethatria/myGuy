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

// ==================== UpdateUser Tests ====================

func TestUpdateUser(t *testing.T) {
	t.Run("successful full update", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:          1,
			Username:    "testuser",
			Email:       "old@example.com",
			FullName:    "Old Name",
			PhoneNumber: "0000000000",
			Bio:         "Old bio",
		}

		input := UpdateUserInput{
			ID:          1,
			FullName:    "New Name",
			Email:       "new@example.com",
			PhoneNumber: "1234567890",
			Bio:         "New bio",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("GetByEmail", ctx, "new@example.com").Return(nil, errors.New("not found"))
		userRepo.On("Update", ctx, mock.MatchedBy(func(user *models.User) bool {
			return user.FullName == "New Name" &&
				user.Email == "new@example.com" &&
				user.PhoneNumber == "1234567890" &&
				user.Bio == "New bio"
		})).Return(nil)

		result, err := service.UpdateUser(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Name", result.FullName)
		assert.Equal(t, "new@example.com", result.Email)
		assert.Equal(t, "1234567890", result.PhoneNumber)
		assert.Equal(t, "New bio", result.Bio)
		userRepo.AssertExpectations(t)
	})

	t.Run("partial update - only name", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@example.com",
			FullName: "Old Name",
		}

		input := UpdateUserInput{
			ID:       1,
			FullName: "New Name",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("Update", ctx, mock.MatchedBy(func(user *models.User) bool {
			return user.FullName == "New Name" && user.Email == "test@example.com"
		})).Return(nil)

		result, err := service.UpdateUser(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "New Name", result.FullName)
		userRepo.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		input := UpdateUserInput{
			ID:       999,
			FullName: "Name",
		}

		userRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.UpdateUser(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})

	t.Run("email taken by another user", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:    1,
			Email: "user1@example.com",
		}

		otherUser := &models.User{
			ID:    2,
			Email: "taken@example.com",
		}

		input := UpdateUserInput{
			ID:    1,
			Email: "taken@example.com",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("GetByEmail", ctx, "taken@example.com").Return(otherUser, nil)

		result, err := service.UpdateUser(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrEmailExists, err)
		assert.Nil(t, result)
		userRepo.AssertExpectations(t)
	})

	t.Run("same user updating to own email - allowed", func(t *testing.T) {
		service, userRepo := setupUserService()
		ctx := context.Background()

		existingUser := &models.User{
			ID:       1,
			Email:    "user@example.com",
			FullName: "Old Name",
		}

		input := UpdateUserInput{
			ID:       1,
			Email:    "user@example.com", // Same email
			FullName: "New Name",
		}

		userRepo.On("GetByID", ctx, uint(1)).Return(existingUser, nil)
		userRepo.On("GetByEmail", ctx, "user@example.com").Return(existingUser, nil) // Same user
		userRepo.On("Update", ctx, mock.Anything).Return(nil)

		result, err := service.UpdateUser(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		userRepo.AssertExpectations(t)
	})
}
