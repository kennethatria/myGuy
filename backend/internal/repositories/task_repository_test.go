package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestTaskRepository(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)

	repo := NewGormTaskRepository(db)
	ctx := context.Background()

	// Create a user for foreign key constraints
	user := &models.User{Username: "taskcreator", Email: "creator@example.com"}
	db.Create(user)

	t.Run("Create and GetByID", func(t *testing.T) {
		task := &models.Task{
			Title:       "Test Task",
			Description: "Task Description",
			Fee:         100.0,
			Deadline:    time.Now().Add(24 * time.Hour),
			CreatedBy:   user.ID,
			Status:      "open",
		}

		err := repo.Create(ctx, task)
		assert.NoError(t, err)
		assert.NotZero(t, task.ID)

		fetched, err := repo.GetByID(ctx, task.ID)
		assert.NoError(t, err)
		assert.Equal(t, task.Title, fetched.Title)
		assert.Equal(t, user.ID, fetched.Creator.ID)
	})

	t.Run("List with Filters", func(t *testing.T) {
		// Create some tasks
		db.Create(&models.Task{Title: "Task 1", Fee: 50, CreatedBy: user.ID, Status: "open"})
		db.Create(&models.Task{Title: "Task 2", Fee: 150, CreatedBy: user.ID, Status: "completed"})

		// Filter by status
		tasks, err := repo.List(ctx, map[string]interface{}{"status": "open"})
		assert.NoError(t, err)
		assert.NotEmpty(t, tasks)
		for _, task := range tasks {
			assert.Equal(t, "open", task.Status)
		}

		// Filter by min_fee
		tasks, err = repo.List(ctx, map[string]interface{}{"min_fee": 100.0})
		assert.NoError(t, err)
		assert.NotEmpty(t, tasks)
		for _, task := range tasks {
			assert.GreaterOrEqual(t, task.Fee, 100.0)
		}
	})

	t.Run("ListWithPagination", func(t *testing.T) {
		// Create 5 tasks
		for i := 0; i < 5; i++ {
			db.Create(&models.Task{Title: "Paging Task", CreatedBy: user.ID})
		}

		tasks, err := repo.ListWithPagination(ctx, map[string]interface{}{
			"page":     1,
			"per_page": 2,
		})
		assert.NoError(t, err)
		assert.Equal(t, 2, len(tasks))
	})

	t.Run("Update", func(t *testing.T) {
		task := &models.Task{Title: "Old Title", CreatedBy: user.ID}
		db.Create(task)

		task.Title = "New Title"
		err := repo.Update(ctx, task)
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, task.ID)
		assert.Equal(t, "New Title", fetched.Title)
	})

	t.Run("Delete", func(t *testing.T) {
		task := &models.Task{Title: "To Delete", CreatedBy: user.ID}
		db.Create(task)

		err := repo.Delete(ctx, task.ID)
		assert.NoError(t, err)

		_, err = repo.GetByID(ctx, task.ID)
		assert.Error(t, err)
	})

	t.Run("ListByUser", func(t *testing.T) {
		task := &models.Task{Title: "User Task", CreatedBy: user.ID}
		db.Create(task)

		tasks, err := repo.ListByUser(ctx, user.ID, "creator")
		assert.NoError(t, err)
		assert.NotEmpty(t, tasks)
		assert.Equal(t, user.ID, tasks[0].CreatedBy)
	})
}
