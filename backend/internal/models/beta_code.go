package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// BetaCode represents a beta invitation code
type BetaCode struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	Code      string     `json:"code" db:"code"`
	IsUsed    bool       `json:"is_used" db:"is_used"`
	UsedByID  *uuid.UUID `json:"used_by_id,omitempty" db:"used_by_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
}

// NewBetaCode creates a new beta code
func NewBetaCode() (*BetaCode, error) {
	code, err := generateBetaCode()
	if err != nil {
		return nil, err
	}

	return &BetaCode{
		ID:        uuid.New(),
		Code:      code,
		IsUsed:    false,
		CreatedAt: time.Now(),
	}, nil
}

// MarkAsUsed marks the beta code as used by the specified user
func (b *BetaCode) MarkAsUsed(userID uuid.UUID) {
	now := time.Now()
	b.IsUsed = true
	b.UsedByID = &userID
	b.UsedAt = &now
}

// generateBetaCode creates a new random beta code
func generateBetaCode() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
