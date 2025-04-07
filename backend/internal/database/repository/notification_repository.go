package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/garrettallen/aiboards/backend/internal/models"
)

// NotificationRepository defines the interface for notification-related database operations
type NotificationRepository interface {
	Repository
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error)
	GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Notification, error)
	MarkAsRead(ctx context.Context, id uuid.UUID) error
	MarkAllAsRead(ctx context.Context, agentID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountUnread(ctx context.Context, agentID uuid.UUID) (int, error)
}

// notificationRepository implements the NotificationRepository interface
type notificationRepository struct {
	*BaseRepository
}

// NewNotificationRepository creates a new NotificationRepository
func NewNotificationRepository(db *sqlx.DB) NotificationRepository {
	return &notificationRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new notification into the database
func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	query := `
		INSERT INTO notifications (id, agent_id, type, content, target_type, target_id, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		notification.ID,
		notification.AgentID,
		notification.Type,
		notification.Content,
		notification.TargetType,
		notification.TargetID,
		notification.IsRead,
		notification.CreatedAt,
	)

	return err
}

// GetByID retrieves a notification by ID
func (r *notificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Notification, error) {
	var notification models.Notification

	query := `
		SELECT id, agent_id, type, content, target_type, target_id, is_read, created_at, read_at
		FROM notifications
		WHERE id = $1
	`

	err := r.GetDB().GetContext(ctx, &notification, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("notification not found")
		}
		return nil, err
	}

	return &notification, nil
}

// GetByAgentID retrieves notifications for an agent with pagination
func (r *notificationRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Notification, error) {
	var notifications []*models.Notification

	query := `
		SELECT id, agent_id, type, content, target_type, target_id, is_read, created_at, read_at
		FROM notifications
		WHERE agent_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.GetDB().SelectContext(ctx, &notifications, query, agentID, limit, offset)
	if err != nil {
		return nil, err
	}

	return notifications, nil
}

// MarkAsRead marks a notification as read
func (r *notificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	// Check if notification exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()

	query := `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE id = $2
	`

	_, err = r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// MarkAllAsRead marks all notifications for an agent as read
func (r *notificationRepository) MarkAllAsRead(ctx context.Context, agentID uuid.UUID) error {
	now := time.Now()

	query := `
		UPDATE notifications
		SET is_read = true, read_at = $1
		WHERE agent_id = $2 AND is_read = false
	`

	_, err := r.GetDB().ExecContext(ctx, query, now, agentID)
	return err
}

// Delete deletes a notification
func (r *notificationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if notification exists
	_, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	query := `
		DELETE FROM notifications
		WHERE id = $1
	`

	_, err = r.GetDB().ExecContext(ctx, query, id)
	return err
}

// CountUnread counts the number of unread notifications for an agent
func (r *notificationRepository) CountUnread(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int

	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE agent_id = $1 AND is_read = false
	`

	err := r.GetDB().GetContext(ctx, &count, query, agentID)
	if err != nil {
		return 0, err
	}

	return count, nil
}
