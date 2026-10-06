package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"myguy/internal/contacts"
	"myguy/internal/models"
	"myguy/internal/proximity"
	"myguy/internal/repositories"
	"sort"
	"strings"
	"time"
)

// A gig is a short sticky note, live for gigLifetime unless someone applies.
const (
	gigLifetime       = 24 * time.Hour
	headlineMaxWords  = 5
	bodyMaxWords      = 20
	headlineMaxChars  = 60
	bodyMaxChars      = 200
	applicationMaxLen = 500
)

var (
	ErrTaskNotFound        = errors.New("task not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrTaskNotOpen         = errors.New("task is not open for applications")
	ErrInvalidStatus       = errors.New("invalid status transition")
	ErrApplicationNotFound = errors.New("application not found")
	ErrApplicationNotPending = errors.New("application is no longer pending")
	ErrOwnTask             = errors.New("you cannot apply to your own task")
	ErrAlreadyApplied      = errors.New("you have already applied to this task")
	ErrTaskWasAssigned     = errors.New("a task that was assigned can't be deleted; cancel it instead")

	// Gig text errors: the request is fine, the words need changing.
	ErrHeadlineRequired = errors.New("headline is required")
	ErrBodyRequired     = errors.New("note is required")
	ErrHeadlineTooLong  = fmt.Errorf("headline can be at most %d words", headlineMaxWords)
	ErrBodyTooLong      = fmt.Errorf("note can be at most %d words", bodyMaxWords)
	ErrMessageTooLong   = fmt.Errorf("message can be at most %d characters", applicationMaxLen)
	ErrContactDetails   = errors.New("remove phone numbers, emails, links and handles; you can share them in chat once you've agreed on the gig")
)

// IsGigTextError reports whether err is about the gig or application text.
func IsGigTextError(err error) bool {
	for _, e := range []error{ErrHeadlineRequired, ErrBodyRequired, ErrHeadlineTooLong, ErrBodyTooLong, ErrMessageTooLong, ErrContactDetails} {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// validateGigText enforces the sticky-note limits and keeps contact details
// off public gigs. It returns the trimmed headline and body.
func validateGigText(title, description string) (string, string, error) {
	title, description = strings.TrimSpace(title), strings.TrimSpace(description)
	switch {
	case title == "":
		return "", "", ErrHeadlineRequired
	case description == "":
		return "", "", ErrBodyRequired
	case len(strings.Fields(title)) > headlineMaxWords || len(title) > headlineMaxChars:
		return "", "", ErrHeadlineTooLong
	case len(strings.Fields(description)) > bodyMaxWords || len(description) > bodyMaxChars:
		return "", "", ErrBodyTooLong
	case contacts.Contains(title) || contacts.Contains(description):
		return "", "", ErrContactDetails
	}
	return title, description, nil
}

// TaskNotifier tells the people involved about task events by posting a
// message into their conversation about the task. Implementations must not
// block or fail the caller (chat is best effort).
type TaskNotifier interface {
	TaskMessage(taskID, senderID, recipientID uint, content string)
	// TaskMatch posts the acceptance message and lets the two people share
	// contact details in that conversation from now on.
	TaskMatch(taskID, senderID, recipientID uint, content string)
}

type noopNotifier struct{}

func (noopNotifier) TaskMessage(uint, uint, uint, string) {}
func (noopNotifier) TaskMatch(uint, uint, uint, string)   {}

// Locator keeps the rough location of posts in the proximity service.
// Implementations must not block or fail the caller (it is best effort).
type Locator interface {
	Save(kind string, id uint, at proximity.Location)
	Delete(kind string, id uint)
}

type noopLocator struct{}

func (noopLocator) Save(string, uint, proximity.Location) {}
func (noopLocator) Delete(string, uint)                   {}

// Distancer measures rough distances to posts, as buckets (an index into
// proximity.Buckets). Posts without a stored location are left out.
type Distancer interface {
	Distances(kind string, at proximity.Location, ids []uint) (map[uint]int, error)
}

type TaskService struct {
	taskRepo        repositories.TaskRepository
	applicationRepo repositories.ApplicationRepository
	notifier        TaskNotifier
	locator         Locator
	distancer       Distancer // nil: no distance sorting or tags
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
		locator:         noopLocator{},
	}
}

// WithLocator saves gigs' rough locations through locator (nil = don't).
func (s *TaskService) WithLocator(locator Locator) *TaskService {
	if locator != nil {
		s.locator = locator
	}
	return s
}

// WithDistancer turns on distance sorting and tags (nil leaves them off).
func (s *TaskService) WithDistancer(distancer Distancer) *TaskService {
	s.distancer = distancer
	return s
}

// SaveLocation stores a gig's rough location; at is optional (nil = keep
// whatever was saved before).
func (s *TaskService) SaveLocation(taskID uint, at *proximity.Location) {
	if at != nil {
		s.locator.Save("task", taskID, *at)
	}
}

type CreateTaskInput struct {
	Title       string
	Description string
	CreatedBy   uint
	// Location is the poster's rough location, if they shared it
	Location *proximity.Location
}

type UpdateTaskInput struct {
	ID          uint
	Title       string
	Description string
	UpdatedBy   uint
}

func (s *TaskService) CreateTask(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
	title, description, err := validateGigText(input.Title, input.Description)
	if err != nil {
		return nil, err
	}

	task := &models.Task{
		Title:       title,
		Description: description,
		Deadline:    time.Now().UTC().Add(gigLifetime),
		CreatedBy:   input.CreatedBy,
		Status:      "open",
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	s.SaveLocation(task.ID, input.Location)

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

	title, description, err := validateGigText(input.Title, input.Description)
	if err != nil {
		return nil, err
	}

	task.Title = title
	task.Description = description

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

	if err := s.taskRepo.Delete(ctx, taskID); err != nil {
		return err
	}
	s.locator.Delete("task", taskID)
	return nil
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

// ListTasksNear is ListTasksWithPagination ordered by distance from at:
// nearest bucket first, newest first within a bucket, gigs without a location
// last. Every match is ranked before the page is cut, so paging and totals
// work as usual. If distances can't be had, it falls back to the usual order.
func (s *TaskService) ListTasksNear(ctx context.Context, filters map[string]interface{}, at proximity.Location) (*PaginatedTasksResult, error) {
	if s.distancer == nil {
		return s.ListTasksWithPagination(ctx, filters)
	}
	ids, err := s.taskRepo.ListIDs(ctx, filters)
	if err != nil {
		return nil, err
	}
	buckets, err := s.distancer.Distances("task", at, ids)
	if err != nil {
		log.Printf("WARNING: distance sort unavailable, showing newest first: %v", err)
		return s.ListTasksWithPagination(ctx, filters)
	}

	unknown := len(proximity.Buckets)
	rank := func(id uint) int {
		if b, ok := buckets[id]; ok {
			return b
		}
		return unknown
	}
	// ids arrive newest first; a stable sort keeps that order within a bucket
	sort.SliceStable(ids, func(a, b int) bool { return rank(ids[a]) < rank(ids[b]) })

	page, _ := filters["page"].(int)
	perPage, _ := filters["per_page"].(int)
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	start := (page - 1) * perPage
	if start > len(ids) {
		start = len(ids)
	}
	end := start + perPage
	if end > len(ids) {
		end = len(ids)
	}
	pageIDs := ids[start:end]

	loaded, err := s.taskRepo.ListByIDs(ctx, pageIDs)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]models.Task, len(loaded))
	for _, task := range loaded {
		byID[task.ID] = task
	}
	tasks := make([]models.Task, 0, len(pageIDs))
	for _, id := range pageIDs {
		if task, ok := byID[id]; ok {
			if b, has := buckets[id]; has {
				task.Distance = proximity.BucketLabel(b)
			}
			tasks = append(tasks, task)
		}
	}

	total := int64(len(ids))
	return &PaginatedTasksResult{
		Tasks:      tasks,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: int((total + int64(perPage) - 1) / int64(perPage)),
	}, nil
}

// TagDistances adds a rough distance tag from at to each task that has a
// location, keeping their order. Best effort: without distances, no tags.
func (s *TaskService) TagDistances(tasks []models.Task, at proximity.Location) {
	if s.distancer == nil || len(tasks) == 0 {
		return
	}
	ids := make([]uint, len(tasks))
	for i, task := range tasks {
		ids[i] = task.ID
	}
	buckets, err := s.distancer.Distances("task", at, ids)
	if err != nil {
		log.Printf("WARNING: distance tags unavailable: %v", err)
		return
	}
	for i := range tasks {
		if b, ok := buckets[tasks[i].ID]; ok {
			tasks[i].Distance = proximity.BucketLabel(b)
		}
	}
}

func (s *TaskService) ListUserTasks(ctx context.Context, userID uint, role string) ([]models.Task, error) {
	return s.taskRepo.ListByUser(ctx, userID, role)
}

// ApplyForTask records interest in an open gig. Price and details are agreed
// in chat, so an application is just a short message.
func (s *TaskService) ApplyForTask(ctx context.Context, taskID, applicantID uint, message string) error {
	message = strings.TrimSpace(message)
	if len(message) > applicationMaxLen {
		return ErrMessageTooLong
	}
	// Applicant and poster aren't matched yet, so no contact details.
	if contacts.Contains(message) {
		return ErrContactDetails
	}

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
	// Past its 24 hours with nobody interested: expired, even if the
	// expiry job hasn't marked it yet.
	if len(existing) == 0 && time.Now().After(task.Deadline) {
		return ErrTaskNotOpen
	}

	application := &models.Application{
		TaskID:      taskID,
		ApplicantID: applicantID,
		Message:     message,
		Status:      "pending",
	}

	if err := s.applicationRepo.Create(ctx, application); err != nil {
		return err
	}

	content := fmt.Sprintf("📩 New application for \"%s\".", task.Title)
	if message != "" {
		content += "\n\n" + message
	}
	s.notifier.TaskMessage(taskID, applicantID, task.CreatedBy, content)
	return nil
}

// AssignTask accepts an application: the applicant gets the task, every
// other pending application is declined, and the pair may now share contacts.
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
	assigned, err := s.taskRepo.AssignIfOpen(ctx, taskID, application.ApplicantID)
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

	s.notifier.TaskMatch(taskID, task.CreatedBy, application.ApplicantID,
		fmt.Sprintf("✅ Your application for \"%s\" was accepted. You can now share phone numbers here to arrange the details.", task.Title))
	for _, applicantID := range others {
		s.notifier.TaskMessage(taskID, task.CreatedBy, applicantID,
			fmt.Sprintf("Your application for \"%s\" wasn't selected this time.", task.Title))
	}

	task.Status = "in_progress"
	task.AssignedTo = &application.ApplicantID
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
	case "expired":
		// From expired: repost (open again for a fresh 24 hours) or cancel
		validTransition = status == "open" || status == "cancelled"
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

	// Moving back to open starts a fresh listing: no assignment, a new
	// 24 hours on the board
	if status == "open" {
		task.AssignedTo = nil
		task.CompletedAt = nil
		task.Deadline = time.Now().UTC().Add(gigLifetime)
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

// ExpireStaleTasks marks open gigs that reached their deadline without any
// application as expired, returning how many it changed.
func (s *TaskService) ExpireStaleTasks(ctx context.Context) (int64, error) {
	return s.taskRepo.ExpireUnanswered(ctx, time.Now().UTC())
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
