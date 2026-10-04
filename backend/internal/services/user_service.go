package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailExists       = errors.New("email already exists")
	ErrUsernameExists    = errors.New("username already exists")
)

type UserService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

type UpdateUserInput struct {
	ID          uint
	FullName    string
	Email       string
	PhoneNumber string
	Bio         string
}

func (s *UserService) GetProfile(ctx context.Context, userID uint) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &models.UserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		FullName:     user.FullName,
		Bio:          user.Bio,
		AverageRating: user.AverageRating,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uint, fullName, bio string) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	user.FullName = fullName
	user.Bio = bio

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:           user.ID,
		Username:     user.Username,
		Email:        user.Email,
		FullName:     user.FullName,
		Bio:          user.Bio,
		AverageRating: user.AverageRating,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *UserService) GetUser(ctx context.Context, id uint) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    user.FullName,
		PhoneNumber: user.PhoneNumber,
		Bio:        user.Bio,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}

func (s *UserService) UpdateUser(ctx context.Context, input UpdateUserInput) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if input.FullName != "" {
		user.FullName = input.FullName
	}
	if input.Email != "" {
		// Check if email is taken by another user
		if existingUser, err := s.userRepo.GetByEmail(ctx, input.Email); err == nil && existingUser.ID != input.ID {
			return nil, ErrEmailExists
		}
		user.Email = input.Email
	}
	if input.PhoneNumber != "" {
		user.PhoneNumber = input.PhoneNumber
	}
	if input.Bio != "" {
		user.Bio = input.Bio
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    user.FullName,
		PhoneNumber: user.PhoneNumber,
		Bio:        user.Bio,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}, nil
}
