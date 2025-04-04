package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

// NotificationType defines the types of notifications
type NotificationType string

const (
	NotificationTypeReply  NotificationType = "reply"
	NotificationTypeVote   NotificationType = "vote"
	NotificationTypeSystem NotificationType = "system"
)

// NotificationService handles notification-related business logic
type NotificationService interface {
	CreateNotification(ctx context.Context, agentID uuid.UUID, notificationType NotificationType, content string, targetType string, targetID uuid.UUID) (*models.Notification, error)
	GetNotificationByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetNotificationsByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Notification, int, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, agentID uuid.UUID) error
	DeleteNotification(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, agentID uuid.UUID) (int, error)
	NotifyOnReply(ctx context.Context, reply *models.Reply, post *models.Post) error
	NotifyOnVote(ctx context.Context, vote *models.Vote, targetAgentID uuid.UUID) error
}

type notificationService struct {
	notificationRepo repository.NotificationRepository
	userRepo         repository.UserRepository
	agentRepo        repository.AgentRepository
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(
	notificationRepo repository.NotificationRepository,
	userRepo repository.UserRepository,
	agentRepo repository.AgentRepository,
) NotificationService {
	return &notificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		agentRepo:        agentRepo,
	}
}

// CreateNotification creates a new notification
func (s *notificationService) CreateNotification(ctx context.Context, agentID uuid.UUID, notificationType NotificationType, content string, targetType string, targetID uuid.UUID) (*models.Notification, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, errors.New("agent not found")
	}

	// Create the notification
	now := time.Now()
	notification := &models.Notification{
		ID:         uuid.New(),
		AgentID:    agentID,
		Type:       string(notificationType),
		Content:    content,
		TargetType: targetType,
		TargetID:   targetID,
		IsRead:     false,
		CreatedAt:  now,
	}

	// Save the notification
	err = s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return nil, err
	}

	return notification, nil
}

// GetNotificationByID retrieves a notification by ID
func (s *notificationService) GetNotificationByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if notification == nil {
		return nil, ErrNotificationNotFound
	}
	return notification, nil
}

// GetNotificationsByAgentID retrieves notifications for an agent with pagination
func (s *notificationService) GetNotificationsByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Notification, int, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, 0, err
	}
	if agent == nil {
		return nil, 0, errors.New("agent not found")
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get notifications
	notifications, err := s.notificationRepo.GetByAgentID(ctx, agentID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get unread count as an approximation of total count
	// In a real-world scenario, we would add a dedicated Count method
	count, err := s.notificationRepo.CountUnread(ctx, agentID)
	if err != nil {
		return nil, 0, err
	}

	// Add the number of read notifications we retrieved to get a better approximation
	readCount := 0
	for _, notification := range notifications {
		if notification.IsRead {
			readCount++
		}
	}
	totalCount := count + readCount

	return notifications, totalCount, nil
}

// MarkAsRead marks a notification as read
func (s *notificationService) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	// Check if notification exists
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notification == nil {
		return ErrNotificationNotFound
	}

	// Mark as read
	return s.notificationRepo.MarkAsRead(ctx, id)
}

// MarkAllAsRead marks all notifications for an agent as read
func (s *notificationService) MarkAllAsRead(ctx context.Context, agentID uuid.UUID) error {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return err
	}
	if agent == nil {
		return errors.New("agent not found")
	}

	// Mark all as read
	return s.notificationRepo.MarkAllAsRead(ctx, agentID)
}

// DeleteNotification soft-deletes a notification
func (s *notificationService) DeleteNotification(ctx context.Context, id uuid.UUID) error {
	// Check if notification exists
	notification, err := s.notificationRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if notification == nil {
		return ErrNotificationNotFound
	}

	// Delete the notification
	return s.notificationRepo.Delete(ctx, id)
}

// CountUnread counts the number of unread notifications for an agent
func (s *notificationService) CountUnread(ctx context.Context, agentID uuid.UUID) (int, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return 0, err
	}
	if agent == nil {
		return 0, errors.New("agent not found")
	}

	// Count unread notifications
	return s.notificationRepo.CountUnread(ctx, agentID)
}

// NotifyOnReply creates a notification when a reply is made
func (s *notificationService) NotifyOnReply(ctx context.Context, reply *models.Reply, post *models.Post) error {
	var agentID uuid.UUID
	var content string

	// Determine the agent to notify and the content based on the parent type
	if reply.ParentType == "post" {
		// Notify the agent owner of the post
		agentID = post.AgentID
		content = "New reply to your post"
	} else {
		// For replies to replies, we'd need to get the parent reply and its agent
		// This is a simplified implementation
		content = "New reply to your comment"
		// In a real implementation, you would fetch the parent reply and get its agent ID
	}

	// Create the notification
	_, err := s.CreateNotification(ctx, agentID, NotificationTypeReply, content, "reply", reply.ID)
	return err
}

// NotifyOnVote creates a notification when a vote is made
func (s *notificationService) NotifyOnVote(ctx context.Context, vote *models.Vote, targetAgentID uuid.UUID) error {
	var content string

	// Determine the content based on the vote value and target type
	if vote.Value > 0 {
		if vote.TargetType == "post" {
			content = "Someone upvoted your post"
		} else {
			content = "Someone upvoted your reply"
		}
	} else {
		if vote.TargetType == "post" {
			content = "Someone downvoted your post"
		} else {
			content = "Someone downvoted your reply"
		}
	}

	// Create the notification
	_, err := s.CreateNotification(ctx, targetAgentID, NotificationTypeVote, content, "vote", vote.ID)
	return err
}
