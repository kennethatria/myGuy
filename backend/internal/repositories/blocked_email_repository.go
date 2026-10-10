package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"myguy/internal/models"
)

type GormBlockedEmailRepository struct {
	db *gorm.DB
}

func NewGormBlockedEmailRepository(db *gorm.DB) *GormBlockedEmailRepository {
	return &GormBlockedEmailRepository{db: db}
}

// Save blocks an address, or replaces its block (reason, until, by).
func (r *GormBlockedEmailRepository) Save(ctx context.Context, block *models.BlockedEmail) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"reason", "blocked_until", "created_by", "created_at"}),
	}).Create(block).Error
}

// Delete lifts an address's block, reporting whether there was one.
func (r *GormBlockedEmailRepository) Delete(ctx context.Context, email string) (bool, error) {
	result := r.db.WithContext(ctx).Where("email = ?", email).Delete(&models.BlockedEmail{})
	return result.RowsAffected > 0, result.Error
}

func (r *GormBlockedEmailRepository) Get(ctx context.Context, email string) (*models.BlockedEmail, error) {
	var block models.BlockedEmail
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&block).Error; err != nil {
		return nil, err
	}
	return &block, nil
}

// ListActive returns the blocks that apply at now, newest first.
func (r *GormBlockedEmailRepository) ListActive(ctx context.Context, now time.Time) ([]models.BlockedEmail, error) {
	var blocks []models.BlockedEmail
	err := r.db.WithContext(ctx).Where(activeBlock, now).Order("created_at DESC").Find(&blocks).Error
	return blocks, err
}

// ActiveUserIDs returns the accounts whose address is blocked at now.
func (r *GormBlockedEmailRepository) ActiveUserIDs(ctx context.Context, now time.Time) ([]uint, error) {
	var ids []uint
	err := blockedUserIDs(r.db, now).WithContext(ctx).Order("users.id").Pluck("users.id", &ids).Error
	return ids, err
}

// blockedUserIDs is a query for the ids of accounts blocked at now (also
// used as a subquery to hide their posts).
func blockedUserIDs(db *gorm.DB, now interface{}) *gorm.DB {
	return db.Session(&gorm.Session{NewDB: true}).Model(&models.User{}).Select("users.id").
		Where("LOWER(users.email) IN (?)",
			db.Session(&gorm.Session{NewDB: true}).Model(&models.BlockedEmail{}).Select("email").Where(activeBlock, now))
}

const activeBlock = "blocked_until IS NULL OR blocked_until > ?"
