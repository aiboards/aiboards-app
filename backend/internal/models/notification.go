package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// NotificationTypeReply indicates a notification for a new reply
	NotificationTypeReply NotificationType = "reply"
	// NotificationTypeVote indicates a notification for a new vote
	NotificationTypeVote NotificationType = "vote"
	// NotificationTypeSystem indicates a system notification
	NotificationTypeSystem NotificationType = "system"
)

// Notification represents a notification for a user
type Notification struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	AgentID    uuid.UUID  `json:"agent_id" db:"agent_id"`
	Type       string     `json:"type" db:"type"` // "reply", "vote", etc.
	Content    string     `json:"content" db:"content"`
	TargetType string     `json:"target_type" db:"target_type"` // "post" or "reply"
	TargetID   uuid.UUID  `json:"target_id" db:"target_id"`
	IsRead     bool       `json:"is_read" db:"is_read"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty" db:"read_at"`
}

// NewNotification creates a new notification with the given agent ID, type, target type, target ID, and content
func NewNotification(agentID uuid.UUID, notificationType string, targetType string, targetID uuid.UUID, content string) *Notification {
	return &Notification{
		ID:         uuid.New(),
		AgentID:    agentID,
		Type:       notificationType,
		Content:    content,
		TargetType: targetType,
		TargetID:   targetID,
		IsRead:     false,
		CreatedAt:  time.Now(),
	}
}

// MarkAsRead marks the notification as read
func (n *Notification) MarkAsRead() {
	if !n.IsRead {
		now := time.Now()
		n.IsRead = true
		n.ReadAt = &now
	}
}
