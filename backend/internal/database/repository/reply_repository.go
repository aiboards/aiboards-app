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

// ReplyRepository defines the interface for reply-related database operations
type ReplyRepository interface {
	Repository
	Create(ctx context.Context, reply *models.Reply) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Reply, error)
	GetByParentID(ctx context.Context, parentType string, parentID uuid.UUID, offset, limit int) ([]*models.Reply, error)
	GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Reply, error)
	Update(ctx context.Context, reply *models.Reply) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateVoteCount(ctx context.Context, id uuid.UUID, value int) error
	UpdateReplyCount(ctx context.Context, id uuid.UUID, value int) error
	CountByParentID(ctx context.Context, parentType string, parentID uuid.UUID) (int, error)
	CountByAgentID(ctx context.Context, agentID uuid.UUID) (int, error)
	GetThreadedReplies(ctx context.Context, postID uuid.UUID) ([]*models.Reply, error)
}

// replyRepository implements the ReplyRepository interface
type replyRepository struct {
	*BaseRepository
}

// NewReplyRepository creates a new ReplyRepository
func NewReplyRepository(db *sqlx.DB) ReplyRepository {
	return &replyRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new reply into the database
func (r *replyRepository) Create(ctx context.Context, reply *models.Reply) error {
	query := `
		INSERT INTO replies (id, parent_type, parent_id, agent_id, content, media_url, vote_count, reply_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		reply.ID,
		reply.ParentType,
		reply.ParentID,
		reply.AgentID,
		reply.Content,
		reply.MediaURL,
		reply.VoteCount,
		reply.ReplyCount,
		reply.CreatedAt,
		reply.UpdatedAt,
	)

	return err
}

// GetByID retrieves a reply by ID
func (r *replyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Reply, error) {
	var reply models.Reply
	query := `SELECT * FROM replies WHERE id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &reply, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Reply not found
		}
		return nil, err
	}

	return &reply, nil
}

// GetByParentID retrieves replies for a parent (post or reply) with pagination
func (r *replyRepository) GetByParentID(ctx context.Context, parentType string, parentID uuid.UUID, offset, limit int) ([]*models.Reply, error) {
	replies := []*models.Reply{}
	query := `
		SELECT * FROM replies
		WHERE parent_type = $1 AND parent_id = $2 AND deleted_at IS NULL
		ORDER BY created_at ASC
		LIMIT $3 OFFSET $4
	`

	err := r.GetDB().SelectContext(ctx, &replies, query, parentType, parentID, limit, offset)
	if err != nil {
		return nil, err
	}

	return replies, nil
}

// GetByAgentID retrieves replies created by an agent with pagination
func (r *replyRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Reply, error) {
	replies := []*models.Reply{}
	query := `
		SELECT * FROM replies
		WHERE agent_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.GetDB().SelectContext(ctx, &replies, query, agentID, limit, offset)
	if err != nil {
		return nil, err
	}

	return replies, nil
}

// Update updates an existing reply
func (r *replyRepository) Update(ctx context.Context, reply *models.Reply) error {
	query := `
		UPDATE replies
		SET parent_type = $1, parent_id = $2, agent_id = $3, content = $4, 
		    media_url = $5, vote_count = $6, reply_count = $7, updated_at = $8
		WHERE id = $9 AND deleted_at IS NULL
	`

	reply.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		reply.ParentType,
		reply.ParentID,
		reply.AgentID,
		reply.Content,
		reply.MediaURL,
		reply.VoteCount,
		reply.ReplyCount,
		reply.UpdatedAt,
		reply.ID,
	)

	return err
}

// Delete soft-deletes a reply
func (r *replyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE replies
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// UpdateVoteCount updates the vote count for a reply
func (r *replyRepository) UpdateVoteCount(ctx context.Context, id uuid.UUID, value int) error {
	query := `
		UPDATE replies
		SET vote_count = vote_count + $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, value, now, id)
	return err
}

// UpdateReplyCount updates the reply count for a reply
func (r *replyRepository) UpdateReplyCount(ctx context.Context, id uuid.UUID, value int) error {
	query := `
		UPDATE replies
		SET reply_count = reply_count + $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, value, now, id)
	return err
}

// CountByParentID counts the number of replies for a parent
func (r *replyRepository) CountByParentID(ctx context.Context, parentType string, parentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM replies WHERE parent_type = $1 AND parent_id = $2 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query, parentType, parentID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountByAgentID counts the number of replies created by an agent
func (r *replyRepository) CountByAgentID(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM replies WHERE agent_id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query, agentID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetThreadedReplies retrieves all replies for a post in a threaded structure
func (r *replyRepository) GetThreadedReplies(ctx context.Context, postID uuid.UUID) ([]*models.Reply, error) {
	replies := []*models.Reply{}
	
	// This query uses a recursive CTE to get all replies in a thread
	query := `
		WITH RECURSIVE reply_tree AS (
			-- Base case: get all direct replies to the post
			SELECT r.*, 0 AS depth
			FROM replies r
			WHERE r.parent_type = 'post' AND r.parent_id = $1 AND r.deleted_at IS NULL
			
			UNION ALL
			
			-- Recursive case: get replies to replies
			SELECT r.*, rt.depth + 1
			FROM replies r
			JOIN reply_tree rt ON r.parent_type = 'reply' AND r.parent_id = rt.id
			WHERE r.deleted_at IS NULL
		)
		SELECT id, parent_type, parent_id, agent_id, content, media_url, 
		       vote_count, reply_count, created_at, updated_at, deleted_at
		FROM reply_tree
		ORDER BY depth ASC, created_at ASC
	`

	err := r.GetDB().SelectContext(ctx, &replies, query, postID)
	if err != nil {
		return nil, err
	}

	return replies, nil
}
