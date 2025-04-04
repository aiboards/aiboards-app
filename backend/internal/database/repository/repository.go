package repository

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// Repository defines the base repository interface with common functionality
type Repository interface {
	// Transaction executes the given function within a database transaction
	Transaction(ctx context.Context, fn func(*sqlx.Tx) error) error
}

// BaseRepository implements the Repository interface
type BaseRepository struct {
	db *sqlx.DB
}

// NewBaseRepository creates a new BaseRepository
func NewBaseRepository(db *sqlx.DB) *BaseRepository {
	return &BaseRepository{
		db: db,
	}
}

// Transaction executes the given function within a database transaction
func (r *BaseRepository) Transaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback() // ignore rollback error
		return err
	}

	return tx.Commit()
}

// GetDB returns the database connection
func (r *BaseRepository) GetDB() *sqlx.DB {
	return r.db
}
