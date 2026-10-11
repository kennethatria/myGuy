package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

func setupSuspensions(t *testing.T) (*SuspensionService, *BlockService, *models.User, *time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.BlockedEmail{}, &models.AccountSuspension{}); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	users := repositories.NewGormUserRepository(db)
	blocks := NewBlockService(repositories.NewGormBlockedEmailRepository(db))
	blocks.now = now
	s := NewSuspensionService(repositories.NewGormSuspensionRepository(db), users, blocks)
	s.now = now
	jane := &models.User{Username: "jane", Email: "Jane@Example.com", FullName: "Jane"}
	assert.NoError(t, users.Create(context.Background(), jane))
	return s, blocks, jane, &clock
}

func TestSuspensionLadder(t *testing.T) {
	ctx := context.Background()
	s, _, jane, clock := setupSuspensions(t)

	// Each suspension in a row is longer: 12 h, 1 d, 2 d, then a week
	for _, want := range []time.Duration{12 * time.Hour, day, 2 * day, 7 * day, 7 * day} {
		got, err := s.Suspend(ctx, "jane@example.com", 0, "spam", "telegram")
		assert.NoError(t, err)
		assert.Equal(t, want, got.EndsAt.Sub(got.StartsAt))
		assert.Equal(t, jane.ID, got.UserID)
		*clock = got.EndsAt.Add(30 * day) // back within 90 days
	}

	// 90 clean days: back to 12 hours
	*clock = clock.Add(61 * day)
	status, err := s.Status(ctx, "jane@example.com")
	assert.NoError(t, err)
	assert.Equal(t, 0, status.InRow)
	assert.Equal(t, 12*time.Hour, status.Next)
}

func TestSuspensionLengthCanBeChosen(t *testing.T) {
	ctx := context.Background()
	s, _, _, _ := setupSuspensions(t)
	got, err := s.Suspend(ctx, "jane@example.com", 7*day, "", "ssh:ops")
	assert.NoError(t, err)
	assert.Equal(t, 7*day, got.EndsAt.Sub(got.StartsAt))
	assert.Equal(t, 1, got.Step)

	_, err = s.Suspend(ctx, "jane@example.com", 5*time.Hour, "", "")
	assert.ErrorIs(t, err, ErrInvalidLength)
}

func TestSuspensionIsReadFromMemoryAndRunsOut(t *testing.T) {
	ctx := context.Background()
	s, _, jane, clock := setupSuspensions(t)
	_, ok := s.SuspendedUntil(jane.ID)
	assert.False(t, ok)

	got, err := s.Suspend(ctx, "jane@example.com", 0, "", "")
	assert.NoError(t, err)
	until, ok := s.SuspendedUntil(jane.ID)
	assert.True(t, ok)
	assert.Equal(t, got.EndsAt, until)
	assert.Equal(t, []SuspendedUser{{UserID: jane.ID, Until: got.EndsAt}}, s.SuspendedUsers())

	// Over before the next refresh: not suspended any more
	*clock = got.EndsAt
	_, ok = s.SuspendedUntil(jane.ID)
	assert.False(t, ok)
	assert.Empty(t, s.SuspendedUsers())
}

func TestUnsuspendStillCountsTowardsTheNext(t *testing.T) {
	ctx := context.Background()
	s, _, jane, clock := setupSuspensions(t)
	_, err := s.Suspend(ctx, "jane@example.com", 0, "", "")
	assert.NoError(t, err)
	_, err = s.Suspend(ctx, "jane@example.com", 0, "", "")
	assert.ErrorIs(t, err, ErrAlreadySuspended)

	lifted, err := s.Unsuspend(ctx, "jane@example.com")
	assert.NoError(t, err)
	assert.True(t, lifted)
	_, ok := s.SuspendedUntil(jane.ID)
	assert.False(t, ok)
	lifted, err = s.Unsuspend(ctx, "jane@example.com")
	assert.NoError(t, err)
	assert.False(t, lifted)

	*clock = clock.Add(time.Hour)
	got, err := s.Suspend(ctx, "jane@example.com", 0, "", "")
	assert.NoError(t, err)
	assert.Equal(t, 2, got.Step)
	assert.Equal(t, day, got.EndsAt.Sub(got.StartsAt))
}

func TestSuspensionMark(t *testing.T) {
	ctx := context.Background()
	s, _, jane, clock := setupSuspensions(t)
	mark, err := s.Mark(ctx, jane.ID)
	assert.NoError(t, err)
	assert.Nil(t, mark)

	got, _ := s.Suspend(ctx, "jane@example.com", 0, "", "")
	mark, _ = s.Mark(ctx, jane.ID)
	assert.Equal(t, got.EndsAt, *mark.Until)
	assert.Nil(t, mark.Ended)

	*clock = got.EndsAt.Add(90 * day)
	mark, _ = s.Mark(ctx, jane.ID)
	assert.Nil(t, mark.Until)
	assert.Equal(t, got.EndsAt, *mark.Ended, "shown for 90 days after")

	*clock = clock.Add(time.Minute)
	mark, _ = s.Mark(ctx, jane.ID)
	assert.Nil(t, mark)
}

func TestSuspensionRefusals(t *testing.T) {
	ctx := context.Background()
	s, blocks, _, _ := setupSuspensions(t)
	_, err := s.Suspend(ctx, "nobody@example.com", 0, "", "")
	assert.ErrorIs(t, err, ErrNoAccount)
	_, err = s.Suspend(ctx, "not an email", 0, "", "")
	assert.ErrorIs(t, err, ErrInvalidEmail)
	_, err = s.Unsuspend(ctx, "nobody@example.com")
	assert.ErrorIs(t, err, ErrNoAccount)

	_, err = blocks.Block(ctx, "jane@example.com", "", 0, "")
	assert.NoError(t, err)
	_, err = s.Suspend(ctx, "jane@example.com", 0, "", "")
	assert.ErrorIs(t, err, ErrSuspendBlocked)
}
