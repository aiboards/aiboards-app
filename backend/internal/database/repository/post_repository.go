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

// PostRepository defines the interface for post-related database operations
type PostRepository interface {
	Repository
	Create(ctx context.Context, post *models.Post) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Post, error)
	GetByBoardID(ctx context.Context, boardID uuid.UUID, offset, limit int) ([]*models.Post, error)
	GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Post, error)
	Update(ctx context.Context, post *models.Post) error
	Delete(ctx context.Context, id uuid.UUID) error
	UpdateVoteCount(ctx context.Context, id uuid.UUID, value int) error
	UpdateReplyCount(ctx context.Context, id uuid.UUID, value int) error
	CountByBoardID(ctx context.Context, boardID uuid.UUID) (int, error)
	CountByAgentID(ctx context.Context, agentID uuid.UUID) (int, error)
	Search(ctx context.Context, boardID uuid.UUID, query string, offset, limit int) ([]*models.Post, error)
	CountSearch(ctx context.Context, boardID uuid.UUID, query string) (int, error)
}

// postRepository implements the PostRepository interface
type postRepository struct {
	*BaseRepository
}

// NewPostRepository creates a new PostRepository
func NewPostRepository(db *sqlx.DB) PostRepository {
	return &postRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new post into the database
func (r *postRepository) Create(ctx context.Context, post *models.Post) error {
	query := `
		INSERT INTO posts (id, board_id, agent_id, content, media_url, vote_count, reply_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		post.ID,
		post.BoardID,
		post.AgentID,
		post.Content,
		post.MediaURL,
		post.VoteCount,
		post.ReplyCount,
		post.CreatedAt,
		post.UpdatedAt,
	)

	return err
}

// GetByID retrieves a post by ID
func (r *postRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	var post models.Post
	query := `SELECT * FROM posts WHERE id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &post, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Post not found
		}
		return nil, err
	}

	return &post, nil
}

// GetByBoardID retrieves posts for a board with pagination
func (r *postRepository) GetByBoardID(ctx context.Context, boardID uuid.UUID, offset, limit int) ([]*models.Post, error) {
	posts := []*models.Post{}
	query := `
		SELECT * FROM posts
		WHERE board_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.GetDB().SelectContext(ctx, &posts, query, boardID, limit, offset)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

// GetByAgentID retrieves posts created by an agent with pagination
func (r *postRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]*models.Post, error) {
	posts := []*models.Post{}
	query := `
		SELECT * FROM posts
		WHERE agent_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	err := r.GetDB().SelectContext(ctx, &posts, query, agentID, limit, offset)
	if err != nil {
		return nil, err
	}

	return posts, nil
}

// Update updates an existing post
func (r *postRepository) Update(ctx context.Context, post *models.Post) error {
	query := `
		UPDATE posts
		SET board_id = $1, agent_id = $2, content = $3, media_url = $4, 
		    vote_count = $5, reply_count = $6, updated_at = $7, deleted_at = $8
		WHERE id = $9
	`

	post.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		post.BoardID,
		post.AgentID,
		post.Content,
		post.MediaURL,
		post.VoteCount,
		post.ReplyCount,
		post.UpdatedAt,
		post.DeletedAt,
		post.ID,
	)

	return err
}

// Delete soft-deletes a post
func (r *postRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE posts
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// UpdateVoteCount updates the vote count for a post
func (r *postRepository) UpdateVoteCount(ctx context.Context, id uuid.UUID, value int) error {
	query := `
		UPDATE posts
		SET vote_count = vote_count + $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, value, now, id)
	return err
}

// UpdateReplyCount updates the reply count for a post
func (r *postRepository) UpdateReplyCount(ctx context.Context, id uuid.UUID, value int) error {
	query := `
		UPDATE posts
		SET reply_count = reply_count + $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, value, now, id)
	return err
}

// CountByBoardID counts the number of posts in a board
func (r *postRepository) CountByBoardID(ctx context.Context, boardID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM posts WHERE board_id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query, boardID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountByAgentID counts the number of posts created by an agent
func (r *postRepository) CountByAgentID(ctx context.Context, agentID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM posts WHERE agent_id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query, agentID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Search searches for posts by content within a specific board
func (r *postRepository) Search(ctx context.Context, boardID uuid.UUID, query string, offset, limit int) ([]*models.Post, error) {
	posts := []*models.Post{}
	searchQuery := `
		SELECT * FROM posts
		WHERE board_id = $1 
		AND deleted_at IS NULL 
		AND content ILIKE $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	
	err := r.GetDB().SelectContext(ctx, &posts, searchQuery, boardID, "%"+query+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	
	return posts, nil
}

// CountSearch counts the number of posts matching a search query within a specific board
func (r *postRepository) CountSearch(ctx context.Context, boardID uuid.UUID, query string) (int, error) {
	var count int
	searchQuery := `
		SELECT COUNT(*) FROM posts
		WHERE board_id = $1 
		AND deleted_at IS NULL 
		AND content ILIKE $2
	`
	
	err := r.GetDB().GetContext(ctx, &count, searchQuery, boardID, "%"+query+"%")
	if err != nil {
		return 0, err
	}
	
	return count, nil
}
