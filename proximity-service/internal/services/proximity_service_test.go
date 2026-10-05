package services

import (
	"context"
	"testing"
	"time"

	"proximity-service/internal/geo"
	"proximity-service/internal/repositories"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*ProximityService, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	return NewProximityService(repositories.NewRedisLocationRepository(rdb)), mr
}

func f(v float64) *float64 { return &v }

func TestSaveAndDistances(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	// Viewer in central Kampala (0.3476, 32.5842)
	require.NoError(t, s.SaveLocation(ctx, "task", 1, 0.3476, 32.5842)) // same cell
	require.NoError(t, s.SaveLocation(ctx, "task", 2, 0.3476, 32.60))   // ~1.7 km east
	require.NoError(t, s.SaveLocation(ctx, "task", 3, 0.3476, 32.63))   // ~5 km
	require.NoError(t, s.SaveLocation(ctx, "task", 4, 0.3476, 32.66))   // ~8.5 km
	require.NoError(t, s.SaveLocation(ctx, "task", 5, 0.05, 32.46))     // Entebbe, ~35 km

	got, err := s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(0.3476), Lng: f(32.5842), IDs: []uint64{5, 4, 3, 2, 1, 99}})

	require.NoError(t, err)
	assert.Equal(t, []Result{{5, 4}, {4, 3}, {3, 2}, {2, 1}, {1, 0}}, got, "asked order kept; 99 has no location")
}

func TestDistancesFromStoredPost(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, s.SaveLocation(ctx, "request", 31, 0.3476, 32.5842))
	require.NoError(t, s.SaveLocation(ctx, "item", 51, 0.3476, 32.60))

	got, err := s.Distances(ctx, DistancesQuery{Kind: "item", From: &Ref{Kind: "request", ID: 31}, IDs: []uint64{51}})
	require.NoError(t, err)
	assert.Equal(t, []Result{{51, 1}}, got)

	none, err := s.Distances(ctx, DistancesQuery{Kind: "item", From: &Ref{Kind: "request", ID: 32}, IDs: []uint64{51}})
	require.NoError(t, err)
	assert.Empty(t, none, "a request without a location gives no buckets")
}

func TestValidation(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()

	assert.ErrorIs(t, s.SaveLocation(ctx, "user", 1, 0, 0), ErrUnknownKind)
	assert.ErrorIs(t, s.SaveLocation(ctx, "task", 0, 0, 0), ErrInvalidID)
	assert.ErrorIs(t, s.SaveLocation(ctx, "task", 1, 95, 0), geo.ErrOutOfRange)
	assert.ErrorIs(t, s.DeleteLocation(ctx, "user", 1), ErrUnknownKind)
	assert.ErrorIs(t, s.DeleteLocation(ctx, "task", 0), ErrInvalidID)

	_, err := s.Distances(ctx, DistancesQuery{Kind: "user", Lat: f(0), Lng: f(0)})
	assert.ErrorIs(t, err, ErrUnknownKind)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(0)})
	assert.ErrorIs(t, err, ErrNoOrigin)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(0), Lng: f(0), From: &Ref{Kind: "task", ID: 1}})
	assert.ErrorIs(t, err, ErrNoOrigin)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", From: &Ref{Kind: "user", ID: 1}})
	assert.ErrorIs(t, err, ErrUnknownKind)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(100), Lng: f(0)})
	assert.ErrorIs(t, err, geo.ErrOutOfRange)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(0), Lng: f(0), IDs: make([]uint64, MaxIDs+1)})
	assert.ErrorIs(t, err, ErrTooManyIDs)
}

func TestDeleteAndCleanup(t *testing.T) {
	s, _ := setup(t)
	ctx := context.Background()
	now := time.Now()
	s.now = func() time.Time { return now.Add(-31 * 24 * time.Hour) }
	require.NoError(t, s.SaveLocation(ctx, "task", 1, 0.35, 32.58))
	require.NoError(t, s.SaveLocation(ctx, "request", 2, 0.35, 32.58))
	s.now = func() time.Time { return now }
	require.NoError(t, s.SaveLocation(ctx, "item", 3, 0.35, 32.58))
	require.NoError(t, s.SaveLocation(ctx, "item", 4, 0.35, 32.58))
	require.NoError(t, s.DeleteLocation(ctx, "item", 4))

	n, err := s.Cleanup(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	got, err := s.Distances(ctx, DistancesQuery{Kind: "item", Lat: f(0.35), Lng: f(32.58), IDs: []uint64{3, 4}})
	require.NoError(t, err)
	assert.Equal(t, []Result{{3, 0}}, got)
}

func TestRedisDown(t *testing.T) {
	s, mr := setup(t)
	ctx := context.Background()
	require.NoError(t, s.Healthy(ctx))
	mr.Close()

	assert.Error(t, s.Healthy(ctx))
	_, err := s.Distances(ctx, DistancesQuery{Kind: "task", Lat: f(0), Lng: f(0), IDs: []uint64{1}})
	assert.Error(t, err)
	_, err = s.Distances(ctx, DistancesQuery{Kind: "task", From: &Ref{Kind: "task", ID: 1}, IDs: []uint64{1}})
	assert.Error(t, err)
	_, err = s.Cleanup(ctx)
	assert.Error(t, err)
}
