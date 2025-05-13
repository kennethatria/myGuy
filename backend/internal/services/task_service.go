package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"time"
)

var (
	ErrTaskNotFound    = errors.New("task not found")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidDeadline = errors.New("deadline must be at least 5 minutes in the future")
	ErrTaskNotOpen     = errors.New("task is not open for applications")
)

type TaskService struct {
	taskRepo        repositories.TaskRepository
	applicationRepo repositories.ApplicationRepository
}

func NewTaskService(taskRepo repositories.TaskRepository, applicationRepo repositories.ApplicationRepository) *TaskService {
	return &TaskService{
		taskRepo:        taskRepo,
		applicationRepo: applicationRepo,
	}
}

type CreateTaskInput struct {
	Title       string
	Description string
	Fee         float64
	Deadline    time.Time
	CreatedBy   uint
}

type UpdateTaskInput struct {
	ID          uint
	Title       string
	Description string
	Fee         float64
	Deadline    time.Time
	UpdatedBy   uint
}

func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
	// Add a small buffer to allow for processing time
	if input.Deadline.Before(time.Now().Add(time.Minute * 5)) {
		return nil, ErrInvalidDeadline
	}

	task := &models.Task{
		Title:       input.Title,
		Description: input.Description,
		Fee:         input.Fee,
		Deadline:    input.Deadline,
		CreatedBy:   input.CreatedBy,
		Status:      "open",
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, input UpdateTaskInput) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if task.CreatedBy != input.UpdatedBy {
		return nil, ErrUnauthorized
	}

	if input.Deadline.Before(time.Now()) {
		return nil, ErrInvalidDeadline
	}

	task.Title = input.Title
	task.Description = input.Description
	task.Fee = input.Fee
	task.Deadline = input.Deadline

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, taskID uint) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// GetTaskByID is an alias for GetTask to match the handler expectations
func (s *TaskService) GetTaskByID(ctx context.Context, taskID uint) (*models.Task, error) {
	return s.GetTask(ctx, taskID)
}

func (s *TaskService) DeleteTask(ctx context.Context, taskID uint, userID uint) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	if task.CreatedBy != userID {
		return ErrUnauthorized
	}

	return s.taskRepo.Delete(ctx, taskID)
}

func (s *TaskService) ListTasks(ctx context.Context, filters map[string]interface{}) ([]models.Task, error) {
	return s.taskRepo.List(ctx, filters)
}

func (s *TaskService) ListUserTasks(ctx context.Context, userID uint, role string) ([]models.Task, error) {
	return s.taskRepo.ListByUser(ctx, userID, role)
}

func (s *TaskService) ApplyForTask(ctx context.Context, taskID, applicantID uint, proposedFee float64, message string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	if task.Status != "open" {
		return ErrTaskNotOpen
	}

	application := &models.Application{
		TaskID:      taskID,
		ApplicantID: applicantID,
		ProposedFee: proposedFee,
		Message:     message,
		Status:      "pending",
	}

	return s.applicationRepo.Create(ctx, application)
}

func (s *TaskService) AssignTask(ctx context.Context, taskID, applicationID uint, creatorID uint) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	if task.CreatedBy != creatorID {
		return ErrUnauthorized
	}

	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return err
	}

	task.Status = "in_progress"
	task.AssignedTo = &application.ApplicantID
	task.Fee = application.ProposedFee

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}

	application.Status = "accepted"
	return s.applicationRepo.Update(ctx, application)
}

func (s *TaskService) CompleteTask(ctx context.Context, taskID uint, userID uint) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	if task.CreatedBy != userID && *task.AssignedTo != userID {
		return ErrUnauthorized
	}

	task.Status = "completed"
	return s.taskRepo.Update(ctx, task)
}
