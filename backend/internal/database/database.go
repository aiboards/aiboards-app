package database

import (
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/garrettallen/aiboards/backend/config"
)

// NewDB creates a new database connection
func NewDB(config *config.Config) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
