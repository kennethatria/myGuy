package services

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

const day = 24 * time.Hour

// SuspensionLadder is how long each suspension in a row lasts: 12 hours,
// then a day, two days, and a week from the fourth on.
var SuspensionLadder = []time.Duration{12 * time.Hour, day, 2 * day, 7 * day}

const (
	// A suspension counts towards the next one's length until the account
	// has gone this long without one.
	suspensionCleanAfter = 90 * day
	// The profile still says "suspended in <month>" this long after.
	suspensionMarkFor = 90 * day
)

var (
	ErrNoAccount        = errors.New("no account uses that address")
	ErrAlreadySuspended = errors.New("that account is already suspended; unsuspend it first")
	ErrSuspendBlocked   = errors.New("that address is blocked, which outranks a suspension")
	ErrInvalidLength    = errors.New("a suspension lasts 12 hours, 1, 2 or 7 days")
)

// Suspension is one account's suspension and where it sits on the ladder.
type Suspension struct {
	models.AccountSuspension
	Email string
	Step  int // 1 for the first in a row
}

// SuspensionStatus is what /status shows about an account.
type SuspensionStatus struct {
	Active *models.AccountSuspension
	InRow  int           // suspensions in a row so far (0: clean)
	Next   time.Duration // the length a bare /suspend would give now
}

// SuspensionMark is what a profile shows: suspended until, or when it last
// was (for 90 days after).
type SuspensionMark struct {
	Until *time.Time `json:"until,omitempty"`
	Ended *time.Time `json:"ended,omitempty"`
}

// SuspendedUser is an account suspended now, for store-service and chat.
type SuspendedUser struct {
	UserID uint      `json:"user_id"`
	Until  time.Time `json:"until"`
}

// SuspensionService suspends and unsuspends accounts, and keeps the ones
// suspended now in memory for the check on every request.  Store-service
// and chat read them with the blocks from GET /internal/v1/blocked-users.
type SuspensionService struct {
	repo   repositories.SuspensionRepository
	users  repositories.UserRepository
	blocks EmailBlocks
	now    func() time.Time

	mu        sync.RWMutex
	suspended map[uint]time.Time
}

func NewSuspensionService(repo repositories.SuspensionRepository, users repositories.UserRepository, blocks EmailBlocks) *SuspensionService {
	return &SuspensionService{repo: repo, users: users, blocks: blocks, now: time.Now, suspended: map[uint]time.Time{}}
}

// Suspend suspends email's account for length, or (0) the next step on the
// ladder.
func (s *SuspensionService) Suspend(ctx context.Context, email string, length time.Duration, reason, by string) (*Suspension, error) {
	if length != 0 && !validLength(length) {
		return nil, ErrInvalidLength
	}
	user, err := s.account(ctx, email)
	if err != nil {
		return nil, err
	}
	if blocked, err := s.blocks.IsBlocked(ctx, user.Email); err != nil {
		return nil, err
	} else if blocked {
		return nil, ErrSuspendBlocked
	}
	history, err := s.repo.ForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()
	if len(history) > 0 && history[0].ActiveAt(now) {
		return nil, ErrAlreadySuspended
	}
	step := inRow(history, now) + 1
	if length == 0 {
		length = ladderStep(step)
	}
	row := models.AccountSuspension{UserID: user.ID, Reason: reason, StartsAt: now, EndsAt: now.Add(length), CreatedBy: by}
	if err := s.repo.Create(ctx, &row); err != nil {
		return nil, err
	}
	return &Suspension{AccountSuspension: row, Email: NormalizeEmail(user.Email), Step: step}, s.Refresh(ctx)
}

// Unsuspend lifts email's suspension now, reporting whether it had one.  It
// still counts towards the next one's length.
func (s *SuspensionService) Unsuspend(ctx context.Context, email string) (bool, error) {
	user, err := s.account(ctx, email)
	if err != nil {
		return false, err
	}
	lifted, err := s.repo.Lift(ctx, user.ID, s.now().UTC())
	if err != nil {
		return false, err
	}
	return lifted, s.Refresh(ctx)
}

// Status returns email's suspension, if any, and its place on the ladder.
func (s *SuspensionService) Status(ctx context.Context, email string) (*SuspensionStatus, error) {
	user, err := s.account(ctx, email)
	if err != nil {
		return nil, err
	}
	history, err := s.repo.ForUser(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	now := s.now()
	status := &SuspensionStatus{InRow: inRow(history, now)}
	status.Next = ladderStep(status.InRow + 1)
	if len(history) > 0 && history[0].ActiveAt(now) {
		status.Active = &history[0]
	}
	return status, nil
}

// Mark is what userID's profile shows, or nil.
func (s *SuspensionService) Mark(ctx context.Context, userID uint) (*SuspensionMark, error) {
	history, err := s.repo.ForUser(ctx, userID)
	if err != nil || len(history) == 0 {
		return nil, err
	}
	now, last := s.now(), history[0]
	end := last.End()
	if last.ActiveAt(now) {
		return &SuspensionMark{Until: &end}, nil
	}
	if now.Sub(end) <= suspensionMarkFor {
		return &SuspensionMark{Ended: &end}, nil
	}
	return nil, nil
}

// SuspendedUntil reports whether an account is suspended, and until when,
// from memory.
func (s *SuspensionService) SuspendedUntil(id uint) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	until, ok := s.suspended[id]
	if ok && !until.After(s.now()) {
		return time.Time{}, false // ran out since the last refresh
	}
	return until, ok
}

// SuspendedUsers returns the accounts suspended now, from memory.
func (s *SuspensionService) SuspendedUsers() []SuspendedUser {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := s.now()
	out := make([]SuspendedUser, 0, len(s.suspended))
	for id, until := range s.suspended {
		if until.After(now) {
			out = append(out, SuspendedUser{UserID: id, Until: until})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UserID < out[j].UserID })
	return out
}

// Refresh reloads the accounts suspended now.
func (s *SuspensionService) Refresh(ctx context.Context) error {
	active, err := s.repo.Active(ctx, s.now())
	if err != nil {
		return err
	}
	suspended := make(map[uint]time.Time, len(active))
	for _, a := range active {
		suspended[a.UserID] = a.End().UTC()
	}
	s.mu.Lock()
	s.suspended = suspended
	s.mu.Unlock()
	return nil
}

// RefreshEvery keeps them current until ctx ends.
func (s *SuspensionService) RefreshEvery(ctx context.Context, interval time.Duration) {
	refreshEvery(ctx, interval, "suspended accounts", s.Refresh)
}

func (s *SuspensionService) account(ctx context.Context, email string) (*models.User, error) {
	email, err := validEmail(email)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNoAccount
	}
	return user, err
}

// inRow counts the suspensions in a row: each began within 90 days of the
// end of the one after it, and the newest ended within 90 days of now.
func inRow(history []models.AccountSuspension, now time.Time) int {
	n, since := 0, now
	for _, h := range history {
		if since.Sub(h.End()) > suspensionCleanAfter {
			break
		}
		n, since = n+1, h.StartsAt
	}
	return n
}

func ladderStep(step int) time.Duration {
	if step > len(SuspensionLadder) {
		step = len(SuspensionLadder)
	}
	return SuspensionLadder[step-1]
}

func validLength(length time.Duration) bool {
	for _, l := range SuspensionLadder {
		if l == length {
			return true
		}
	}
	return false
}
