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

// BetaCodeRepository defines the interface for beta code-related database operations
type BetaCodeRepository interface {
	Repository
	Create(ctx context.Context, betaCode *models.BetaCode) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.BetaCode, error)
	GetByCode(ctx context.Context, code string) (*models.BetaCode, error)
	Update(ctx context.Context, betaCode *models.BetaCode) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*models.BetaCode, error)
	MarkAsUsed(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	CountActive(ctx context.Context) (int, error)
}

// betaCodeRepository implements the BetaCodeRepository interface
type betaCodeRepository struct {
	*BaseRepository
}

// NewBetaCodeRepository creates a new BetaCodeRepository
func NewBetaCodeRepository(db *sqlx.DB) BetaCodeRepository {
	return &betaCodeRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new beta code into the database
func (r *betaCodeRepository) Create(ctx context.Context, betaCode *models.BetaCode) error {
	query := `
		INSERT INTO beta_codes (id, code, is_used, used_by_id, used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		betaCode.ID,
		betaCode.Code,
		betaCode.IsUsed,
		betaCode.UsedByID,
		betaCode.UsedAt,
		betaCode.CreatedAt,
	)

	return err
}

// GetByID retrieves a beta code by ID
func (r *betaCodeRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BetaCode, error) {
	var betaCode models.BetaCode
	query := `SELECT * FROM beta_codes WHERE id = $1`

	err := r.GetDB().GetContext(ctx, &betaCode, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Beta code not found
		}
		return nil, err
	}

	return &betaCode, nil
}

// GetByCode retrieves a beta code by code string
func (r *betaCodeRepository) GetByCode(ctx context.Context, code string) (*models.BetaCode, error) {
	var betaCode models.BetaCode
	query := `SELECT * FROM beta_codes WHERE code = $1`

	err := r.GetDB().GetContext(ctx, &betaCode, query, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Beta code not found
		}
		return nil, err
	}

	return &betaCode, nil
}

// Update updates an existing beta code
func (r *betaCodeRepository) Update(ctx context.Context, betaCode *models.BetaCode) error {
	query := `
		UPDATE beta_codes
		SET code = $1, is_used = $2, used_by_id = $3, used_at = $4
		WHERE id = $5
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		betaCode.Code,
		betaCode.IsUsed,
		betaCode.UsedByID,
		betaCode.UsedAt,
		betaCode.ID,
	)

	return err
}

// Delete deletes a beta code
func (r *betaCodeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		DELETE FROM beta_codes
		WHERE id = $1
	`

	_, err := r.GetDB().ExecContext(ctx, query, id)
	return err
}

// List retrieves a paginated list of beta codes
func (r *betaCodeRepository) List(ctx context.Context, offset, limit int) ([]*models.BetaCode, error) {
	var betaCodes []*models.BetaCode
	query := `
		SELECT * FROM beta_codes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.GetDB().SelectContext(ctx, &betaCodes, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return betaCodes, nil
}

// MarkAsUsed marks a beta code as used by a user
func (r *betaCodeRepository) MarkAsUsed(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE beta_codes
		SET is_used = true, used_by_id = $1, used_at = $2
		WHERE id = $3 AND is_used = false
	`

	now := time.Now()
	result, err := r.GetDB().ExecContext(ctx, query, userID, now, id)
	if err != nil {
		return err
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	} else if rowsAffected == 0 {
		return errors.New("beta code not found or already used")
	}

	return nil
}

// CountActive counts the number of unused beta codes
func (r *betaCodeRepository) CountActive(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM beta_codes WHERE is_used = false`

	err := r.GetDB().GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}
