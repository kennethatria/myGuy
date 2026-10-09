package services

import (
	"context"
	"errors"
	"testing"

	"myguy/internal/models"
	"myguy/internal/proximity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeDistancer struct {
	buckets map[uint]int
	err     error
	asked   [][]uint
}

func (f *fakeDistancer) Distances(_ context.Context, kind string, at proximity.Location, ids []uint) (map[uint]int, error) {
	f.asked = append(f.asked, append([]uint(nil), ids...))
	return f.buckets, f.err
}

func tasksFor(ids ...uint) []models.Task {
	out := make([]models.Task, len(ids))
	for i, id := range ids {
		out[i] = models.Task{ID: id, Title: "Gig"}
	}
	return out
}

func idsOf(tasks []models.Task) []uint {
	out := make([]uint, len(tasks))
	for i, task := range tasks {
		out[i] = task.ID
	}
	return out
}

func TestListTasksNear(t *testing.T) {
	ctx := context.Background()
	at := proximity.Location{Lat: 0.35, Lng: 32.585}
	// newest first: 6, 5, 4, 3, 2, 1
	all := []uint{6, 5, 4, 3, 2, 1}
	// 6 and 3 have no location; 5 and 2 are near (<1 km); 4 is ~5 km; 1 is ~2 km
	dist := &fakeDistancer{buckets: map[uint]int{5: 0, 2: 0, 4: 2, 1: 1}}

	t.Run("nearest bucket first, newest within a bucket, no location last; paged", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		service.WithDistancer(dist)
		filters := map[string]interface{}{"status": "open", "page": 1, "per_page": 4}
		taskRepo.On("ListIDs", ctx, filters).Return(append([]uint(nil), all...), nil)
		taskRepo.On("ListByIDs", ctx, []uint{5, 2, 1, 4}).Return(tasksFor(4, 1, 5, 2), nil)

		got, err := service.ListTasksNear(ctx, filters, at)

		require.NoError(t, err)
		assert.Equal(t, []uint{5, 2, 1, 4}, idsOf(got.Tasks))
		assert.Equal(t, []string{"<1 km", "<1 km", "~2 km", "~5 km"}, []string{got.Tasks[0].Distance, got.Tasks[1].Distance, got.Tasks[2].Distance, got.Tasks[3].Distance})
		assert.Equal(t, int64(6), got.Total)
		assert.Equal(t, 2, got.TotalPages)
	})

	t.Run("second page holds the posts without a location, untagged", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		service.WithDistancer(dist)
		filters := map[string]interface{}{"page": 2, "per_page": 4}
		taskRepo.On("ListIDs", ctx, filters).Return(append([]uint(nil), all...), nil)
		taskRepo.On("ListByIDs", ctx, []uint{6, 3}).Return(tasksFor(3, 6), nil)

		got, err := service.ListTasksNear(ctx, filters, at)

		require.NoError(t, err)
		assert.Equal(t, []uint{6, 3}, idsOf(got.Tasks))
		assert.Empty(t, got.Tasks[0].Distance)
	})

	t.Run("a page past the end is empty", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		service.WithDistancer(dist)
		filters := map[string]interface{}{"page": 9, "per_page": 4}
		taskRepo.On("ListIDs", ctx, filters).Return(append([]uint(nil), all...), nil)
		taskRepo.On("ListByIDs", ctx, []uint{}).Return([]models.Task{}, nil)

		got, err := service.ListTasksNear(ctx, filters, at)
		require.NoError(t, err)
		assert.Empty(t, got.Tasks)
	})

	t.Run("falls back to newest first when distances fail or are off", func(t *testing.T) {
		for _, d := range []Distancer{&fakeDistancer{err: errors.New("timeout")}, nil} {
			service, taskRepo, _ := setupTaskService()
			service.WithDistancer(d)
			filters := map[string]interface{}{"page": 1, "per_page": 20}
			taskRepo.On("ListIDs", ctx, filters).Return(all, nil).Maybe()
			taskRepo.On("Count", ctx, filters).Return(int64(1), nil)
			taskRepo.On("ListWithPagination", ctx, filters).Return(tasksFor(6), nil)

			got, err := service.ListTasksNear(ctx, filters, at)
			require.NoError(t, err)
			assert.Equal(t, []uint{6}, idsOf(got.Tasks))
			assert.Empty(t, got.Tasks[0].Distance)
		}
	})

	t.Run("repository errors are returned", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		service.WithDistancer(dist)
		taskRepo.On("ListIDs", ctx, mock.Anything).Return([]uint(nil), errors.New("db down"))
		_, err := service.ListTasksNear(ctx, map[string]interface{}{}, at)
		assert.Error(t, err)
	})
}

func TestTagDistances(t *testing.T) {
	at := proximity.Location{Lat: 0.35, Lng: 32.585}
	service, _, _ := setupTaskService()

	tasks := tasksFor(1, 2)
	service.TagDistances(context.Background(), tasks, at) // no distancer: no tags, no panic
	assert.Empty(t, tasks[0].Distance)

	service.WithDistancer(&fakeDistancer{buckets: map[uint]int{2: 4}})
	service.TagDistances(context.Background(), tasks, at)
	assert.Empty(t, tasks[0].Distance)
	assert.Equal(t, "10+ km", tasks[1].Distance)

	service.WithDistancer(&fakeDistancer{err: errors.New("down")})
	fresh := tasksFor(1)
	service.TagDistances(context.Background(), fresh, at)
	assert.Empty(t, fresh[0].Distance)
}
