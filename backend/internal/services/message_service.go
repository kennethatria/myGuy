package services

import (
	"context"
	"errors"
	"myguy/internal/models"
	"myguy/internal/repositories"
	"regexp"
	"strings"
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

	// Filter content
	filteredContent, hasRemovedContent := s.filterContent(input.Content)

	message := &models.Message{
		TaskID:          input.TaskID,
		SenderID:        input.SenderID,
		RecipientID:     input.RecipientID,
		Content:         filteredContent,
		OriginalContent: input.Content,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, err
	}

	message.HasRemovedContent = hasRemovedContent
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

// EditMessage allows users to edit their own messages
func (s *MessageService) EditMessage(ctx context.Context, messageID uint, userID uint, newContent string) (*models.Message, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if message.SenderID != userID {
		return nil, errors.New("unauthorized: can only edit your own messages")
	}

	if message.IsDeleted {
		return nil, errors.New("cannot edit deleted message")
	}

	// Filter content
	filteredContent, hasRemovedContent := s.filterContent(newContent)

	message.Content = filteredContent
	message.OriginalContent = newContent
	message.IsEdited = true

	if err := s.messageRepo.Update(ctx, message); err != nil {
		return nil, err
	}

	message.HasRemovedContent = hasRemovedContent
	return message, nil
}

// DeleteMessage soft deletes a message
func (s *MessageService) DeleteMessage(ctx context.Context, messageID uint, userID uint) error {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return err
	}

	if message.SenderID != userID {
		return errors.New("unauthorized: can only delete your own messages")
	}

	message.IsDeleted = true
	message.Content = "[Message deleted]"

	return s.messageRepo.Update(ctx, message)
}

// MarkAsRead marks a message as read
func (s *MessageService) MarkAsRead(ctx context.Context, messageID uint, userID uint) (*models.Message, error) {
	message, err := s.messageRepo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}

	if message.RecipientID != userID {
		return nil, errors.New("unauthorized: can only mark messages sent to you as read")
	}

	if !message.IsRead {
		message.IsRead = true
		if err := s.messageRepo.Update(ctx, message); err != nil {
			return nil, err
		}
	}

	return message, nil
}

// GetUserConversations gets all conversations for a user
func (s *MessageService) GetUserConversations(ctx context.Context, userID uint) ([]models.ConversationSummary, error) {
	return s.messageRepo.GetUserConversations(ctx, userID)
}

// filterContent removes URLs, emails, and phone numbers from content
func (s *MessageService) filterContent(content string) (string, bool) {
	if content == "" {
		return content, false
	}

	originalContent := content
	
	// Regex patterns
	urlPattern := regexp.MustCompile(`(?i)(?:https?|ftp|ftps):\/\/[^\s]+|www\.[^\s]+\.[^\s]+|[^\s]+\.[a-z]{2,}\/[^\s]*`)
	emailPattern := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	phonePattern := regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}|[0-9]{10,15}`)

	// Replace patterns
	content = urlPattern.ReplaceAllString(content, "[link removed]")
	content = emailPattern.ReplaceAllString(content, "[email removed]")
	
	// For phone numbers, check if they're likely actual phone numbers
	phoneMatches := phonePattern.FindAllString(content, -1)
	for _, match := range phoneMatches {
		digits := regexp.MustCompile(`\D`).ReplaceAllString(match, "")
		if len(digits) >= 10 && len(digits) <= 15 {
			content = strings.Replace(content, match, "[phone removed]", 1)
		}
	}

	hasRemovedContent := content != originalContent
	return strings.TrimSpace(content), hasRemovedContent
}
