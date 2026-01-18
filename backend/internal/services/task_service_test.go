package services

import (
	"context"
	"testing"
	"time"

	"myguy/internal/models"
	"myguy/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTaskService() (*TaskService, *tests.MockTaskRepository, *tests.MockApplicationRepository) {
	taskRepo := new(tests.MockTaskRepository)
	appRepo := new(tests.MockApplicationRepository)
	service := NewTaskService(taskRepo, appRepo)
	return service, taskRepo, appRepo
}

func TestCreateTask(t *testing.T) {
	t.Run("successful task creation", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()
		
		deadline := time.Now().Add(48 * time.Hour)
		input := CreateTaskInput{
			Title:       "Test Task",
			Description: "Description",
			Fee:         100.0,
			Deadline:    deadline,
			CreatedBy:   1,
		}

		taskRepo.On("Create", ctx, mock.MatchedBy(func(task *models.Task) bool {
			return task.Title == input.Title && 
			       task.Fee == input.Fee && 
				   task.Status == "open"
		})).Return(nil)

		task, err := service.CreateTask(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, input.Title, task.Title)
		assert.Equal(t, "open", task.Status)
		taskRepo.AssertExpectations(t)
	})

	t.Run("invalid deadline - too soon", func(t *testing.T) {
		service, _, _ := setupTaskService()
		ctx := context.Background()
		
		// Deadline 1 hour from now (should fail, needs 24h)
		deadline := time.Now().Add(1 * time.Hour)
		input := CreateTaskInput{
			Title:    "Test Task",
			Deadline: deadline,
		}

		task, err := service.CreateTask(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, task)
		assert.Equal(t, ErrInvalidDeadline, err)
	})
}
