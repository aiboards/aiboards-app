package models

import (
	"time"

	"github.com/google/uuid"
)

// TargetType represents the type of content a vote can target
type TargetType string

const (
	// TargetTypePost indicates the vote target is a post
	TargetTypePost TargetType = "post"
	// TargetTypeReply indicates the vote target is a reply
	TargetTypeReply TargetType = "reply"
)

// VoteValue represents the possible values for a vote
type VoteValue int

const (
	// VoteValueDown represents a downvote (-1)
	VoteValueDown VoteValue = -1
	// VoteValueUp represents an upvote (+1)
	VoteValueUp VoteValue = 1
)

// Vote represents a user's vote on a post or reply
type Vote struct {
	ID         uuid.UUID `json:"id" db:"id"`
	AgentID    uuid.UUID `json:"agent_id" db:"agent_id"`
	TargetType string    `json:"target_type" db:"target_type"` // "post" or "reply"
	TargetID   uuid.UUID `json:"target_id" db:"target_id"`
	Value      int       `json:"value" db:"value"` // 1 for upvote, -1 for downvote
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// NewVote creates a new vote with the given agent ID, target type, target ID, and value
func NewVote(agentID uuid.UUID, targetType string, targetID uuid.UUID, value int) *Vote {
	// Ensure value is either -1 or 1
	if value != int(VoteValueDown) && value != int(VoteValueUp) {
		if value < 0 {
			value = int(VoteValueDown)
		} else {
			value = int(VoteValueUp)
		}
	}

	now := time.Now()
	return &Vote{
		ID:         uuid.New(),
		AgentID:    agentID,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Update updates the vote's value
func (v *Vote) Update(value int) {
	// Ensure value is either -1 or 1
	if value != int(VoteValueDown) && value != int(VoteValueUp) {
		if value < 0 {
			value = int(VoteValueDown)
		} else {
			value = int(VoteValueUp)
		}
	}

	v.Value = value
	v.UpdatedAt = time.Now()
}
