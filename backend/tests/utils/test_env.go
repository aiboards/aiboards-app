package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/pkg/migration"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TestEnv provides a complete test environment
type TestEnv struct {
	T                  *testing.T
	Ctx                context.Context
	DB                 *sqlx.DB
	UserRepository     repository.UserRepository
	BetaCodeRepository repository.BetaCodeRepository
	AgentRepository    repository.AgentRepository
	AuthService        services.AuthService
	UserService        services.UserService
	AgentService       services.AgentService
	cleanupFuncs       []func()
}

// NewTestEnv creates a new test environment
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	// Create a context with timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	
	// Get database URL from environment or use default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aiboards_test:aiboards_test@localhost:5433/aiboards_test?sslmode=disable"
	}

	// Connect to the test database
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Get migration path from environment or use default
	migrationPath := os.Getenv("MIGRATION_PATH")
	if migrationPath == "" {
		migrationPath = "../../migrations"
	}

	// Run migrations
	log.Println("Running database migrations...")
	err = migration.RunMigrations(db, migrationPath)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	// Clear all tables before each test
	clearTables(t, db)

	// Create repositories
	userRepo := repository.NewUserRepository(db)
	betaCodeRepo := repository.NewBetaCodeRepository(db)
	agentRepo := repository.NewAgentRepository(db)

	// Create JWT secret for testing
	jwtSecret := "test-secret-key"
	
	// Default token expiration times
	accessExp := time.Hour
	refreshExp := time.Hour * 24

	// Create services
	authService := services.NewAuthService(
		userRepo,
		betaCodeRepo,
		jwtSecret,
		accessExp,
		refreshExp,
	)
	userService := services.NewUserService(userRepo)
	agentService := services.NewAgentService(agentRepo, userRepo)

	// Create cleanup functions
	cleanupFuncs := []func(){
		cancel, // Cancel the context
		func() { clearTables(t, db) }, // Clear tables
		func() { db.Close() }, // Close database connection
	}

	return &TestEnv{
		T:                  t,
		Ctx:                ctx,
		DB:                 db,
		UserRepository:     userRepo,
		BetaCodeRepository: betaCodeRepo,
		AgentRepository:    agentRepo,
		AuthService:        authService,
		UserService:        userService,
		AgentService:       agentService,
		cleanupFuncs:       cleanupFuncs,
	}
}

// Cleanup cleans up the test environment
func (e *TestEnv) Cleanup() {
	for _, cleanup := range e.cleanupFuncs {
		cleanup()
	}
}

// CreateTestUser creates a test user for testing and returns the user ID and password
func (e *TestEnv) CreateTestUser() (uuid.UUID, string) {
	return CreateTestUser(e.T, e.DB)
}

// CreateTestBetaCode creates a test beta code for testing
func (e *TestEnv) CreateTestBetaCode() string {
	return CreateTestBetaCode(e.T, e.DB)
}

// CreateTestAgent creates a test agent for testing
func (e *TestEnv) CreateTestAgent(userID uuid.UUID) *models.Agent {
	agent := &models.Agent{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        fmt.Sprintf("Test Agent %s", time.Now().Format("20060102150405")),
		Description: "Test agent description",
		APIKey:      fmt.Sprintf("test-api-key-%s", uuid.New().String()),
		DailyLimit:  100,
		UsedToday:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO agents (id, user_id, name, description, api_key, daily_limit, used_today, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := e.DB.Exec(
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
	)

	if err != nil {
		e.T.Fatalf("Failed to insert test agent: %v", err)
	}

	return agent
}

// WithTransaction runs a function within a transaction
func (e *TestEnv) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := e.DB.BeginTxx(e.Ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
