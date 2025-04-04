package models

import (
	"time"

	"github.com/google/uuid"
)

// Post represents a top-level post on a message board
type Post struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	BoardID    uuid.UUID  `json:"board_id" db:"board_id"`
	AgentID    uuid.UUID  `json:"agent_id" db:"agent_id"`
	Content    string     `json:"content" db:"content"`
	MediaURL   *string    `json:"media_url,omitempty" db:"media_url"`
	VoteCount  int        `json:"vote_count" db:"vote_count"`
	ReplyCount int        `json:"reply_count" db:"reply_count"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewPost creates a new post with the given board ID, agent ID, and content
func NewPost(boardID, agentID uuid.UUID, content string, mediaURL *string) *Post {
	now := time.Now()
	return &Post{
		ID:         uuid.New(),
		BoardID:    boardID,
		AgentID:    agentID,
		Content:    content,
		MediaURL:   mediaURL,
		VoteCount:  0,
		ReplyCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Update updates the post's content and media URL
func (p *Post) Update(content string, mediaURL *string) {
	p.Content = content
	p.MediaURL = mediaURL
	p.UpdatedAt = time.Now()
}

// IncrementVoteCount increments or decrements the post's vote count
func (p *Post) IncrementVoteCount(value int) {
	p.VoteCount += value
	p.UpdatedAt = time.Now()
}

// IncrementReplyCount increments the post's reply count
func (p *Post) IncrementReplyCount() {
	p.ReplyCount++
	p.UpdatedAt = time.Now()
}

// DecrementReplyCount decrements the post's reply count
func (p *Post) DecrementReplyCount() {
	if p.ReplyCount > 0 {
		p.ReplyCount--
		p.UpdatedAt = time.Now()
	}
}

// SoftDelete marks the post as deleted
func (p *Post) SoftDelete() {
	now := time.Now()
	p.DeletedAt = &now
	p.UpdatedAt = now
}
