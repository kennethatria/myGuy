package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"strconv"
	"strings"
	"time"
)

var (
	ErrTaskNotFound        = errors.New("task not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidDeadline     = errors.New("deadline must be at least one day (24 hours) in the future")
	ErrTaskNotOpen         = errors.New("task is not open for applications")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrApplicationNotFound = errors.New("application not found")
	ErrApplicationNotPending = errors.New("application is no longer pending")
	ErrOwnTask             = errors.New("you cannot apply to your own task")
	ErrAlreadyApplied      = errors.New("you have already applied to this task")
	ErrTaskWasAssigned     = errors.New("a task that was assigned can't be deleted; cancel it instead")
)

// TaskNotifier tells the people involved about task events by posting a
// message into their conversation about the task. Implementations must not
// block or fail the caller (chat is best effort).
type TaskNotifier interface {
	TaskMessage(taskID, senderID, recipientID uint, content string)
}

type noopNotifier struct{}

func (noopNotifier) TaskMessage(uint, uint, uint, string) {}

type TaskService struct {
	taskRepo        repositories.TaskRepository
	applicationRepo repositories.ApplicationRepository
	notifier        TaskNotifier
}

// NewTaskService builds the service; notifier may be nil (no notifications).
func NewTaskService(taskRepo repositories.TaskRepository, applicationRepo repositories.ApplicationRepository, notifier TaskNotifier) *TaskService {
	if notifier == nil {
		notifier = noopNotifier{}
	}
	return &TaskService{
		taskRepo:        taskRepo,
		applicationRepo: applicationRepo,
		notifier:        notifier,
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

	// Assigned tasks carry work history and possibly reviews (which affect
	// ratings); keep them and let the owner cancel instead.
	if task.AssignedTo != nil {
		return ErrTaskWasAssigned
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
	if task.CreatedBy == applicantID {
		return ErrOwnTask
	}

	existing, err := s.applicationRepo.ListByTask(ctx, taskID)
	if err != nil {
		return err
	}
	for _, app := range existing {
		if app.ApplicantID == applicantID {
			return ErrAlreadyApplied
		}
	}

	application := &models.Application{
		TaskID:      taskID,
		ApplicantID: applicantID,
		ProposedFee: proposedFee,
		Message:     message,
		Status:      "pending",
	}

	if err := s.applicationRepo.Create(ctx, application); err != nil {
		return err
	}

	content := fmt.Sprintf("📩 New application for \"%s\": UGX %s proposed.", task.Title, formatUGX(proposedFee))
	if trimmed := strings.TrimSpace(message); trimmed != "" {
		content += "\n\n" + trimmed
	}
	s.notifier.TaskMessage(taskID, applicantID, task.CreatedBy, content)
	return nil
}

// AssignTask accepts an application: the applicant gets the task at their
// proposed fee and every other pending application is declined.
func (s *TaskService) AssignTask(ctx context.Context, taskID, applicationID uint) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	if task.Status != "open" {
		return nil, ErrTaskNotOpen
	}

	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil || application.TaskID != taskID {
		return nil, ErrApplicationNotFound
	}
	if application.Status != "pending" {
		return nil, ErrApplicationNotPending
	}

	// Who is still waiting, to tell them once someone else is chosen.
	others, err := s.pendingApplicants(ctx, taskID, applicationID)
	if err != nil {
		return nil, err
	}

	// Conditional update: only one acceptance can win, even when racing.
	assigned, err := s.taskRepo.AssignIfOpen(ctx, taskID, application.ApplicantID, application.ProposedFee)
	if err != nil {
		return nil, err
	}
	if !assigned {
		return nil, ErrTaskNotOpen
	}

	application.Status = "accepted"
	if err := s.applicationRepo.Update(ctx, application); err != nil {
		return nil, err
	}
	if err := s.applicationRepo.DeclinePending(ctx, taskID, applicationID); err != nil {
		return nil, err
	}

	s.notifier.TaskMessage(taskID, task.CreatedBy, application.ApplicantID,
		fmt.Sprintf("✅ Your application for \"%s\" was accepted at UGX %s. Use this chat to arrange the details.", task.Title, formatUGX(application.ProposedFee)))
	for _, applicantID := range others {
		s.notifier.TaskMessage(taskID, task.CreatedBy, applicantID,
			fmt.Sprintf("Your application for \"%s\" wasn't selected this time.", task.Title))
	}

	task.Status = "in_progress"
	task.AssignedTo = &application.ApplicantID
	task.Fee = application.ProposedFee
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
		// From open: cancel. Starting work happens by accepting an
		// application (AssignTask), so a task is never in progress unassigned.
		validTransition = status == "cancelled" || (status == "in_progress" && task.AssignedTo != nil)
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

	// A cancelled task no longer needs anyone: tell waiting applicants.
	if status == "cancelled" {
		waiting, err := s.pendingApplicants(ctx, taskID, 0)
		if err != nil {
			return nil, err
		}
		if err := s.applicationRepo.DeclinePending(ctx, taskID, 0); err != nil {
			return nil, err
		}
		for _, applicantID := range waiting {
			s.notifier.TaskMessage(taskID, task.CreatedBy, applicantID,
				fmt.Sprintf("\"%s\" was cancelled by the poster, so your application is closed.", task.Title))
		}
	}

	return task, nil
}


// DeclineApplication declines a pending application to taskID.
func (s *TaskService) DeclineApplication(ctx context.Context, taskID, applicationID uint) error {
	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil || application.TaskID != taskID {
		return ErrApplicationNotFound
	}

	if application.Status != "pending" {
		return ErrApplicationNotPending
	}

	application.Status = "declined"
	application.UpdatedAt = time.Now()
	if err := s.applicationRepo.Update(ctx, application); err != nil {
		return err
	}

	s.notifier.TaskMessage(taskID, application.Task.CreatedBy, application.ApplicantID,
		fmt.Sprintf("Your application for \"%s\" wasn't selected this time.", application.Task.Title))
	return nil
}

// pendingApplicants lists the applicants still waiting on taskID, except
// the application exceptID.
func (s *TaskService) pendingApplicants(ctx context.Context, taskID, exceptID uint) ([]uint, error) {
	applications, err := s.applicationRepo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	var ids []uint
	for _, app := range applications {
		if app.Status == "pending" && app.ID != exceptID {
			ids = append(ids, app.ApplicantID)
		}
	}
	return ids, nil
}

// formatUGX renders 120000 as "120,000".
func formatUGX(amount float64) string {
	digits := strconv.FormatInt(int64(math.Round(amount)), 10)
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return digits
}

// ApplicationParticipants are the two people who may chat about an application.
type ApplicationParticipants struct {
	ApplicationID uint `json:"application_id"`
	TaskID        uint `json:"task_id"`
	ApplicantID   uint `json:"applicant_id"`
	TaskOwnerID   uint `json:"task_owner_id"`
}

// GetApplicationParticipants returns who may chat about an application, but
// only to one of them: anyone else gets ErrApplicationNotFound, so ids can't
// be probed.
func (s *TaskService) GetApplicationParticipants(ctx context.Context, applicationID, userID uint) (*ApplicationParticipants, error) {
	application, err := s.applicationRepo.GetByID(ctx, applicationID)
	if err != nil {
		return nil, ErrApplicationNotFound
	}
	if userID != application.ApplicantID && userID != application.Task.CreatedBy {
		return nil, ErrApplicationNotFound
	}
	return &ApplicationParticipants{
		ApplicationID: application.ID,
		TaskID:        application.TaskID,
		ApplicantID:   application.ApplicantID,
		TaskOwnerID:   application.Task.CreatedBy,
	}, nil
}

// ListUserApplications returns the applications userID has made.
func (s *TaskService) ListUserApplications(ctx context.Context, userID uint) ([]models.Application, error) {
	return s.applicationRepo.ListByUser(ctx, userID)
}

// GetTaskApplications returns all applications for a given task
func (s *TaskService) GetTaskApplications(ctx context.Context, taskID uint) ([]models.Application, error) {
	return s.applicationRepo.ListByTask(ctx, taskID)
}
