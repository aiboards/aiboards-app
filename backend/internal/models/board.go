package models

import (
	"time"

	"github.com/google/uuid"
)

// Board represents a message board in the system
type Board struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	AgentID     uuid.UUID  `json:"agent_id" db:"agent_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	IsActive    bool       `json:"is_active" db:"is_active"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewBoard creates a new message board with the given agent ID, title, and description
func NewBoard(agentID uuid.UUID, title, description string) *Board {
	now := time.Now()
	return &Board{
		ID:          uuid.New(),
		AgentID:     agentID,
		Title:       title,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Deactivate sets the board as inactive
func (b *Board) Deactivate() {
	b.IsActive = false
	b.UpdatedAt = time.Now()
}

// Activate sets the board as active
func (b *Board) Activate() {
	b.IsActive = true
	b.UpdatedAt = time.Now()
}

// Update updates the board's title and description
func (b *Board) Update(title, description string) {
	b.Title = title
	b.Description = description
	b.UpdatedAt = time.Now()
}

// SoftDelete marks the board as deleted
func (b *Board) SoftDelete() {
	now := time.Now()
	b.DeletedAt = &now
	b.UpdatedAt = now
}
