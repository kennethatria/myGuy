package repositories

import (
	"context"
	"myguy/internal/models"
	"time"
	"gorm.io/gorm"
)

type GormMessageRepository struct {
	db *gorm.DB
}

func NewGormMessageRepository(db *gorm.DB) *GormMessageRepository {
	return &GormMessageRepository{db: db}
}

func (r *GormMessageRepository) Create(ctx context.Context, message *models.Message) error {
	err := r.db.WithContext(ctx).Create(message).Error
	if err != nil {
		return err
	}
	
	// Reload message with related data
	return r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		First(message, message.ID).Error
}

func (r *GormMessageRepository) ListByTask(ctx context.Context, taskID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) GetByID(ctx context.Context, messageID uint) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		First(&message, messageID).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *GormMessageRepository) Update(ctx context.Context, message *models.Message) error {
	// Set timestamps
	now := time.Now()
	if message.IsEdited && message.EditedAt == nil {
		message.EditedAt = &now
	}
	if message.IsDeleted && message.DeletedAt == nil {
		message.DeletedAt = &now
	}
	if message.IsRead && message.ReadAt == nil {
		message.ReadAt = &now
	}
	
	return r.db.WithContext(ctx).Save(message).Error
}

func (r *GormMessageRepository) ListByUser(ctx context.Context, userID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("sender_id = ? OR recipient_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) ListByApplication(ctx context.Context, applicationID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).
		Preload("Sender").
		Preload("Recipient").
		Where("application_id = ?", applicationID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *GormMessageRepository) GetApplication(ctx context.Context, applicationID uint) (*models.Application, error) {
	var application models.Application
	err := r.db.WithContext(ctx).First(&application, applicationID).Error
	if err != nil {
		return nil, err
	}
	return &application, nil
}

func (r *GormMessageRepository) GetTask(ctx context.Context, taskID uint) (*models.Task, error) {
	var task models.Task
	err := r.db.WithContext(ctx).First(&task, taskID).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *GormMessageRepository) GetUserConversations(ctx context.Context, userID uint) ([]models.ConversationSummary, error) {
	var conversations []models.ConversationSummary
	
	query := `
		WITH LastMessages AS (
			SELECT DISTINCT ON (task_id)
				m.task_id,
				m.content as last_message,
				m.created_at as last_message_time,
				CASE 
					WHEN m.sender_id = ? THEN m.recipient_id
					ELSE m.sender_id
				END as other_user_id
			FROM messages m
			WHERE m.sender_id = ? OR m.recipient_id = ?
			ORDER BY m.task_id, m.created_at DESC
		),
		UnreadCounts AS (
			SELECT 
				task_id,
				COUNT(*) as unread_count
			FROM messages
			WHERE recipient_id = ? AND is_read = false
			GROUP BY task_id
		)
		SELECT 
			lm.task_id,
			t.title as task_title,
			t.description as task_description,
			t.status as task_status,
			lm.last_message,
			lm.last_message_time,
			lm.other_user_id,
			u.name as other_user_name,
			COALESCE(uc.unread_count, 0) as unread_count
		FROM LastMessages lm
		INNER JOIN tasks t ON lm.task_id = t.id
		INNER JOIN users u ON lm.other_user_id = u.id
		LEFT JOIN UnreadCounts uc ON lm.task_id = uc.task_id
		ORDER BY lm.last_message_time DESC
	`
	
	err := r.db.WithContext(ctx).Raw(query, userID, userID, userID, userID).Scan(&conversations).Error
	if err != nil {
		return nil, err
	}
	
	return conversations, nil
}
