package services

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

// MaxCodePause is the longest an address's codes can be paused.
const MaxCodePause = 72 * time.Hour

var ErrInvalidPause = errors.New("a pause lasts 1 to 72 hours")

// CodePauseService pauses sign-in codes for an address (email bombing).
type CodePauseService struct {
	repo repositories.CodePauseRepository
	now  func() time.Time
}

func NewCodePauseService(repo repositories.CodePauseRepository) *CodePauseService {
	return &CodePauseService{repo: repo, now: time.Now}
}

// Pause stops codes to email for length, replacing any pause it had.
func (s *CodePauseService) Pause(ctx context.Context, email string, length time.Duration, by string) (*models.CodePause, error) {
	email, err := validEmail(email)
	if err != nil {
		return nil, err
	}
	if length < time.Hour || length > MaxCodePause {
		return nil, ErrInvalidPause
	}
	now := s.now().UTC()
	p := &models.CodePause{Email: email, Until: now.Add(length), CreatedBy: by, CreatedAt: now}
	return p, s.repo.Save(ctx, p)
}

// Unpause lets codes go to email again, reporting whether it was paused.
func (s *CodePauseService) Unpause(ctx context.Context, email string) (bool, error) {
	email, err := validEmail(email)
	if err != nil {
		return false, err
	}
	return s.repo.Delete(ctx, email)
}

// Status returns email's pause in force, or nil.
func (s *CodePauseService) Status(ctx context.Context, email string) (*models.CodePause, error) {
	p, err := s.repo.Get(ctx, NormalizeEmail(email))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !p.Until.After(s.now()) {
		return nil, nil
	}
	return p, nil
}

// IsPaused reports whether codes to email are paused now (sign-in).
func (s *CodePauseService) IsPaused(ctx context.Context, email string) (bool, error) {
	p, err := s.Status(ctx, email)
	return p != nil, err
}
