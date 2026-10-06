package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"myguy/internal/chatnotify"
	"myguy/internal/models"
	"myguy/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTaskService() (*TaskService, *tests.MockTaskRepository, *tests.MockApplicationRepository) {
	taskRepo := new(tests.MockTaskRepository)
	appRepo := new(tests.MockApplicationRepository)
	service := NewTaskService(taskRepo, appRepo, nil)
	return service, taskRepo, appRepo
}

// recordingNotifier captures task event messages instead of posting them.
type recordingNotifier struct{ sent []chatnotify.Message }

func (r *recordingNotifier) Post(m chatnotify.Message) {
	r.sent = append(r.sent, m)
}

// ==================== CreateTask Tests ====================

func TestCreateTask(t *testing.T) {
	t.Run("posts a note that runs for 24 hours", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("Create", ctx, mock.MatchedBy(func(task *models.Task) bool {
			return task.Title == "Paint my fence" &&
				task.Description == "Small garden fence, white paint provided." &&
				task.Fee == 0 &&
				task.Status == "open"
		})).Return(nil)

		before := time.Now()
		task, err := service.CreateTask(ctx, CreateTaskInput{
			Title:       "  Paint my fence ",
			Description: "Small garden fence, white paint provided.",
			CreatedBy:   1,
		})

		assert.NoError(t, err)
		assert.WithinDuration(t, before.Add(24*time.Hour), task.Deadline, time.Minute)
		taskRepo.AssertExpectations(t)
	})

	t.Run("enforces the sticky-note limits", func(t *testing.T) {
		cases := map[string]struct {
			title, description string
			want               error
		}{
			"no headline":          {"", "body", ErrHeadlineRequired},
			"no body":              {"Paint fence", "   ", ErrBodyRequired},
			"six-word headline":    {"one two three four five six", "body", ErrHeadlineTooLong},
			"overlong single word": {strings.Repeat("a", 61), "body", ErrHeadlineTooLong},
			"21-word body":         {"Paint fence", strings.TrimSpace(strings.Repeat("word ", 21)), ErrBodyTooLong},
			"phone number":         {"Paint fence", "call 0772 123 456", ErrContactDetails},
			"email in headline":    {"mail john@gmail.com", "body", ErrContactDetails},
		}
		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				service, taskRepo, _ := setupTaskService()

				task, err := service.CreateTask(context.Background(), CreateTaskInput{Title: tc.title, Description: tc.description})

				assert.ErrorIs(t, err, tc.want)
				assert.True(t, IsGigTextError(err))
				assert.Nil(t, task)
				taskRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
			})
		}
	})

	t.Run("five-word headline and twenty-word body fit", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		taskRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

		_, err := service.CreateTask(context.Background(), CreateTaskInput{
			Title:       "one two three four five",
			Description: strings.TrimSpace(strings.Repeat("word ", 20)),
		})

		assert.NoError(t, err)
	})

	t.Run("repository create error", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("Create", ctx, mock.Anything).Return(errors.New("database error"))

		task, err := service.CreateTask(ctx, CreateTaskInput{Title: "Paint fence", Description: "Today"})

		assert.Error(t, err)
		assert.False(t, IsGigTextError(err))
		assert.Nil(t, task)
	})
}

// ==================== UpdateTask Tests ====================

