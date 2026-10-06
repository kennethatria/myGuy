package repositories

import (
	"context"
	"testing"
	"time"

	"myguy/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskRepository_ListIDsAndListByIDs(t *testing.T) {
	db, err := setupTestDB()
	require.NoError(t, err)
	repo := NewGormTaskRepository(db)
	ctx := context.Background()

	poster := &models.User{Username: "poster", Email: "poster@example.com"}
	other := &models.User{Username: "other", Email: "other@example.com"}
	require.NoError(t, db.Create(poster).Error)
	require.NoError(t, db.Create(other).Error)

	base := time.Now().Add(-time.Hour)
	var ids []uint
	for i, status := range []string{"open", "open", "expired", "open"} {
		task := &models.Task{Title: "Gig", Description: "Note", Deadline: base.Add(24 * time.Hour),
			CreatedBy: poster.ID, Status: status, CreatedAt: base.Add(time.Duration(i) * time.Minute)}
		require.NoError(t, repo.Create(ctx, task))
		ids = append(ids, task.ID)
	}
	mine := &models.Task{Title: "Mine", Description: "Note", Deadline: base.Add(24 * time.Hour), CreatedBy: other.ID, Status: "open", CreatedAt: base.Add(10 * time.Minute)}
	require.NoError(t, repo.Create(ctx, mine))

	got, err := repo.ListIDs(ctx, map[string]interface{}{"status": "open", "exclude_created_by": other.ID, "page": 2, "per_page": 1})

	require.NoError(t, err)
	assert.Equal(t, []uint{ids[3], ids[1], ids[0]}, got, "every open match, newest first, ignoring paging")

	tasks, err := repo.ListByIDs(ctx, []uint{ids[1], mine.ID})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	for _, task := range tasks {
		assert.NotEmpty(t, task.Creator.Username, "creator loaded")
	}

	none, err := repo.ListByIDs(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}
