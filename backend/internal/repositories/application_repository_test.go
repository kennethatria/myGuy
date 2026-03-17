package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"myguy/internal/models"
)

func TestApplicationRepository(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)

	repo := NewGormApplicationRepository(db)
	ctx := context.Background()

	// Create a user and a task
	user := &models.User{Username: "applicant", Email: "applicant@example.com"}
	db.Create(user)
	creator := &models.User{Username: "creator", Email: "creator@example.com"}
	db.Create(creator)
	task := &models.Task{Title: "Task for App", CreatedBy: creator.ID}
	db.Create(task)

	t.Run("Create and GetByID", func(t *testing.T) {
		app := &models.Application{
			TaskID:      task.ID,
			ApplicantID: user.ID,
			Message:     "I want to do this task",
			Status:      "pending",
		}

		err := repo.Create(ctx, app)
		assert.NoError(t, err)
		assert.NotZero(t, app.ID)

		fetched, err := repo.GetByID(ctx, app.ID)
		assert.NoError(t, err)
		assert.Equal(t, app.Message, fetched.Message)
		assert.Equal(t, user.ID, fetched.Applicant.ID)
		assert.Equal(t, task.ID, fetched.Task.ID)
	})

	t.Run("ListByTask", func(t *testing.T) {
		apps, err := repo.ListByTask(ctx, task.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, apps)
		assert.Equal(t, task.ID, apps[0].TaskID)
	})

	t.Run("ListByUser", func(t *testing.T) {
		apps, err := repo.ListByUser(ctx, user.ID)
		assert.NoError(t, err)
		assert.NotEmpty(t, apps)
		assert.Equal(t, user.ID, apps[0].ApplicantID)
	})

	t.Run("Update", func(t *testing.T) {
		app := &models.Application{TaskID: task.ID, ApplicantID: user.ID, Status: "pending"}
		db.Create(app)

		app.Status = "accepted"
		err := repo.Update(ctx, app)
		assert.NoError(t, err)

		fetched, _ := repo.GetByID(ctx, app.ID)
		assert.Equal(t, "accepted", fetched.Status)
	})
}
