package repositories

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"myguy/internal/models"
)

type GormCodePauseRepository struct {
	db *gorm.DB
}

func NewGormCodePauseRepository(db *gorm.DB) *GormCodePauseRepository {
	return &GormCodePauseRepository{db: db}
}

// Save pauses an address, or replaces its pause.
func (r *GormCodePauseRepository) Save(ctx context.Context, p *models.CodePause) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"paused_until", "created_by", "created_at"}),
	}).Create(p).Error
}

// Delete lifts an address's pause, reporting whether there was one.
func (r *GormCodePauseRepository) Delete(ctx context.Context, email string) (bool, error) {
	result := r.db.WithContext(ctx).Where("email = ?", email).Delete(&models.CodePause{})
	return result.RowsAffected > 0, result.Error
}

func (r *GormCodePauseRepository) Get(ctx context.Context, email string) (*models.CodePause, error) {
	var p models.CodePause
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}
