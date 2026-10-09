package services

import (
	"context"
	"errors"
	"testing"

	"store-service/internal/models"
	"store-service/internal/proximity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeDistancer struct {
	buckets map[uint]int
	err     error
	from    string
}

func (f *fakeDistancer) Distances(_ context.Context, kind string, at proximity.Location, ids []uint) (map[uint]int, error) {
	return f.buckets, f.err
}

func (f *fakeDistancer) DistancesFrom(_ context.Context, kind, fromKind string, fromID uint, ids []uint) (map[uint]int, error) {
	f.from = fromKind
	return f.buckets, f.err
}

var at = proximity.Location{Lat: 0.35, Lng: 32.585}

func itemIDs(items []models.StoreItem) []uint {
	out := make([]uint, len(items))
	for i, item := range items {
		out[i] = item.ID
	}
	return out
}

func TestGetItemsNear(t *testing.T) {
	// newest first 6..1; 6 and 3 have no location
	dist := &fakeDistancer{buckets: map[uint]int{5: 0, 2: 0, 4: 2, 1: 1}}

	t.Run("ranked, paged and tagged", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		service.WithDistancer(dist)
		filter := models.StoreItemFilter{Status: "active", Page: 1, PerPage: 4}
		itemRepo.On("ListIDs", filter).Return([]uint{6, 5, 4, 3, 2, 1}, nil)
		itemRepo.On("GetByIDs", []uint{5, 2, 1, 4}).Return([]models.StoreItem{{ID: 4}, {ID: 1}, {ID: 2}, {ID: 5}}, nil)

		items, total, err := service.GetItemsNear(context.Background(), filter, at)

		require.NoError(t, err)
		assert.Equal(t, []uint{5, 2, 1, 4}, itemIDs(items))
		assert.Equal(t, "<1 km", items[0].Distance)
		assert.Equal(t, "~5 km", items[3].Distance)
		assert.Equal(t, int64(6), total)
	})

	t.Run("listings without a location come last, untagged", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		service.WithDistancer(dist)
		filter := models.StoreItemFilter{Page: 2, PerPage: 4}
		itemRepo.On("ListIDs", filter).Return([]uint{6, 5, 4, 3, 2, 1}, nil)
		itemRepo.On("GetByIDs", []uint{6, 3}).Return([]models.StoreItem{{ID: 3}, {ID: 6}}, nil)

		items, _, err := service.GetItemsNear(context.Background(), filter, at)
		require.NoError(t, err)
		assert.Equal(t, []uint{6, 3}, itemIDs(items))
		assert.Empty(t, items[0].Distance)
	})

	t.Run("falls back to newest first when distances fail or are off", func(t *testing.T) {
		for _, d := range []Distancer{&fakeDistancer{err: errors.New("timeout")}, nil} {
			service, itemRepo, _ := setupService()
			service.WithDistancer(d)
			filter := models.StoreItemFilter{Page: 1, PerPage: 20}
			itemRepo.On("ListIDs", filter).Return([]uint{1}, nil).Maybe()
			itemRepo.On("GetAll", filter).Return([]models.StoreItem{{ID: 1}}, int64(1), nil)

			items, total, err := service.GetItemsNear(context.Background(), filter, at)
			require.NoError(t, err)
			assert.Equal(t, []uint{1}, itemIDs(items))
			assert.Equal(t, int64(1), total)
		}
	})

	t.Run("repository errors are returned", func(t *testing.T) {
		service, itemRepo, _ := setupService()
		service.WithDistancer(dist)
		itemRepo.On("ListIDs", mock.Anything).Return([]uint(nil), errors.New("db down"))
		_, _, err := service.GetItemsNear(context.Background(), models.StoreItemFilter{}, at)
		assert.Error(t, err)
	})
}

