package services

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

var (
	ErrInvalidEmail = errors.New("enter a valid email address")
	ErrInvalidDays  = errors.New("days must be 0 (for good) or more")
	// ErrBlocked: the address is blocked.  Asking for a code answers as if
	// one was sent, so a block isn't revealed to whoever is guessing.
	ErrBlocked = errors.New("this address is blocked")
	// ErrAccountUnavailable is what a blocked person sees.
	ErrAccountUnavailable = errors.New("this account isn't available")
)

// BlockService blocks and unblocks email addresses, and keeps the set of
// blocked accounts in memory for the session check on every request.
// Store-service and chat read that set from GET /internal/v1/blocked-users.
type BlockService struct {
	repo repositories.BlockedEmailRepository
	now  func() time.Time

	mu    sync.RWMutex
	users map[uint]bool
}

func NewBlockService(repo repositories.BlockedEmailRepository) *BlockService {
	return &BlockService{repo: repo, now: time.Now, users: map[uint]bool{}}
}

// Block blocks email for days (0: for good), replacing any block it had.
func (s *BlockService) Block(ctx context.Context, email, reason string, days int, by string) (*models.BlockedEmail, error) {
	email, err := validEmail(email)
	if err != nil {
		return nil, err
	}
	if days < 0 {
		return nil, ErrInvalidDays
	}
	block := &models.BlockedEmail{Email: email, Reason: reason, CreatedBy: by, CreatedAt: s.now().UTC()}
	if days > 0 {
		until := block.CreatedAt.Add(time.Duration(days) * 24 * time.Hour)
		block.Until = &until
	}
	if err := s.repo.Save(ctx, block); err != nil {
		return nil, err
	}
	return block, s.Refresh(ctx)
}

// Unblock lifts email's block, reporting whether it had one.
func (s *BlockService) Unblock(ctx context.Context, email string) (bool, error) {
	email, err := validEmail(email)
	if err != nil {
		return false, err
	}
	removed, err := s.repo.Delete(ctx, email)
	if err != nil {
		return false, err
	}
	return removed, s.Refresh(ctx)
}

// Status returns email's block, or nil if it has none (or it ran out).
func (s *BlockService) Status(ctx context.Context, email string) (*models.BlockedEmail, error) {
	email, err := validEmail(email)
	if err != nil {
		return nil, err
	}
	block, err := s.repo.Get(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !block.ActiveAt(s.now()) {
		return nil, nil
	}
	return block, nil
}

// List returns the blocks in force, newest first.
func (s *BlockService) List(ctx context.Context) ([]models.BlockedEmail, error) {
	return s.repo.ListActive(ctx, s.now())
}

// IsBlocked reports whether email is blocked now (sign-in and sign-up).
func (s *BlockService) IsBlocked(ctx context.Context, email string) (bool, error) {
	block, err := s.repo.Get(ctx, NormalizeEmail(email))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return block.ActiveAt(s.now()), nil
}

// UserBlocked reports whether an account is blocked, from memory.
func (s *BlockService) UserBlocked(id uint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id]
}

// BlockedUserIDs returns the blocked accounts, from memory, in id order.
func (s *BlockService) BlockedUserIDs() []uint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]uint, 0, len(s.users))
	for id := range s.users {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// Refresh reloads the blocked accounts (a block may also have run out).
func (s *BlockService) Refresh(ctx context.Context) error {
	ids, err := s.repo.ActiveUserIDs(ctx, s.now())
	if err != nil {
		return err
	}
	users := make(map[uint]bool, len(ids))
	for _, id := range ids {
		users[id] = true
	}
	s.mu.Lock()
	s.users = users
	s.mu.Unlock()
	return nil
}

// RefreshEvery keeps the blocked accounts current until ctx ends.  A failed
// refresh keeps the last set (blocks stay in force) and is logged.
func (s *BlockService) RefreshEvery(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.Refresh(ctx); err != nil {
			log.Printf("blocked accounts: refresh failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func validEmail(email string) (string, error) {
	email = NormalizeEmail(email)
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrInvalidEmail
	}
	return email, nil
}