func TestUpdateTask(t *testing.T) {
	t.Run("successful update keeps the deadline", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		deadline := time.Now().Add(5 * time.Hour)
		existingTask := &models.Task{ID: 1, Title: "Old Title", CreatedBy: 1, Status: "open", Deadline: deadline}

		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)
		taskRepo.On("Update", ctx, mock.MatchedBy(func(task *models.Task) bool {
			return task.Title == "New Title" && task.Deadline.Equal(deadline)
		})).Return(nil)

		task, err := service.UpdateTask(ctx, UpdateTaskInput{ID: 1, Title: "New Title", Description: "New note", UpdatedBy: 1})

		assert.NoError(t, err)
		assert.Equal(t, "New Title", task.Title)
		taskRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		task, err := service.UpdateTask(ctx, UpdateTaskInput{ID: 999, UpdatedBy: 1})

		assert.Equal(t, ErrTaskNotFound, err)
		assert.Nil(t, task)
	})

	t.Run("unauthorized - not owner", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1}, nil)

		task, err := service.UpdateTask(ctx, UpdateTaskInput{ID: 1, Title: "T", Description: "D", UpdatedBy: 2})

		assert.Equal(t, ErrUnauthorized, err)
		assert.Nil(t, task)
	})

	t.Run("edits follow the same limits", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1}, nil)

		_, err := service.UpdateTask(ctx, UpdateTaskInput{ID: 1, Title: "Paint fence", Description: "www.myshop.ug", UpdatedBy: 1})

		assert.ErrorIs(t, err, ErrContactDetails)
		taskRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
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
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		existingTask := &models.Task{ID: 1, CreatedBy: 1}
		taskRepo.On("GetByID", ctx, uint(1)).Return(existingTask, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
		taskRepo.On("Delete", ctx, uint(1)).Return(nil)

		err := service.DeleteTask(ctx, 1, 1)

		assert.NoError(t, err)
		taskRepo.AssertExpectations(t)
	})

	t.Run("removing a gig tells whoever was still waiting", func(t *testing.T) {
		taskRepo := new(tests.MockTaskRepository)
		appRepo := new(tests.MockApplicationRepository)
		notifier := &recordingNotifier{}
		service := NewTaskService(taskRepo, appRepo, notifier)
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Walk dog", CreatedBy: 1, Status: "open"}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{
			{ID: 5, ApplicantID: 7, Status: "pending"},
			{ID: 6, ApplicantID: 8, Status: "declined"},
		}, nil)
		taskRepo.On("Delete", ctx, uint(1)).Return(nil)

		assert.NoError(t, service.DeleteTask(ctx, 1, 1))

		assert.Equal(t, []chatnotify.Message{{
			TaskID: 1, SenderID: 1, RecipientID: 7,
			Content: `"Walk dog" was removed by the poster, so your application is closed.`,
			Event:   chatnotify.EventCancelled,
		}}, notifier.sent)
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

func TestDeleteTaskRules(t *testing.T) {
	t.Run("a task that was assigned cannot be deleted", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignee := uint(2)
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignee, Status: "cancelled"}, nil)

		err := service.DeleteTask(ctx, 1, 1)

		assert.Equal(t, ErrTaskWasAssigned, err)
		taskRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	})
}

func TestCancelTaskDeclinesPendingApplications(t *testing.T) {
	taskRepo := new(tests.MockTaskRepository)
	appRepo := new(tests.MockApplicationRepository)
	notifier := &recordingNotifier{}
	service := NewTaskService(taskRepo, appRepo, notifier)
	ctx := context.Background()

	taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Paint fence", CreatedBy: 1, Status: "open"}, nil)
	taskRepo.On("Update", ctx, mock.Anything).Return(nil)
	appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{{ID: 5, ApplicantID: 7, Status: "pending"}}, nil)
	appRepo.On("DeclinePending", ctx, uint(1), uint(0)).Return(nil)

	task, err := service.UpdateTaskStatus(ctx, 1, "cancelled", 1)

	assert.NoError(t, err)
	assert.Equal(t, "cancelled", task.Status)
	appRepo.AssertExpectations(t)
	assert.Equal(t, []chatnotify.Message{{TaskID: 1, SenderID: 1, RecipientID: 7, Content: `"Paint fence" was cancelled by the poster, so your application is closed.`, Event: chatnotify.EventCancelled}}, notifier.sent)
}

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

		openTask := &models.Task{ID: 1, Status: "open", CreatedBy: 9}
		taskRepo.On("GetByID", ctx, uint(1)).Return(openTask, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{{ApplicantID: 3}}, nil)
		appRepo.On("Create", ctx, mock.MatchedBy(func(app *models.Application) bool {
			return app.TaskID == 1 &&
				app.ApplicantID == 2 &&
				app.Status == "pending"
		})).Return(nil)

		err := service.ApplyForTask(ctx, 1, 2, "I can do this")

		assert.NoError(t, err)
		taskRepo.AssertExpectations(t)
		appRepo.AssertExpectations(t)
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.ApplyForTask(ctx, 999, 2, "Message")

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotFound, err)
	})

	t.Run("task not open", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		closedTask := &models.Task{ID: 1, Status: "in_progress"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(closedTask, nil)

		err := service.ApplyForTask(ctx, 1, 2, "Message")

		assert.Error(t, err)
		assert.Equal(t, ErrTaskNotOpen, err)
	})

	t.Run("cannot apply to own task", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open", CreatedBy: 2}, nil)

		err := service.ApplyForTask(ctx, 1, 2, "Message")

		assert.Equal(t, ErrOwnTask, err)
		appRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("cannot apply twice", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open", CreatedBy: 9}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{{ApplicantID: 2}}, nil)

		err := service.ApplyForTask(ctx, 1, 2, "Again")

		assert.Equal(t, ErrAlreadyApplied, err)
		appRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

func TestApplyForTaskGigRules(t *testing.T) {
	t.Run("past 24 hours with no applications counts as expired", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open", CreatedBy: 9, Deadline: time.Now().Add(-time.Minute)}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)

		err := service.ApplyForTask(ctx, 1, 2, "Interested")

		assert.Equal(t, ErrTaskNotOpen, err)
		appRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	t.Run("a gig someone applied to stays open past 24 hours", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open", CreatedBy: 9, Deadline: time.Now().Add(-time.Hour)}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{{ApplicantID: 3}}, nil)
		appRepo.On("Create", ctx, mock.Anything).Return(nil)

		assert.NoError(t, service.ApplyForTask(ctx, 1, 2, "Interested"))
	})

	t.Run("no contact details before a match", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()

		err := service.ApplyForTask(context.Background(), 1, 2, "whatsapp me on 0772123456")

		assert.Equal(t, ErrContactDetails, err)
		taskRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	})

	t.Run("message length is capped", func(t *testing.T) {
		service, _, _ := setupTaskService()

		err := service.ApplyForTask(context.Background(), 1, 2, strings.Repeat("a", 501))

		assert.Equal(t, ErrMessageTooLong, err)
	})
}

func TestExpireStaleTasks(t *testing.T) {
	service, taskRepo, _ := setupTaskService()
	ctx := context.Background()

	before := time.Now()
	taskRepo.On("ExpireUnanswered", ctx, mock.MatchedBy(func(now time.Time) bool {
		return !now.Before(before.UTC().Add(-time.Second))
	})).Return(int64(3), nil)

	n, err := service.ExpireStaleTasks(ctx)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), n)
}

