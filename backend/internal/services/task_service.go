package services

import (
	"context"
	"errors"
	"fmt"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"time"
)

var (
	ErrTaskNotFound        = errors.New("task not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidDeadline     = errors.New("deadline must be at least one day (24 hours) in the future")
	ErrTaskNotOpen         = errors.New("task is not open for applications")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrApplicationNotFound = errors.New("application not found")
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
	// Compare dates in UTC
	now := time.Now().UTC()
	minDeadline := now.AddDate(0, 0, 1) // Add 1 day to current time
	deadline := input.Deadline.UTC()

	fmt.Printf("CreateTask: Now=%v, MinDeadline=%v, ProvidedDeadline=%v\n", 
		now, minDeadline, deadline)

	if deadline.Before(minDeadline) {
		fmt.Printf("Validation error: Deadline (%v) is before minimum deadline (%v)\n", 
			deadline, minDeadline)
		return nil, ErrInvalidDeadline
	}

	task := &models.Task{
		Title:       input.Title,
		Description: input.Description,
		Fee:         input.Fee,
		Deadline:    deadline,
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

	// Require deadline to be at least one day in the future
	now := time.Now().UTC()
	minDeadline := now.AddDate(0, 0, 1)
	
	if input.Deadline.UTC().Before(minDeadline) {
		fmt.Printf("Validation error: Deadline (%v) is before minimum deadline (%v)\n", 
			input.Deadline.UTC(), minDeadline)
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

type PaginatedTasksResult struct {
	Tasks      []models.Task `json:"tasks"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

func (s *TaskService) ListTasksWithPagination(ctx context.Context, filters map[string]interface{}) (*PaginatedTasksResult, error) {
	// Extract pagination params
	page, _ := filters["page"].(int)
	perPage, _ := filters["per_page"].(int)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	
	// Get total count
	total, err := s.taskRepo.Count(ctx, filters)
	if err != nil {
		return nil, err
	}
	
	// Get paginated tasks
	tasks, err := s.taskRepo.ListWithPagination(ctx, filters)
	if err != nil {
		return nil, err
	}
	
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	
	return &PaginatedTasksResult{
		Tasks:      tasks,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
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

func (s *TaskService) AssignTask(ctx context.Context, taskID, applicationID uint) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, err
	}

	task.Status = "in_progress"
	task.AssignedTo = &application.ApplicantID
	task.Fee = application.ProposedFee

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	application.Status = "accepted"
	if err := s.applicationRepo.Update(ctx, application); err != nil {
		return nil, err
	}
	
	return task, nil
}

func (s *TaskService) CompleteTask(ctx context.Context, taskID uint, userID uint) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	if task.CreatedBy != userID && (task.AssignedTo == nil || *task.AssignedTo != userID) {
		return ErrUnauthorized
	}

	task.Status = "completed"
	now := time.Now()
	task.CompletedAt = &now
	return s.taskRepo.Update(ctx, task)
}

// UpdateTaskStatus updates the status of a task
// Task creator can update any status, assigned users can only mark as completed
func (s *TaskService) UpdateTaskStatus(ctx context.Context, taskID uint, status string, userID uint) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	// Check authorization based on the status being set
	if status == "completed" {
		// Both task creator and assigned user can mark as completed
		if task.CreatedBy != userID && (task.AssignedTo == nil || *task.AssignedTo != userID) {
			return nil, ErrUnauthorized
		}
	} else {
		// Only task creator can change to other statuses
		if task.CreatedBy != userID {
			return nil, ErrUnauthorized
		}
	}

	// Validate status transitions
	// This enforces a simple workflow where tasks generally move forward
	// open -> in_progress -> completed
	// But creator can also cancel a task or reopen it
	validTransition := false
	switch task.Status {
	case "open":
		// From open: can move to in_progress or cancelled
		validTransition = status == "in_progress" || status == "cancelled"
	case "in_progress":
		// From in_progress: can move to completed or cancelled
		validTransition = status == "completed" || status == "cancelled"
	case "completed":
		// From completed: can move to cancelled
		validTransition = status == "cancelled"
	case "cancelled":
		// From cancelled: can move to open (reopen)
		validTransition = status == "open"
	}

	if !validTransition {
		return nil, ErrInvalidStatus
	}

	// Update the status
	task.Status = status

	// Set completed timestamp when task is completed
	if status == "completed" {
		now := time.Now()
		task.CompletedAt = &now
	}

	// If moving back to open, clear any assignments and completed timestamp
	if status == "open" {
		task.AssignedTo = nil
		task.CompletedAt = nil
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	return task, nil
}


// DeclineApplication updates an application status to declined
func (s *TaskService) DeclineApplication(ctx context.Context, applicationID uint) error {
	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return ErrApplicationNotFound
	}

	if application.Status != "pending" {
		return errors.New("can only decline pending applications")
	}

	application.Status = "declined"
	application.UpdatedAt = time.Now()
	return s.applicationRepo.Update(ctx, application)
}

// GetTaskApplications returns all applications for a given task
func (s *TaskService) GetTaskApplications(ctx context.Context, taskID uint) ([]models.Application, error) {
	return s.applicationRepo.ListByTask(ctx, taskID)
}
