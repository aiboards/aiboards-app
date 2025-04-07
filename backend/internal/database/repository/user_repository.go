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

// UserRepository defines the interface for user-related database operations
type UserRepository interface {
	Repository
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*models.User, error)
	Count(ctx context.Context) (int, error)
}

// userRepository implements the UserRepository interface
type userRepository struct {
	*BaseRepository
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *sqlx.DB) UserRepository {
	return &userRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, name, is_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.IsAdmin,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err
	}

	return &user, nil
}

// Update updates an existing user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET email = $1, password_hash = $2, name = $3, is_admin = $4, updated_at = $5, deleted_at = $6
		WHERE id = $7
	`

	user.UpdatedAt = time.Now()

	_, err := r.GetDB().ExecContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.IsAdmin,
		user.UpdatedAt,
		user.DeletedAt,
		user.ID,
	)

	return err
}

// Delete soft-deletes a user
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE users
		SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND deleted_at IS NULL
	`

	now := time.Now()

	_, err := r.GetDB().ExecContext(ctx, query, now, id)
	return err
}

// List retrieves a paginated list of users
func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	users := []*models.User{}
	query := `
		SELECT * FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	err := r.GetDB().SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// Count returns the total number of users
func (r *userRepository) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`

	err := r.GetDB().GetContext(ctx, &count, query)
	if err != nil {
		return 0, err
	}

	return count, nil
}