func TestTagItemDistances(t *testing.T) {
	service, _, _ := setupService()
	items := []models.StoreItem{{ID: 1}, {ID: 2}}
	service.TagItemDistances(context.Background(), items, at)
	assert.Empty(t, items[1].Distance)

	service.WithDistancer(&fakeDistancer{buckets: map[uint]int{2: 1}})
	service.TagItemDistances(context.Background(), items, at)
	assert.Empty(t, items[0].Distance)
	assert.Equal(t, "~2 km", items[1].Distance)

	service.WithDistancer(&fakeDistancer{err: errors.New("down")})
	fresh := []models.StoreItem{{ID: 2}}
	service.TagItemDistances(context.Background(), fresh, at)
	assert.Empty(t, fresh[0].Distance)
}

func TestGetRequestsNear(t *testing.T) {
	service, requests, _ := setupRequestService()
	service.WithDistancer(&fakeDistancer{buckets: map[uint]int{2: 0}})
	filter := models.ItemRequestFilter{Page: 1, PerPage: 10}
	requests.On("ListIDs", filter).Return([]uint{3, 2, 1}, nil)
	requests.On("GetByIDs", []uint{2, 3, 1}).Return([]models.ItemRequest{{ID: 1}, {ID: 3}, {ID: 2}}, nil)

	got, total, err := service.GetRequestsNear(context.Background(), filter, at)

	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Equal(t, uint(2), got[0].ID)
	assert.Equal(t, "<1 km", got[0].Distance)
	assert.Equal(t, []uint{3, 1}, []uint{got[1].ID, got[2].ID}, "the rest newest first")

	// tags, and fallbacks
	tagged := []models.ItemRequest{{ID: 2}, {ID: 9}}
	service.TagRequestDistances(context.Background(), tagged, at)
	assert.Equal(t, "<1 km", tagged[0].Distance)
	assert.Empty(t, tagged[1].Distance)

	off, offRequests, _ := setupRequestService()
	offRequests.On("GetAll", filter).Return([]models.ItemRequest{{ID: 3}}, int64(1), nil)
	plain, _, err := off.GetRequestsNear(context.Background(), filter, at)
	require.NoError(t, err)
	assert.Equal(t, uint(3), plain[0].ID)
	off.TagRequestDistances(context.Background(), plain, at)
	assert.Empty(t, plain[0].Distance)

	broken, brokenRequests, _ := setupRequestService()
	broken.WithDistancer(&fakeDistancer{err: errors.New("down")})
	brokenRequests.On("ListIDs", filter).Return([]uint{3}, nil)
	brokenRequests.On("GetAll", filter).Return([]models.ItemRequest{{ID: 3}}, int64(1), nil)
	_, _, err = broken.GetRequestsNear(context.Background(), filter, at)
	assert.NoError(t, err)
	broken.TagRequestDistances(context.Background(), plain, at)
}

func TestGetRequestListingsNearestToRequester(t *testing.T) {
	service, _, items := setupRequestService()
	dist := &fakeDistancer{buckets: map[uint]int{52: 0, 51: 3}}
	service.WithDistancer(dist)
	items.On("GetAll", models.StoreItemFilter{RequestID: 31, Status: "active", PerPage: 100}).
		Return([]models.StoreItem{{ID: 53}, {ID: 52}, {ID: 51}}, int64(3), nil)

	got, err := service.GetRequestListings(context.Background(), 31)

	require.NoError(t, err)
	assert.Equal(t, "request", dist.from)
	assert.Equal(t, []uint{52, 51, 53}, itemIDs(got))
	assert.Equal(t, "<1 km", got[0].Distance)
	assert.Equal(t, "~10 km", got[1].Distance)
	assert.Empty(t, got[2].Distance)

	// distances down: newest first, untagged
	service.WithDistancer(&fakeDistancer{err: errors.New("down")})
	got, err = service.GetRequestListings(context.Background(), 31)
	require.NoError(t, err)
	assert.Equal(t, []uint{53, 52, 51}, itemIDs(got))
}
