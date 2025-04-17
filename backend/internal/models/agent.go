package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// Agent represents an AI agent in the system
type Agent struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	UserID      uuid.UUID  `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	APIKey      string     `json:"-" db:"api_key"` // Never sent to client
	DailyLimit  int        `json:"daily_limit" db:"daily_limit"`
	UsedToday   int        `json:"used_today" db:"used_today"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewAgent creates a new agent with the given user ID, name, and description
func NewAgent(userID uuid.UUID, name, description string) (*Agent, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Agent{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Description: description,
		APIKey:      apiKey,
		DailyLimit:  500, // Default daily limit of 500 requests
		UsedToday:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// RefreshAPIKey generates a new API key for the agent
func (a *Agent) RefreshAPIKey() error {
	apiKey, err := generateAPIKey()
	if err != nil {
		return err
	}

	a.APIKey = apiKey
	a.UpdatedAt = time.Now()
	return nil
}

// IncrementUsage increments the agent's usage count for the day
// Returns true if the agent has exceeded its daily limit
func (a *Agent) IncrementUsage() bool {
	a.UsedToday++
	a.UpdatedAt = time.Now()
	return a.UsedToday > a.DailyLimit
}

// ResetDailyUsage resets the agent's daily usage count
func (a *Agent) ResetDailyUsage() {
	a.UsedToday = 0
	a.UpdatedAt = time.Now()
}

// generateAPIKey creates a new random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
