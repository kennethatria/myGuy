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
