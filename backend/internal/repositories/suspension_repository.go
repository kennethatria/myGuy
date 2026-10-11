package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
)

type GormSuspensionRepository struct {
	db *gorm.DB
}

func NewGormSuspensionRepository(db *gorm.DB) *GormSuspensionRepository {
	return &GormSuspensionRepository{db: db}
}

func (r *GormSuspensionRepository) Create(ctx context.Context, s *models.AccountSuspension) error {
	return r.db.WithContext(ctx).Create(s).Error
}

// ForUser returns an account's suspensions, newest first.
func (r *GormSuspensionRepository) ForUser(ctx context.Context, userID uint) ([]models.AccountSuspension, error) {
	var out []models.AccountSuspension
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("starts_at DESC").Find(&out).Error
	return out, err
}

// Lift ends an account's suspension at now, reporting whether it had one.
func (r *GormSuspensionRepository) Lift(ctx context.Context, userID uint, now time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&models.AccountSuspension{}).
		Where("user_id = ?", userID).Where(activeSuspension, now).Update("lifted_at", now)
	return result.RowsAffected > 0, result.Error
}

// Active returns the suspensions in force at now.
func (r *GormSuspensionRepository) Active(ctx context.Context, now time.Time) ([]models.AccountSuspension, error) {
	var out []models.AccountSuspension
	err := r.db.WithContext(ctx).Where(activeSuspension, now).Order("user_id").Find(&out).Error
	return out, err
}

// suspendedUserIDs is a subquery for the accounts suspended at now (their
// posts are hidden, like blocked accounts').
func suspendedUserIDs(db *gorm.DB, now interface{}) *gorm.DB {
	return db.Session(&gorm.Session{NewDB: true}).Model(&models.AccountSuspension{}).
		Select("user_id").Where(activeSuspension, now)
}

const activeSuspension = "lifted_at IS NULL AND ends_at > ?"
