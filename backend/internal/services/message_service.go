package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
)

type MessageService struct {
	messageRepo repositories.MessageRepository
}

func NewMessageService(messageRepo repositories.MessageRepository) *MessageService {
	return &MessageService{
		messageRepo: messageRepo,
	}
}

type CreateMessageInput struct {
	TaskID      uint
	SenderID    uint
	RecipientID uint
	Content     string
}

func (s *MessageService) CreateMessage(ctx context.Context, input CreateMessageInput) (*models.Message, error) {
	if input.Content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	message := &models.Message{
		TaskID:      input.TaskID,
		SenderID:    input.SenderID,
		RecipientID: input.RecipientID,
		Content:     input.Content,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) GetTaskMessages(ctx context.Context, taskID uint) ([]models.Message, error) {
	return s.messageRepo.ListByTask(ctx, taskID)
}

func (s *MessageService) GetUserMessages(ctx context.Context, userID uint) ([]models.Message, error) {
	return s.messageRepo.ListByUser(ctx, userID)
}

func (s *MessageService) CreateApplicationMessage(ctx context.Context, applicationID uint, senderID uint, content string) (*models.Message, error) {
	if content == "" {
		return nil, errors.New("message content cannot be empty")
	}

	// Get application to validate sender and determine recipient
	application, err := s.messageRepo.GetApplication(ctx, applicationID)
	if err != nil {
		return nil, errors.New("application not found")
	}

	// Get task to get task creator
	task, err := s.messageRepo.GetTask(ctx, application.TaskID)
	if err != nil {
		return nil, errors.New("task not found")
	}

	// Determine recipient based on sender
	var recipientID uint
	if senderID == application.ApplicantID {
		// Applicant is sending to task creator
		recipientID = task.CreatedBy
	} else if senderID == task.CreatedBy {
		// Task creator is sending to applicant
		recipientID = application.ApplicantID
	} else {
		return nil, errors.New("sender must be either applicant or task creator")
	}

	message := &models.Message{
		TaskID:        application.TaskID,
		ApplicationID: &applicationID,
		SenderID:      senderID,
		RecipientID:   recipientID,
		Content:       content,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	return message, nil
}

func (s *MessageService) GetApplicationMessages(ctx context.Context, applicationID uint, userID uint) ([]models.Message, error) {
	// Verify user has access to these messages
	application, err := s.messageRepo.GetApplication(ctx, applicationID)
	if err != nil {
		return nil, errors.New("application not found")
	}

	task, err := s.messageRepo.GetTask(ctx, application.TaskID)
	if err != nil {
		return nil, errors.New("task not found")
	}

	// Check if user is either applicant or task creator
	if userID != application.ApplicantID && userID != task.CreatedBy {
		return nil, errors.New("unauthorized to view these messages")
	}

	return s.messageRepo.ListByApplication(ctx, applicationID)
}
