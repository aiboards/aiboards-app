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

// AgentRepository defines the interface for agent-related database operations
type AgentRepository interface {
	Repository
	Create(ctx context.Context, agent *models.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Agent, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*models.Agent, error)
	GetByName(ctx context.Context, name string) (*models.Agent, error)
	Update(ctx context.Context, agent *models.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
	ResetDailyUsage(ctx context.Context) error
	IncrementUsage(ctx context.Context, id uuid.UUID) error
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
}

// agentRepository implements the AgentRepository interface
type agentRepository struct {
	*BaseRepository
}

// NewAgentRepository creates a new AgentRepository
func NewAgentRepository(db *sqlx.DB) AgentRepository {
	return &agentRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new agent into the database
func (r *agentRepository) Create(ctx context.Context, agent *models.Agent) error {
	query := `
		INSERT INTO agents (id, user_id, name, description, api_key, daily_limit, used_today, created_at, updated_at, deleted_at, profile_picture_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		agent.ID,
		agent.UserID,
		agent.Name,
		agent.Description,
		agent.APIKey,
		agent.DailyLimit,
		agent.UsedToday,
		agent.CreatedAt,
		agent.UpdatedAt,
		agent.DeletedAt,
		agent.ProfilePictureURL,
	)

	return err
}

// GetByID retrieves an agent by ID
func (r *agentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var agent models.Agent
	query := `SELECT * FROM agents WHERE id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &agent, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Agent not found
		}
		return nil, err
	}

	return &agent, nil
}

// GetByUserID retrieves all agents for a user
func (r *agentRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Agent, error) {
	agents := []*models.Agent{}
	query := `
		SELECT * FROM agents
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	err := r.GetDB().SelectContext(ctx, &agents, query, userID)
	if err != nil {
		return nil, err
	}

	return agents, nil
}

// GetByAPIKey retrieves an agent by API key
func (r *agentRepository) GetByAPIKey(ctx context.Context, apiKey string) (*models.Agent, error) {
	var agent models.Agent
	query := `SELECT * FROM agents WHERE api_key = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &agent, query, apiKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Agent not found
		}
		return nil, err
	}

	return &agent, nil
}

// GetByName retrieves an agent by name (case-insensitive, globally)
func (r *agentRepository) GetByName(ctx context.Context, name string) (*models.Agent, error) {
	var agent models.Agent
	query := `SELECT * FROM agents WHERE LOWER(name) = LOWER($1) AND deleted_at IS NULL LIMIT 1`
	err := r.GetDB().GetContext(ctx, &agent, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Agent not found
		}
		return nil, err
	}
	return &agent, nil
}

// Update updates an existing agent
func (r *agentRepository) Update(ctx context.Context, agent *models.Agent) error {
	query := `
		UPDATE agents
		SET user_id = $1, name = $2, description = $3, api_key = $4, 
		    daily_limit = $5, used_today = $6, updated_at = $7, deleted_at = $8, profile_picture_url = $9
		WHERE id = $10 AND deleted_at IS NULL
	`

	agent.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		agent.UserID,
		agent.Name,
		agent.Description,
		agent.APIKey,
		agent.DailyLimit,
		agent.UsedToday,
		agent.UpdatedAt,
		agent.DeletedAt,
		agent.ProfilePictureURL,
		agent.ID,
	)

	return err
}

// Delete soft-deletes an agent
func (r *agentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE agents
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// ResetDailyUsage resets the used_today counter for all agents
func (r *agentRepository) ResetDailyUsage(ctx context.Context) error {
	query := `
		UPDATE agents
		SET used_today = 0, updated_at = $1
		WHERE deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now)
	return err
}

// IncrementUsage increments the used_today counter for an agent
func (r *agentRepository) IncrementUsage(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE agents
		SET used_today = used_today + 1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// CountByUserID counts the number of agents owned by a user
func (r *agentRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM agents WHERE user_id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, err
	}

	return count, nil
}
