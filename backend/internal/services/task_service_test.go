package services

import (
	"context"
	"errors"
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

// ==================== CreateTask Tests ====================

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

	t.Run("repository create error", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		deadline := time.Now().Add(48 * time.Hour)
		input := CreateTaskInput{
			Title:    "Test Task",
			Deadline: deadline,
		}

		taskRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		task, err := service.CreateTask(ctx, input)

		assert.Error(t, err)
		assert.Nil(t, task)
		taskRepo.AssertExpectations(t)
	})
}

// ==================== UpdateTask Tests ====================

func TestUpdateTask(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{
			ID:        1,
			Title:     "Old Title",
			CreatedBy: 1,
			Status:    "open",
		}

		deadline := time.Now().Add(48 * time.Hour)
		input := UpdateTaskInput{
			ID:          1,
			Title:       "New Title",
			Description: "New Description",
			Fee:         150.0,
			Deadline:    deadline,
			UpdatedBy:   1,
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(task *models.Task) bool {
			return task.Title == "New Title" && task.Fee == 150.0
		})).Return(nil)

		task, err := service.UpdateTask(ctx, input)

		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "New Title", task.Title)
		taskRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		input := UpdateTaskInput{ID: 999, UpdatedBy: 1}
		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		task, err := service.UpdateTask(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
		assert.Nil(t, task)
	})

	t.Run("unauthorized - not owner", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{ID: 1, CreatedBy: 1}
		input := UpdateTaskInput{
			ID:        1,
			UpdatedBy: 2, // Different user
			Deadline:  time.Now().Add(48 * time.Hour),
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)

		task, err := service.UpdateTask(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		assert.Nil(t, task)
	})

	t.Run("invalid deadline", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{ID: 1, CreatedBy: 1}
		input := UpdateTaskInput{
			ID:        1,
			UpdatedBy: 1,
			Deadline:  time.Now().Add(1 * time.Hour), // Too soon
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)

		task, err := service.UpdateTask(ctx, input)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidDeadline, err)
		assert.Nil(t, task)
	})
}

// ==================== GetTask Tests ====================

func TestGetTask(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{
			ID:    1,
			Title: "Test Task",
		}

		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)

		task, err := service.GetTask(ctx, 1)

		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "Test Task", task.Title)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		task, err := service.GetTask(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
		assert.Nil(t, task)
	})
}

// ==================== DeleteTask Tests ====================

func TestDeleteTask(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{ID: 1, CreatedBy: 1}
		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)
		taskRepo.On("Delete", ctx, uint(1)).Return(nil)

		err := service.DeleteTask(ctx, 1, 1)

		assert.NoError(t, err)
		taskRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.DeleteTask(ctx, 999, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
	})

	t.Run("unauthorized - not owner", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{ID: 1, CreatedBy: 1}
		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)

		err := service.DeleteTask(ctx, 1, 2) // Different user

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
}

// ==================== ListTasks Tests ====================

func TestListTasks(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		expectedTasks := []models.Task{
			{ID: 1, Title: "Task 1"},
			{ID: 2, Title: "Task 2"},
		}

		filters := map[string]interface{}{"status": "open"}
		taskRepo.On("List", ctx, filters).Return(expectedTasks, nil)

		tasks, err := service.ListTasks(ctx, filters)

		assert.NoError(t, err)
		assert.Len(t, tasks, 2)
		taskRepo.AssertExpectations(t)
	})
}

// ==================== ListTasksWithPagination Tests ====================

func TestListTasksWithPagination(t *testing.T) {
	t.Run("successful pagination", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		expectedTasks := []models.Task{
			{ID: 1, Title: "Task 1"},
		}

		filters := map[string]interface{}{"page": 1, "per_page": 10}
		taskRepo.On("Count", ctx, filters).Return(int64(25), nil)
		taskRepo.On("ListWithPagination", ctx, filters).Return(expectedTasks, nil)

		result, err := service.ListTasksWithPagination(ctx, filters)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(25), result.Total)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 10, result.PerPage)
		assert.Equal(t, 3, result.TotalPages)
		taskRepo.AssertExpectations(t)
	})

	t.Run("default pagination values", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		filters := map[string]interface{}{} // No page/per_page
		taskRepo.On("Count", ctx, filters).Return(int64(100), nil)
		taskRepo.On("ListWithPagination", ctx, filters).Return([]models.Task{}, nil)

		result, err := service.ListTasksWithPagination(ctx, filters)

		assert.NoError(t, err)
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 20, result.PerPage)
		assert.Equal(t, 5, result.TotalPages)
	})
}

// ==================== ApplyForTask Tests ====================

func TestApplyForTask(t *testing.T) {
	t.Run("successful application", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		openTask := &models.Task{ID: 1, Status: "open"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(openTask, nil)
		appRepo.On("Create", ctx, mock.MatchedBy(func(app *models.Application) bool {
			return app.TaskID == 1 &&
				app.ApplicantID == 2 &&
				app.ProposedFee == 50.0 &&
				app.Status == "pending"
		})).Return(nil)

		err := service.ApplyForTask(ctx, 1, 2, 50.0, "I can do this")

		assert.NoError(t, err)
		taskRepo.AssertExpectations(t)
		appRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.ApplyForTask(ctx, 999, 2, 50.0, "Message")

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
	})

	t.Run("task not open", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		closedTask := &models.Task{ID: 1, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(closedTask, nil)

		err := service.ApplyForTask(ctx, 1, 2, 50.0, "Message")

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotOpen, err)
	})
}

// ==================== AssignTask Tests ====================

