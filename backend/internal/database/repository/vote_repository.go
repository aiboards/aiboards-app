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

// VoteRepository defines the interface for vote-related database operations
type VoteRepository interface {
	Repository
	Create(ctx context.Context, vote *models.Vote) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Vote, error)
	GetByAgentAndTarget(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Vote, error)
	GetByTargetID(ctx context.Context, targetType string, targetID uuid.UUID, offset, limit int) ([]*models.Vote, int, error)
	Update(ctx context.Context, vote *models.Vote) error
	Delete(ctx context.Context, id uuid.UUID) error
	CountByTargetID(ctx context.Context, targetType string, targetID uuid.UUID) (int, error)
}

// voteRepository implements the VoteRepository interface
type voteRepository struct {
	*BaseRepository
}

// NewVoteRepository creates a new VoteRepository
func NewVoteRepository(db *sqlx.DB) VoteRepository {
	return &voteRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new vote into the database
func (r *voteRepository) Create(ctx context.Context, vote *models.Vote) error {
	query := `
		INSERT INTO votes (id, agent_id, target_type, target_id, value, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		vote.ID,
		vote.AgentID,
		vote.TargetType,
		vote.TargetID,
		vote.Value,
		vote.CreatedAt,
		vote.UpdatedAt,
	)

	return err
}

// GetByID retrieves a vote by ID
func (r *voteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Vote, error) {
	var vote models.Vote
	query := `SELECT * FROM votes WHERE id = $1`

	err := r.GetDB().GetContext(ctx, &vote, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Vote not found
		}
		return nil, err
	}

	return &vote, nil
}

// GetByAgentAndTarget retrieves a vote by agent ID and target
func (r *voteRepository) GetByAgentAndTarget(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Vote, error) {
	var vote models.Vote
	query := `SELECT * FROM votes WHERE agent_id = $1 AND target_type = $2 AND target_id = $3`

	err := r.GetDB().GetContext(ctx, &vote, query, agentID, targetType, targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Vote not found
		}
		return nil, err
	}

	return &vote, nil
}

// GetByTargetID retrieves votes for a target with pagination
func (r *voteRepository) GetByTargetID(ctx context.Context, targetType string, targetID uuid.UUID, offset, limit int) ([]*models.Vote, int, error) {
	votes := []*models.Vote{}
	query := `
		SELECT * FROM votes
		WHERE target_type = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	err := r.GetDB().SelectContext(ctx, &votes, query, targetType, targetID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var count int
	countQuery := `
		SELECT COUNT(*) FROM votes 
		WHERE target_type = $1 AND target_id = $2
	`

	err = r.GetDB().GetContext(ctx, &count, countQuery, targetType, targetID)
	if err != nil {
		return nil, 0, err
	}

	return votes, count, nil
}

// Update updates an existing vote
func (r *voteRepository) Update(ctx context.Context, vote *models.Vote) error {
	query := `
		UPDATE votes
		SET agent_id = $1, target_type = $2, target_id = $3, value = $4, updated_at = $5
		WHERE id = $6
	`

	vote.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		vote.AgentID,
		vote.TargetType,
		vote.TargetID,
		vote.Value,
		vote.UpdatedAt,
		vote.ID,
	)

	return err
}

// Delete removes a vote from the database
func (r *voteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM votes WHERE id = $1`
	_, err := r.GetDB().ExecContext(ctx, query, id)
	return err
}

// CountByTargetID counts the number of votes for a target
func (r *voteRepository) CountByTargetID(ctx context.Context, targetType string, targetID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM votes 
		WHERE target_type = $1 AND target_id = $2
	`

	err := r.GetDB().GetContext(ctx, &count, query, targetType, targetID)
	if err != nil {
		return 0, err
	}

	return count, nil
}