func TestRepostExpiredTask(t *testing.T) {
	t.Run("expired gig can be reposted for a fresh 24 hours", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1, Status: "expired", Deadline: time.Now().Add(-time.Hour)}, nil)
		taskRepo.On("Update", ctx, mock.Anything).Return(nil)

		task, err := service.UpdateTaskStatus(ctx, 1, "open", 1)

		assert.NoError(t, err)
		assert.Equal(t, "open", task.Status)
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), task.Deadline, time.Minute)
	})

	t.Run("expired gig can be cancelled but not started", func(t *testing.T) {
		for status, ok := range map[string]bool{"cancelled": true, "in_progress": false, "completed": false} {
			service, taskRepo, appRepo := setupTaskService()
			ctx := context.Background()
			taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1, Status: "expired"}, nil)
			taskRepo.On("Update", ctx, mock.Anything).Return(nil)
			appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
			appRepo.On("DeclinePending", ctx, uint(1), uint(0)).Return(nil)

			_, err := service.UpdateTaskStatus(ctx, 1, status, 1)

			if ok {
				assert.NoError(t, err, status)
			} else {
				assert.Equal(t, ErrInvalidStatus, err, status)
			}
		}
	})
}

// ==================== AssignTask Tests ====================

func TestAssignTask(t *testing.T) {
	pending := func() *models.Application {
		return &models.Application{ID: 1, TaskID: 1, ApplicantID: 2, Status: "pending"}
	}

	t.Run("successful assignment declines the other applicants", func(t *testing.T) {
		taskRepo := new(tests.MockTaskRepository)
		appRepo := new(tests.MockApplicationRepository)
		notifier := &recordingNotifier{}
		service := NewTaskService(taskRepo, appRepo, notifier)
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Paint fence", CreatedBy: 9, Status: "open", Fee: 100}, nil)
		appRepo.On("GetByID", ctx, uint(1)).Return(pending(), nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{
			{ID: 1, ApplicantID: 2, Status: "pending"},
			{ID: 2, ApplicantID: 3, Status: "pending"},
			{ID: 3, ApplicantID: 4, Status: "declined"},
		}, nil)
		taskRepo.On("AssignIfOpen", ctx, uint(1), uint(2)).Return(true, nil)
		appRepo.On("Update", ctx, mock.MatchedBy(func(a *models.Application) bool {
			return a.Status == "accepted"
		})).Return(nil)
		appRepo.On("DeclinePending", ctx, uint(1), uint(1)).Return(nil)

		result, err := service.AssignTask(ctx, 1, 1)

		assert.NoError(t, err)
		assert.Equal(t, "in_progress", result.Status)
		assert.Equal(t, uint(2), *result.AssignedTo)
		taskRepo.AssertExpectations(t)
		appRepo.AssertExpectations(t)

		// The chosen applicant hears they got it; the other pending one that
		// they didn't; the already-declined one hears nothing new.
		assert.Len(t, notifier.sent, 2)
		assert.Equal(t, uint(2), notifier.sent[0].RecipientID)
		assert.Contains(t, notifier.sent[0].Content, "was accepted")
		assert.Equal(t, chatnotify.EventAccepted, notifier.sent[0].Event)
		assert.True(t, notifier.sent[0].UnlockContacts, "acceptance unlocks chat and contact sharing")
		assert.False(t, notifier.sent[1].UnlockContacts)
		assert.Equal(t, uint(3), notifier.sent[1].RecipientID)
		assert.Contains(t, notifier.sent[1].Content, "wasn't selected")
		assert.Equal(t, chatnotify.EventDeclined, notifier.sent[1].Event)
		for _, m := range notifier.sent {
			assert.Equal(t, uint(9), m.SenderID, "sent as the task owner")
		}
	})

	t.Run("task not found", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		result, err := service.AssignTask(ctx, 999, 1)

		assert.Equal(t, ErrTaskNotFound, err)
		assert.Nil(t, result)
	})

	t.Run("already assigned task cannot be reassigned", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "in_progress"}, nil)

		_, err := service.AssignTask(ctx, 1, 1)

		assert.Equal(t, ErrTaskNotOpen, err)
		appRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("application from another task is rejected", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		other := pending()
		other.TaskID = 7
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open"}, nil)
		appRepo.On("GetByID", ctx, uint(1)).Return(other, nil)

		_, err := service.AssignTask(ctx, 1, 1)

		assert.Equal(t, ErrApplicationNotFound, err)
	})

	t.Run("declined application cannot be accepted", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		declined := pending()
		declined.Status = "declined"
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open"}, nil)
		appRepo.On("GetByID", ctx, uint(1)).Return(declined, nil)

		_, err := service.AssignTask(ctx, 1, 1)

		assert.Equal(t, ErrApplicationNotPending, err)
	})

	t.Run("losing a concurrent acceptance changes nothing", func(t *testing.T) {
		service, taskRepo, appRepo := setupTaskService()
		ctx := context.Background()

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Status: "open"}, nil)
		appRepo.On("GetByID", ctx, uint(1)).Return(pending(), nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
		taskRepo.On("AssignIfOpen", ctx, uint(1), uint(2)).Return(false, nil)

		_, err := service.AssignTask(ctx, 1, 1)

		assert.Equal(t, ErrTaskNotOpen, err)
		appRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})
}

