package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrAgentNotFound       = errors.New("agent not found")
	ErrAgentLimitExceeded  = errors.New("agent limit exceeded")
	ErrAgentRateLimited    = errors.New("agent has reached daily message limit")
)

// AgentService handles agent-related business logic
type AgentService interface {
	CreateAgent(ctx context.Context, userID uuid.UUID, name, description string, dailyLimit int) (*models.Agent, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentByAPIKey(ctx context.Context, apiKey string) (*models.Agent, error)
	GetAgentsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Agent, error)
	UpdateAgent(ctx context.Context, agent *models.Agent) error
	DeleteAgent(ctx context.Context, id uuid.UUID) error
	RegenerateAPIKey(ctx context.Context, id uuid.UUID) (string, error)
	ResetDailyUsage(ctx context.Context) error
	IncrementUsage(ctx context.Context, id uuid.UUID) error
	CheckRateLimit(ctx context.Context, id uuid.UUID) (bool, error)
}

type agentService struct {
	agentRepo repository.AgentRepository
	userRepo  repository.UserRepository
}

// NewAgentService creates a new AgentService
func NewAgentService(agentRepo repository.AgentRepository, userRepo repository.UserRepository) AgentService {
	return &agentService{
		agentRepo: agentRepo,
		userRepo:  userRepo,
	}
}

// generateAPIKey creates a new random API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CreateAgent creates a new agent
func (s *agentService) CreateAgent(ctx context.Context, userID uuid.UUID, name, description string, dailyLimit int) (*models.Agent, error) {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// Set default daily limit if not specified
	if dailyLimit <= 0 {
		dailyLimit = 50 // Default to 50 messages per day
	}

	// Create the agent
	now := time.Now()
	agent := &models.Agent{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        name,
		Description: description,
		APIKey:      apiKey,
		DailyLimit:  dailyLimit,
		UsedToday:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save the agent
	err = s.agentRepo.Create(ctx, agent)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

// GetAgentByID retrieves an agent by ID
func (s *agentService) GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

// GetAgentByAPIKey retrieves an agent by API key
func (s *agentService) GetAgentByAPIKey(ctx context.Context, apiKey string) (*models.Agent, error) {
	agent, err := s.agentRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}
	return agent, nil
}

// GetAgentsByUserID retrieves all agents for a user
func (s *agentService) GetAgentsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Agent, error) {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Get agents
	return s.agentRepo.GetByUserID(ctx, userID)
}

// UpdateAgent updates an existing agent
func (s *agentService) UpdateAgent(ctx context.Context, agent *models.Agent) error {
	// Check if agent exists
	existingAgent, err := s.agentRepo.GetByID(ctx, agent.ID)
	if err != nil {
		return err
	}
	if existingAgent == nil {
		return ErrAgentNotFound
	}

	// Preserve the API key (it should only be changed via RegenerateAPIKey)
	agent.APIKey = existingAgent.APIKey
	
	// Update the agent
	agent.UpdatedAt = time.Now()
	return s.agentRepo.Update(ctx, agent)
}

// DeleteAgent soft-deletes an agent
func (s *agentService) DeleteAgent(ctx context.Context, id uuid.UUID) error {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}

	// Delete the agent
	return s.agentRepo.Delete(ctx, id)
}

// RegenerateAPIKey generates a new API key for an agent
func (s *agentService) RegenerateAPIKey(ctx context.Context, id uuid.UUID) (string, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if agent == nil {
		return "", ErrAgentNotFound
	}

	// Generate new API key
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}

	// Update agent with new API key
	agent.APIKey = apiKey
	agent.UpdatedAt = time.Now()
	err = s.agentRepo.Update(ctx, agent)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

// ResetDailyUsage resets the used_today counter for all agents
func (s *agentService) ResetDailyUsage(ctx context.Context) error {
	return s.agentRepo.ResetDailyUsage(ctx)
}

// IncrementUsage increments the used_today counter for an agent
func (s *agentService) IncrementUsage(ctx context.Context, id uuid.UUID) error {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if agent == nil {
		return ErrAgentNotFound
	}

	// Increment usage
	return s.agentRepo.IncrementUsage(ctx, id)
}

// CheckRateLimit checks if an agent has reached its daily message limit
func (s *agentService) CheckRateLimit(ctx context.Context, id uuid.UUID) (bool, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, id)
	if err != nil {
		return false, err
	}
	if agent == nil {
		return false, ErrAgentNotFound
	}

	// Check if agent has reached daily limit
	return agent.UsedToday >= agent.DailyLimit, nil
}
