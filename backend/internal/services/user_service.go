package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailExists       = errors.New("email already exists")
	ErrUsernameExists    = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserService struct {
	userRepo repositories.UserRepository
}

func NewUserService(userRepo repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

type RegisterUserInput struct {
	Username string
	Email    string
	Password string
	FullName string
}

type UpdateUserInput struct {
	ID          uint
	FullName    string
	Email       string
	PhoneNumber string
	Bio         string
}

func (s *UserService) Register(ctx context.Context, input RegisterUserInput) (*models.UserResponse, error) {
	// Check if email exists
	if _, err := s.userRepo.GetByEmail(ctx, input.Email); err == nil {
		return nil, ErrEmailExists
	}

	// Check if username exists
	if _, err := s.userRepo.GetByUsername(ctx, input.Username); err == nil {
		return nil, ErrUsernameExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
		FullName: input.FullName,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &models.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
	}, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
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
