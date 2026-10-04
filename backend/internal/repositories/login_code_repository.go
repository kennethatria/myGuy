package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
)

type GormLoginCodeRepository struct {
	db *gorm.DB
}

func NewGormLoginCodeRepository(db *gorm.DB) *GormLoginCodeRepository {
	return &GormLoginCodeRepository{db: db}
}

func (r *GormLoginCodeRepository) Create(ctx context.Context, code *models.LoginCode) error {
	return r.db.WithContext(ctx).Create(code).Error
}

// LatestActive returns the newest unconsumed, unexpired code for email.
func (r *GormLoginCodeRepository) LatestActive(ctx context.Context, email string, now time.Time) (*models.LoginCode, error) {
	var code models.LoginCode
	err := r.db.WithContext(ctx).
		Where("email = ? AND consumed_at IS NULL AND expires_at > ?", email, now).
		Order("id DESC").
		First(&code).Error
	if err != nil {
		return nil, err
	}
	return &code, nil
}

func (r *GormLoginCodeRepository) IncrementAttempts(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.LoginCode{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

// MarkConsumed marks the code used, reporting false if it was already used,
// so the same code can never sign in twice.
func (r *GormLoginCodeRepository) MarkConsumed(ctx context.Context, id uint, at time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&models.LoginCode{}).
		Where("id = ? AND consumed_at IS NULL", id).
		UpdateColumn("consumed_at", at)
	return res.RowsAffected == 1, res.Error
}

// InvalidateActive consumes every outstanding code for email.
func (r *GormLoginCodeRepository) InvalidateActive(ctx context.Context, email string, at time.Time) error {
	return r.db.WithContext(ctx).Model(&models.LoginCode{}).
		Where("email = ? AND consumed_at IS NULL", email).
		UpdateColumn("consumed_at", at).Error
}

func (r *GormLoginCodeRepository) CountSince(ctx context.Context, email string, since time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.LoginCode{}).
		Where("email = ? AND created_at > ?", email, since).
		Count(&count).Error
	return count, err
}
