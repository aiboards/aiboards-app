package utils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/config"
	"github.com/garrettallen/aiboards/backend/internal/database"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TestDB provides a clean database for testing
func TestDB(t *testing.T) *sqlx.DB {
	// Connect to test database
	cfg := &config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/aiboards_test?sslmode=disable",
	}

	db, err := database.NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Clear all tables before each test
	clearTables(t, db)

	return db
}

// clearTables truncates all tables in the test database
func clearTables(t *testing.T, db *sqlx.DB) {
	tables := []string{
		"users",
		"agents",
		"beta_codes",
		"replies",
		"posts",
		"boards",
		"notifications",
		"votes",
		// Add other tables as they are created
	}

	for _, table := range tables {
		_, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}

// WithTestDB provides a database connection for the duration of the test
func WithTestDB(t *testing.T, fn func(*sqlx.DB)) {
	db := TestDB(t)
	defer db.Close()
	fn(db)
}

// WithTx provides a transaction for the duration of the test
func WithTx(t *testing.T, db *sqlx.DB, fn func(*sqlx.Tx)) {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	fn(tx)

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Failed to commit transaction: %v", err)
	}
}

// CreateTestUser creates a test user for testing
func CreateTestUser(t *testing.T, db *sqlx.DB) (uuid.UUID, string) {
	// Generate a unique email with both timestamp and UUID to ensure uniqueness
	email := fmt.Sprintf("test-%s-%s@example.com", time.Now().Format("20060102150405"), uuid.New().String())
	password := "password123"
	name := "Test User"

	// Create user directly in the database
	user, err := models.NewUser(email, password, name)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	query := `
		INSERT INTO users (id, email, password_hash, name, is_admin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = db.Exec(
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.IsAdmin,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		t.Fatalf("Failed to insert test user: %v", err)
	}

	return user.ID, password
}

// CreateTestBetaCode creates a test beta code for testing
func CreateTestBetaCode(t *testing.T, db *sqlx.DB) string {
	// Generate a shorter code that fits within the 16 character limit
	code := "T" + time.Now().Format("0102150405")
	now := time.Now()

	id := uuid.New()

	query := `
		INSERT INTO beta_codes (id, code, is_used, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := db.Exec(
		query,
		id,
		code,
		false,
		now,
	)

	if err != nil {
		t.Fatalf("Failed to insert test beta code: %v", err)
	}

	return code
}