// ==================== UpdateTaskStatus Tests ====================

func TestUpdateTaskStatus(t *testing.T) {
	t.Run("open to in_progress without an assignee is rejected", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		task := &models.Task{ID: 1, CreatedBy: 1, Status: "open"}
		taskRepo.On("GetByID", ctx, uint(1)).Return(task, nil)

		_, err := service.UpdateTaskStatus(ctx, 1, "in_progress", 1)

		assert.Equal(t, ErrInvalidStatus, err)
		taskRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
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

	t.Run("assignee can't complete a gig themselves", func(t *testing.T) {
		service, taskRepo, _ := setupTaskService()
		ctx := context.Background()

		assignedTo := uint(2)
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignedTo, Status: "in_progress"}, nil)

		_, err := service.UpdateTaskStatus(ctx, 1, "completed", 2)

		assert.Equal(t, ErrUnauthorized, err)
	})
}

func TestCompletionNeedsApproval(t *testing.T) {
	ctx := context.Background()
	assignee := uint(2)
	// A gig for poster 1 and assignee 2 in status, and what it posts to chat
	setup := func(status string) (*TaskService, *recordingNotifier) {
		taskRepo := new(tests.MockTaskRepository)
		notifier := &recordingNotifier{}
		service := NewTaskService(taskRepo, new(tests.MockApplicationRepository), notifier)
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Fix sink", CreatedBy: 1, AssignedTo: &assignee, Status: status}, nil)
		taskRepo.On("Update", ctx, mock.Anything).Return(nil)
		return service, notifier
	}

	t.Run("the assignee marks it done, which asks the poster", func(t *testing.T) {
		service, notifier := setup("in_progress")

		task, err := service.UpdateTaskStatus(ctx, 1, "pending_approval", 2)

		assert.NoError(t, err)
		assert.Equal(t, "pending_approval", task.Status)
		assert.Len(t, notifier.sent, 1)
		assert.Equal(t, chatnotify.EventDone, notifier.sent[0].Event)
		assert.Equal(t, uint(2), notifier.sent[0].SenderID)
		assert.Equal(t, uint(1), notifier.sent[0].RecipientID)
	})

	t.Run("only the assignee marks it done", func(t *testing.T) {
		service, _ := setup("in_progress")
		_, err := service.UpdateTaskStatus(ctx, 1, "pending_approval", 1)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("the poster approves it", func(t *testing.T) {
		service, notifier := setup("pending_approval")

		task, err := service.UpdateTaskStatus(ctx, 1, "completed", 1)

		assert.NoError(t, err)
		assert.Equal(t, "completed", task.Status)
		assert.NotNil(t, task.CompletedAt)
		assert.Equal(t, chatnotify.EventCompleted, notifier.sent[0].Event)
		assert.Equal(t, uint(2), notifier.sent[0].RecipientID)
	})

	t.Run("the poster says not yet", func(t *testing.T) {
		service, notifier := setup("pending_approval")

		task, err := service.UpdateTaskStatus(ctx, 1, "in_progress", 1)

		assert.NoError(t, err)
		assert.Equal(t, "in_progress", task.Status)
		assert.Equal(t, chatnotify.EventNotDone, notifier.sent[0].Event)
	})

	t.Run("the assignee can't approve their own work", func(t *testing.T) {
		service, _ := setup("pending_approval")
		_, err := service.UpdateTaskStatus(ctx, 1, "completed", 2)
		assert.Equal(t, ErrUnauthorized, err)
	})

	t.Run("cancelling a gig being done tells the assignee", func(t *testing.T) {
		for _, status := range []string{"in_progress", "pending_approval"} {
			taskRepo := new(tests.MockTaskRepository)
			appRepo := new(tests.MockApplicationRepository)
			notifier := &recordingNotifier{}
			service := NewTaskService(taskRepo, appRepo, notifier)
			taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Fix sink", CreatedBy: 1, AssignedTo: &assignee, Status: status}, nil)
			taskRepo.On("Update", ctx, mock.Anything).Return(nil)
			appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
			appRepo.On("DeclinePending", ctx, uint(1), uint(0)).Return(nil)

			_, err := service.UpdateTaskStatus(ctx, 1, "cancelled", 1)

			assert.NoError(t, err, status)
			assert.Equal(t, []chatnotify.Message{{
				TaskID: 1, SenderID: 1, RecipientID: 2, Content: `🚫 "Fix sink" was cancelled by the poster.`, Event: chatnotify.EventCancelled,
			}}, notifier.sent, status)
		}
	})

	t.Run("an open gig can't be marked done", func(t *testing.T) {
		taskRepo := new(tests.MockTaskRepository)
		service := NewTaskService(taskRepo, new(tests.MockApplicationRepository), nil)
		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, CreatedBy: 1, AssignedTo: &assignee, Status: "open"}, nil)
		_, err := service.UpdateTaskStatus(ctx, 1, "pending_approval", 2)
		assert.Equal(t, ErrInvalidStatus, err)
	})
}

