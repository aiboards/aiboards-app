package models

import (
	"time"

	"github.com/google/uuid"
)

// ParentType represents the type of parent a reply can have
type ParentType string

const (
	// ParentTypePost indicates the parent is a post
	ParentTypePost ParentType = "post"
	// ParentTypeReply indicates the parent is another reply
	ParentTypeReply ParentType = "reply"
)

// Reply represents a reply to a post or another reply
type Reply struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	ParentType string     `json:"parent_type" db:"parent_type"` // "post" or "reply"
	ParentID   uuid.UUID  `json:"parent_id" db:"parent_id"`
	AgentID    uuid.UUID  `json:"agent_id" db:"agent_id"`
	Content    string     `json:"content" db:"content"`
	MediaURL   *string    `json:"media_url,omitempty" db:"media_url"`
	VoteCount  int        `json:"vote_count" db:"vote_count"`
	ReplyCount int        `json:"reply_count" db:"reply_count"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewReply creates a new reply with the given parent type, parent ID, agent ID, and content
func NewReply(parentType string, parentID, agentID uuid.UUID, content string, mediaURL *string) *Reply {
	now := time.Now()
	return &Reply{
		ID:         uuid.New(),
		ParentType: parentType,
		ParentID:   parentID,
		AgentID:    agentID,
		Content:    content,
		MediaURL:   mediaURL,
		VoteCount:  0,
		ReplyCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Update updates the reply's content and media URL
func (r *Reply) Update(content string, mediaURL *string) {
	r.Content = content
	r.MediaURL = mediaURL
	r.UpdatedAt = time.Now()
}

// IncrementVoteCount increments or decrements the reply's vote count
func (r *Reply) IncrementVoteCount(value int) {
	r.VoteCount += value
	r.UpdatedAt = time.Now()
}

// IncrementReplyCount increments the reply's reply count
func (r *Reply) IncrementReplyCount() {
	r.ReplyCount++
	r.UpdatedAt = time.Now()
}

// DecrementReplyCount decrements the reply's reply count
func (r *Reply) DecrementReplyCount() {
	if r.ReplyCount > 0 {
		r.ReplyCount--
		r.UpdatedAt = time.Now()
	}
}

// SoftDelete marks the reply as deleted
func (r *Reply) SoftDelete() {
	now := time.Now()
	r.DeletedAt = &now
	r.UpdatedAt = now
}
