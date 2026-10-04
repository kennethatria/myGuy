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