type recordingUnlocker struct {
	pairs [][3]uint
	err   error
}

func (r *recordingUnlocker) Unlock(taskID, a, b uint) error {
	r.pairs = append(r.pairs, [3]uint{taskID, a, b})
	return r.err
}

func TestUnlockMatchedChats(t *testing.T) {
	ctx := context.Background()
	appRepo := new(tests.MockApplicationRepository)
	service := NewTaskService(new(tests.MockTaskRepository), appRepo, nil)
	appRepo.On("ListAccepted", ctx).Return([]models.Application{
		{TaskID: 1, ApplicantID: 2, Task: models.Task{CreatedBy: 9}},
		{TaskID: 3, ApplicantID: 4, Task: models.Task{CreatedBy: 9}},
	}, nil)

	unlocker := &recordingUnlocker{}
	n, err := service.UnlockMatchedChats(ctx, unlocker)

	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, [][3]uint{{1, 9, 2}, {3, 9, 4}}, unlocker.pairs)

	failing := &recordingUnlocker{err: errors.New("chat down")}
	_, err = service.UnlockMatchedChats(ctx, failing)
	assert.Error(t, err)
}

// ==================== DeclineApplication Tests ====================

func TestDeclineApplication(t *testing.T) {
	t.Run("successful decline", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		application := &models.Application{ID: 1, TaskID: 1, Status: "pending"}
		appRepo.On("GetByID", ctx, uint(1)).Return(application, nil)
		appRepo.On("Update", ctx, mock.MatchedBy(func(a *models.Application) bool {
			return a.Status == "declined"
		})).Return(nil)

		err := service.DeclineApplication(ctx, 1, 1)

		assert.NoError(t, err)
		appRepo.AssertExpectations(t)
	})

	t.Run("application not found", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		appRepo.On("GetByID", ctx, uint(999)).Return(nil, errors.New("not found"))

		err := service.DeclineApplication(ctx, 1, 999)

		assert.Error(t, err)
		assert.Equal(t, ErrApplicationNotFound, err)
	})

	t.Run("cannot decline non-pending application", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		application := &models.Application{ID: 1, TaskID: 1, Status: "accepted"}
		appRepo.On("GetByID", ctx, uint(1)).Return(application, nil)

		err := service.DeclineApplication(ctx, 1, 1)

		assert.Equal(t, ErrApplicationNotPending, err)
	})

	t.Run("application from another task is rejected", func(t *testing.T) {
		service, _, appRepo := setupTaskService()
		ctx := context.Background()

		appRepo.On("GetByID", ctx, uint(1)).Return(&models.Application{ID: 1, TaskID: 7, Status: "pending"}, nil)

		err := service.DeclineApplication(ctx, 1, 1)

		assert.Equal(t, ErrApplicationNotFound, err)
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

func TestApplyAndDeclineNotify(t *testing.T) {
	ctx := context.Background()

	t.Run("applying tells the owner, with the message", func(t *testing.T) {
		taskRepo := new(tests.MockTaskRepository)
		appRepo := new(tests.MockApplicationRepository)
		notifier := &recordingNotifier{}
		service := NewTaskService(taskRepo, appRepo, notifier)

		taskRepo.On("GetByID", ctx, uint(1)).Return(&models.Task{ID: 1, Title: "Paint fence", Status: "open", CreatedBy: 9, Deadline: time.Now().Add(time.Hour)}, nil)
		appRepo.On("ListByTask", ctx, uint(1)).Return([]models.Application{}, nil)
		appRepo.On("Create", ctx, mock.Anything).Return(nil)

		assert.NoError(t, service.ApplyForTask(ctx, 1, 2, "  I have a van  "))

		assert.Equal(t, []chatnotify.Message{{
			TaskID: 1, SenderID: 2, RecipientID: 9, Content: "📩 New application for \"Paint fence\".\n\nI have a van",
			Event: chatnotify.EventApplication,
		}}, notifier.sent)
	})

	t.Run("declining tells the applicant", func(t *testing.T) {
		taskRepo := new(tests.MockTaskRepository)
		appRepo := new(tests.MockApplicationRepository)
		notifier := &recordingNotifier{}
		service := NewTaskService(taskRepo, appRepo, notifier)

		appRepo.On("GetByID", ctx, uint(4)).Return(&models.Application{ID: 4, TaskID: 1, ApplicantID: 2, Status: "pending", Task: models.Task{ID: 1, Title: "Paint fence", CreatedBy: 9}}, nil)
		appRepo.On("Update", ctx, mock.Anything).Return(nil)

		assert.NoError(t, service.DeclineApplication(ctx, 1, 4))

		assert.Equal(t, []chatnotify.Message{{
			TaskID: 1, SenderID: 9, RecipientID: 2, Content: "Your application for \"Paint fence\" wasn't selected this time.",
			Event: chatnotify.EventDeclined, ApplicationID: 4,
		}}, notifier.sent)
	})
}