func TestAssignTask(t *testing.T) {
	t.Run("successful assignment", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, Status: "open", Fee: 100}
		application := &models.Application{ID: 1, TaskID: 1, ApplicantID: 2, ProposedFee: 80}

		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		appRepo.On("GetByID", ctx, uint(1)).Return(application, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(t *models.Task) bool {
			return t.Status == "in_progress" &&
				t.AssignedTo != nil &&
				*t.AssignedTo == uint(2) &&
				t.Fee == 80
		})).Return(nil)
		appRepo.On("Update", ctx, mock.MatchedBy(func(a *models.Application) bool {
			return a.Status == "accepted"
		})).Return(nil)

		result, err := service.AssignTask(ctx, 1, 1)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "in_progress", result.Status)
		taskRepo.AssertExpectations(t)
		appRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.AssignTask(ctx, 999, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("application not found", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		appRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.AssignTask(ctx, 1, 999)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

// ==================== CompleteTask Tests ====================

func TestCompleteTask(t *testing.T) {
	t.Run("creator completes task", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(t *models.Task) bool {
			return t.Status == "completed" && t.CompletedAt != nil
		})).Return(nil)

		err := service.CompleteTask(ctx, 1, 1)

		assert.NoError(t, err)
		taskRepo.AssertExpectations(t)
	})

	t.Run("assignee completes task", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignedTo := uint(2)
		task := &models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignedTo, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.Anything).Return(nil)

		err := service.CompleteTask(ctx, 1, 2)

		assert.NoError(t, err)
	})

	t.Run("unauthorized user", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, AssignedTo: nil}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)

		err := service.CompleteTask(ctx, 1, 3) // Neither creator nor assignee

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.CompleteTask(ctx, 999, 1)

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
	})
}

// ==================== UpdateTaskStatus Tests ====================

func TestUpdateTaskStatus(t *testing.T) {
	t.Run("open to in_progress", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(t *models.Task) bool {
			return t.Status == "in_progress"
		})).Return(nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "in_progress", 1)

		assert.NoError(t, err)
		assert.Equal(t, "in_progress", result.Status)
	})

	t.Run("in_progress to completed", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(t *models.Task) bool {
			return t.Status == "completed" && t.CompletedAt != nil
		})).Return(nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "completed", 1)

		assert.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
	})

	t.Run("cancelled to open - reopen", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignedTo := uint(2)
		task := &models.Task{ID: 1, CreatedBy: 1, Status: "cancelled", AssignedTo: &assignedTo}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(t *models.Task) bool {
			return t.Status == "open" && t.AssignedTo == nil && t.CompletedAt == nil
		})).Return(nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "open", 1)

		assert.NoError(t, err)
		assert.Equal(t, "open", result.Status)
	})

	t.Run("invalid transition - open to completed", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "completed", 1)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidStatus, err)
		assert.Nil(t, result)
	})

	t.Run("unauthorized - non-creator setting to in_progress", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "in_progress", 2) // Not creator

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		assert.Nil(t, result)
	})

	t.Run("assignee can mark completed", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignedTo := uint(2)
		task := &models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignedTo, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)
		taskRepo.On("Update", ctx, mock.Anything).Return(nil)

		result, err := service.UpdateTaskStatus(ctx, 1, "completed", 2) // Assignee

		assert.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
	})
}

// ==================== DeclineApplication Tests ====================

func TestDeclineApplication(t *testing.T) {
	t.Run("successful decline", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		application := &models.Application{ID: 1, Status: "pending"}
		appRepo.On("GetByID", ctx, uint(1)).Return(application, nil)
		appRepo.On("Update", ctx, mock.MatchedBy(func(a *models.Application) bool {
			return a.Status == "declined"
		})).Return(nil)

		err := service.DeclineApplication(ctx, 1)

		assert.NoError(t, err)
		appRepo.AssertExpectations(t)
	})

	t.Run("application not found", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		appRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.DeclineApplication(ctx, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrApplicationNotFound, err)
	})

	t.Run("cannot decline non-pending application", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		application := &models.Application{ID: 1, Status: "accepted"}
		appRepo.On("GetByID", ctx, uint(1)).Return(application, nil)

		err := service.DeclineApplication(ctx, 1)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pending")
	})
}

// ==================== GetTaskApplications Tests ====================

func TestGetTaskApplications(t *testing.T) {
	t.Run("successful get applications", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		applications := []models.Application{
			{ID: 1, TaskID: 1, ApplicantID: 2},
			{ID: 2, TaskID: 1, ApplicantID: 3},
		}
		appRepo.On("ListByTask", ctx, uint(1)).Return(applications, nil)

		result, err := service.GetTaskApplications(ctx, 1)

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		appRepo.AssertExpectations(t)
	})
}

// ==================== ListUserTasks Tests ====================

func TestListUserTasks(t *testing.T) {
	t.Run("list user created tasks", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		tasks := []models.Task{
			{ID: 1, CreatedBy: 1, Title: "Task 1"},
		}
		taskRepo.On("ListByUser", ctx, uint(1), "creator").Return(tasks, nil)

		result, err := service.ListUserTasks(ctx, 1, "creator")

		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("list user assigned tasks", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignedTo := uint(1)
		tasks := []models.Task{
			{ID: 2, AssignedTo: &assignedTo, Title: "Assigned Task"},
		}
		taskRepo.On("ListByUser", ctx, uint(1), "assignee").Return(tasks, nil)

		result, err := service.ListUserTasks(ctx, 1, "assignee")

		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
}
