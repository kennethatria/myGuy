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

func setupBlocks(t *testing.T) (*BlockService, *AuthService, *fakeSender, *repositories.GormUserRepository, *time.Time) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.LoginCode{}, &models.BlockedEmail{}); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return clock }
	blocks := NewBlockService(repositories.NewGormBlockedEmailRepository(db))
	blocks.now = now
	userRepo := repositories.NewGormUserRepository(db)
	sender := &fakeSender{codes: map[string]string{}}
	auth := NewAuthService(userRepo, repositories.NewGormLoginCodeRepository(db), sender, "test-secret").WithBlocks(blocks)
	auth.now = now
	return blocks, auth, sender, userRepo, &clock
}

func TestBlockService(t *testing.T) {
	ctx := context.Background()

	t.Run("blocks an account by its address, case-insensitively", func(t *testing.T) {
		blocks, _, _, users, _ := setupBlocks(t)
		jane := &models.User{Username: "jane", Email: "Jane@Example.com", FullName: "Jane"}
		assert.NoError(t, users.Create(ctx, jane))

		block, err := blocks.Block(ctx, " JANE@example.com ", "spam", 0, "ops")
		assert.NoError(t, err)
		assert.Equal(t, "jane@example.com", block.Email)
		assert.Nil(t, block.Until, "0 days is for good")
		assert.True(t, blocks.UserBlocked(jane.ID))
		assert.Equal(t, []uint{jane.ID}, blocks.BlockedUserIDs())

		removed, err := blocks.Unblock(ctx, "jane@example.com")
		assert.NoError(t, err)
		assert.True(t, removed)
		assert.False(t, blocks.UserBlocked(jane.ID))

		removed, err = blocks.Unblock(ctx, "jane@example.com")
		assert.NoError(t, err)
		assert.False(t, removed)
	})

	t.Run("a block for some days runs out", func(t *testing.T) {
		blocks, _, _, users, clock := setupBlocks(t)
		jane := &models.User{Username: "jane", Email: "jane@example.com", FullName: "Jane"}
		assert.NoError(t, users.Create(ctx, jane))

		block, err := blocks.Block(ctx, "jane@example.com", "", 7, "ops")
		assert.NoError(t, err)
		assert.Equal(t, clock.Add(7*24*time.Hour), *block.Until)

		*clock = clock.Add(8 * 24 * time.Hour)
		assert.NoError(t, blocks.Refresh(ctx))
		assert.False(t, blocks.UserBlocked(jane.ID))
		status, err := blocks.Status(ctx, "jane@example.com")
		assert.NoError(t, err)
		assert.Nil(t, status)
		list, err := blocks.List(ctx)
		assert.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("blocking again replaces the block", func(t *testing.T) {
		blocks, _, _, _, _ := setupBlocks(t)
		_, err := blocks.Block(ctx, "x@example.com", "first", 1, "ops")
		assert.NoError(t, err)
		_, err = blocks.Block(ctx, "x@example.com", "second", 0, "ops")
		assert.NoError(t, err)
		status, err := blocks.Status(ctx, "x@example.com")
		assert.NoError(t, err)
		assert.Equal(t, "second", status.Reason)
		assert.Nil(t, status.Until)
	})

	t.Run("refuses bad input", func(t *testing.T) {
		blocks, _, _, _, _ := setupBlocks(t)
		for _, email := range []string{"", "not-an-email", "Jane <jane@example.com>", "a@b.com, c@d.com"} {
			_, err := blocks.Block(ctx, email, "", 0, "ops")
			assert.ErrorIs(t, err, ErrInvalidEmail, email)
		}
		_, err := blocks.Block(ctx, "jane@example.com", "", -1, "ops")
		assert.ErrorIs(t, err, ErrInvalidDays)
	})
}

func TestAuthService_Blocked(t *testing.T) {
	ctx := context.Background()

	t.Run("no code is sent to a blocked address", func(t *testing.T) {
		blocks, auth, sender, _, _ := setupBlocks(t)
		_, err := blocks.Block(ctx, "jane@example.com", "", 0, "ops")
		assert.NoError(t, err)
		assert.ErrorIs(t, auth.RequestCode(ctx, "Jane@example.com"), ErrBlocked)
		assert.Empty(t, sender.codes)
	})

	t.Run("a code sent before the block no longer signs in", func(t *testing.T) {
		blocks, auth, sender, users, _ := setupBlocks(t)
		assert.NoError(t, users.Create(ctx, &models.User{Username: "jane", Email: "jane@example.com", FullName: "Jane"}))
		assert.NoError(t, auth.RequestCode(ctx, "jane@example.com"))
		_, err := blocks.Block(ctx, "jane@example.com", "", 0, "ops")
		assert.NoError(t, err)
		_, err = auth.VerifyCode(ctx, "jane@example.com", sender.codes["jane@example.com"])
		assert.ErrorIs(t, err, ErrAccountUnavailable)
	})

	t.Run("a blocked address can't sign up", func(t *testing.T) {
		blocks, auth, _, _, _ := setupBlocks(t)
		_, err := blocks.Block(ctx, "new@example.com", "", 0, "ops")
		assert.NoError(t, err)
		_, err = auth.CompleteSignup(ctx, "new@example.com", "New Person")
		assert.ErrorIs(t, err, ErrAccountUnavailable)
	})

	t.Run("unblocked, sign-in works again", func(t *testing.T) {
		blocks, auth, sender, _, _ := setupBlocks(t)
		_, err := blocks.Block(ctx, "jane@example.com", "", 0, "ops")
		assert.NoError(t, err)
		_, err = blocks.Unblock(ctx, "jane@example.com")
		assert.NoError(t, err)
		assert.NoError(t, auth.RequestCode(ctx, "jane@example.com"))
		assert.Len(t, sender.codes["jane@example.com"], 6)
	})
}
