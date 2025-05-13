package repositories

import (
	"context"
	"gorm.io/gorm"
	"myguy/internal/models"
)

type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *GormUserRepository) UpdateRating(ctx context.Context, userID uint) error {
	// Calculate average rating
	var avgRating float64
	err := r.db.WithContext(ctx).Model(&models.Review{}).
		Select("COALESCE(AVG(rating), 0)").
		Where("reviewed_user_id = ?", userID).
		Scan(&avgRating).Error
	if err != nil {
		return err
	}

	// Update user's average rating
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("average_rating", avgRating).Error
}
