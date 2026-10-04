package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"myguy/internal/models"
)

func TestLoginCodeRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	setup := func(t *testing.T) *GormLoginCodeRepository {
		db, err := setupTestDB()
		mustNoError(t, err)
		return NewGormLoginCodeRepository(db)
	}

	t.Run("LatestActive returns newest unconsumed unexpired code", func(t *testing.T) {
		repo := setup(t)
		mustNoError(t, repo.Create(ctx, &models.LoginCode{Email: "a@example.com", CodeHash: "old", ExpiresAt: now.Add(time.Minute)}))
		mustNoError(t, repo.Create(ctx, &models.LoginCode{Email: "a@example.com", CodeHash: "new", ExpiresAt: now.Add(time.Minute)}))
		mustNoError(t, repo.Create(ctx, &models.LoginCode{Email: "a@example.com", CodeHash: "expired", ExpiresAt: now.Add(-time.Minute)}))

		code, err := repo.LatestActive(ctx, "a@example.com", now)
		mustNoError(t, err)
		assert.Equal(t, "new", code.CodeHash)
	})

	t.Run("MarkConsumed succeeds only once", func(t *testing.T) {
		repo := setup(t)
		code := &models.LoginCode{Email: "a@example.com", CodeHash: "h", ExpiresAt: now.Add(time.Minute)}
		mustNoError(t, repo.Create(ctx, code))

		ok, err := repo.MarkConsumed(ctx, code.ID, now)
		mustNoError(t, err)
		assert.True(t, ok)

		ok, err = repo.MarkConsumed(ctx, code.ID, now)
		mustNoError(t, err)
		assert.False(t, ok)

		_, err = repo.LatestActive(ctx, "a@example.com", now)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})

	t.Run("InvalidateActive consumes outstanding codes for that email only", func(t *testing.T) {
		repo := setup(t)
		mustNoError(t, repo.Create(ctx, &models.LoginCode{Email: "a@example.com", CodeHash: "a", ExpiresAt: now.Add(time.Minute)}))
		mustNoError(t, repo.Create(ctx, &models.LoginCode{Email: "b@example.com", CodeHash: "b", ExpiresAt: now.Add(time.Minute)}))

		mustNoError(t, repo.InvalidateActive(ctx, "a@example.com", now))

		_, err := repo.LatestActive(ctx, "a@example.com", now)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = repo.LatestActive(ctx, "b@example.com", now)
		assert.NoError(t, err)
	})

	t.Run("IncrementAttempts and CountSince", func(t *testing.T) {
		repo := setup(t)
		code := &models.LoginCode{Email: "a@example.com", CodeHash: "h", ExpiresAt: now.Add(time.Minute)}
		mustNoError(t, repo.Create(ctx, code))
		mustNoError(t, repo.IncrementAttempts(ctx, code.ID))
		mustNoError(t, repo.IncrementAttempts(ctx, code.ID))

		got, err := repo.LatestActive(ctx, "a@example.com", now)
		mustNoError(t, err)
		assert.Equal(t, 2, got.Attempts)

		count, err := repo.CountSince(ctx, "a@example.com", now.Add(-time.Hour))
		mustNoError(t, err)
		assert.Equal(t, int64(1), count)
		count, err = repo.CountSince(ctx, "a@example.com", now.Add(time.Hour))
		mustNoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func mustNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
