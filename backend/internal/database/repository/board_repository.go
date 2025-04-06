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

// BoardRepository defines the interface for board-related database operations
type BoardRepository interface {
	Repository
	Create(ctx context.Context, board *models.Board) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Board, error)
	GetByAgentID(ctx context.Context, agentID uuid.UUID) (*models.Board, error)
	Update(ctx context.Context, board *models.Board) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*models.Board, error)
	SetActive(ctx context.Context, id uuid.UUID, isActive bool) error
	Count(ctx context.Context) (int, error)
}

// boardRepository implements the BoardRepository interface
type boardRepository struct {
	*BaseRepository
}

// NewBoardRepository creates a new BoardRepository
func NewBoardRepository(db *sqlx.DB) BoardRepository {
	return &boardRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new board into the database
func (r *boardRepository) Create(ctx context.Context, board *models.Board) error {
	query := `
		INSERT INTO boards (id, agent_id, title, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		board.ID,
		board.AgentID,
		board.Title,
		board.Description,
		board.IsActive,
		board.CreatedAt,
		board.UpdatedAt,
	)

	return err
}

// GetByID retrieves a board by ID
func (r *boardRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Board, error) {
	var board models.Board
	query := `SELECT * FROM boards WHERE id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &board, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Board not found
		}
		return nil, err
	}

	return &board, nil
}

// GetByAgentID retrieves a board by agent ID
func (r *boardRepository) GetByAgentID(ctx context.Context, agentID uuid.UUID) (*models.Board, error) {
	var board models.Board
	query := `SELECT * FROM boards WHERE agent_id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &board, query, agentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Board not found
		}
		return nil, err
	}

	return &board, nil
}

// Update updates an existing board
func (r *boardRepository) Update(ctx context.Context, board *models.Board) error {
	query := `
		UPDATE boards
		SET agent_id = $1, title = $2, description = $3, is_active = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
	`

	board.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		board.AgentID,
		board.Title,
		board.Description,
		board.IsActive,
		board.UpdatedAt,
		board.ID,
	)

	return err
}

// Delete soft-deletes a board
func (r *boardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE boards
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// List retrieves a paginated list of boards
func (r *boardRepository) List(ctx context.Context, offset, limit int) ([]*models.Board, error) {
	boards := []*models.Board{}
	query := `
		SELECT * FROM boards
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.GetDB().SelectContext(ctx, &boards, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return boards, nil
}

// Count returns the total number of non-deleted boards
func (r *boardRepository) Count(ctx context.Context) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM boards
		WHERE deleted_at IS NULL
	`

	err := r.GetDB().GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// SetActive sets the is_active status of a board
func (r *boardRepository) SetActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	query := `
		UPDATE boards
		SET is_active = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, isActive, now, id)
	return err
}
