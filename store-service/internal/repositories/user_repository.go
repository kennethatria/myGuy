package repositories

import (
	"store-service/internal/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.User) error
	GetByID(id uint) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	Update(user *models.User) error
	UpsertFromJWT(userID uint, username, email, name string) (*models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) UpsertFromJWT(userID uint, username, email, name string) (*models.User, error) {
	user := &models.User{
		ID:       userID,
		Username: username,
		Email:    email,
		Name:     name,
	}

	// Try to find existing user
	var existingUser models.User
	if err := r.db.First(&existingUser, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new user
			if err := r.db.Create(user).Error; err != nil {
				return nil, err
			}
			return user, nil
		}
		return nil, err
	}

	// Update existing user
	existingUser.Username = username
	existingUser.Email = email
	existingUser.Name = name
	if err := r.db.Save(&existingUser).Error; err != nil {
		return nil, err
	}

	return &existingUser, nil
}