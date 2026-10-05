package repositories

import (
	"context"
	"testing"
	"time"

	"proximity-service/internal/geo"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (LocationRepository, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	return NewRedisLocationRepository(rdb), mr
}

// cell builds a point the way the service does: snapped to a cell centre
func cell(t *testing.T, lat, lng float64) geo.Point {
	p, err := geo.Snap(lat, lng)
	require.NoError(t, err)
	return p
}

func TestSaveAndPositions(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	kampala := cell(t, 0.3476, 32.5842)
	south := cell(t, -0.33, 31.735)

	require.NoError(t, repo.Save(ctx, "task", 1, kampala, time.Now()))
	require.NoError(t, repo.Save(ctx, "task", 2, south, time.Now()))
	require.NoError(t, repo.Save(ctx, "item", 1, south, time.Now()))

	got, err := repo.Positions(ctx, "task", []uint64{1, 2, 3})

	require.NoError(t, err)
	assert.Equal(t, map[uint64]geo.Point{1: kampala, 2: south}, got, "snapped back to exact cells; 3 has none")

	items, err := repo.Positions(ctx, "item", []uint64{1})
	require.NoError(t, err)
	assert.Equal(t, south, items[1], "kinds are kept apart")

	none, err := repo.Positions(ctx, "task", nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestSaveReplacesAndDelete(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, repo.Save(ctx, "request", 7, cell(t, 0.35, 32.585), time.Now()))
	moved := cell(t, 0.3, 32.6)
	require.NoError(t, repo.Save(ctx, "request", 7, moved, time.Now()))

	got, err := repo.Positions(ctx, "request", []uint64{7})
	require.NoError(t, err)
	assert.Equal(t, moved, got[7])

	require.NoError(t, repo.Delete(ctx, "request", 7))
	got, err = repo.Positions(ctx, "request", []uint64{7})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRemoveSavedBefore(t *testing.T) {
	repo, _ := setup(t)
	ctx := context.Background()
	now := time.Now()
	p := cell(t, 0.35, 32.585)
	require.NoError(t, repo.Save(ctx, "item", 1, p, now.Add(-40*24*time.Hour)))
	require.NoError(t, repo.Save(ctx, "item", 2, p, now.Add(-time.Hour)))

	n, err := repo.RemoveSavedBefore(ctx, "item", now.Add(-30*24*time.Hour))

	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	got, err := repo.Positions(ctx, "item", []uint64{1, 2})
	require.NoError(t, err)
	assert.Equal(t, map[uint64]geo.Point{2: p}, got)

	n, err = repo.RemoveSavedBefore(ctx, "item", now.Add(-30*24*time.Hour))
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestRedisDown(t *testing.T) {
	repo, mr := setup(t)
	ctx := context.Background()
	require.NoError(t, repo.Ping(ctx))
	mr.Close()

	assert.Error(t, repo.Ping(ctx))
	assert.Error(t, repo.Save(ctx, "task", 1, geo.Point{}, time.Now()))
	assert.Error(t, repo.Delete(ctx, "task", 1))
	_, err := repo.Positions(ctx, "task", []uint64{1})
	assert.Error(t, err)
	_, err = repo.RemoveSavedBefore(ctx, "task", time.Now())
	assert.Error(t, err)
}
