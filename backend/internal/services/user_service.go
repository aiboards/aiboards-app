package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
)

// UserService handles user-related business logic
type UserService interface {
	CreateUser(ctx context.Context, email, password, name string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ListUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error)
	Authenticate(ctx context.Context, email, password string) (*models.User, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error
	GetUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error)
	EnsureAdminUser(ctx context.Context) error
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService creates a new UserService
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

// CreateUser creates a new user
func (s *userService) CreateUser(ctx context.Context, email, password, name string) (*models.User, error) {
	// Check if email already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create the user
	now := time.Now()
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		IsAdmin:      false, // Default to non-admin
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Save the user
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateUser updates an existing user
func (s *userService) UpdateUser(ctx context.Context, user *models.User) error {
	// Check if user exists
	existingUser, err := s.userRepo.GetByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return ErrUserNotFound
	}

	// Check if email is being changed and if it's already in use
	if existingUser.Email != user.Email {
		userWithEmail, err := s.userRepo.GetByEmail(ctx, user.Email)
		if err != nil {
			return err
		}
		if userWithEmail != nil && userWithEmail.ID != user.ID {
			return ErrEmailAlreadyExists
		}
	}

	// Update the user
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(ctx, user)
}

// DeleteUser soft-deletes a user
func (s *userService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Delete the user
	return s.userRepo.Delete(ctx, id)
}

// ListUsers retrieves a paginated list of users
func (s *userService) ListUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get users
	users, err := s.userRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// Authenticate verifies user credentials and returns the user if valid
func (s *userService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// ChangePassword changes a user's password
func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify current password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword))
	if err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Update password
	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()
	return s.userRepo.Update(ctx, user)
}

// GetUsers retrieves a paginated list of users
func (s *userService) GetUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get users
	users, err := s.userRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// EnsureAdminUser checks if an admin user exists and creates one if not
func (s *userService) EnsureAdminUser(ctx context.Context) error {
	// Check if admin email and password are set in environment variables
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminEmail == "" || adminPassword == "" {
		log.Printf("Admin credentials not found in environment variables. Skipping admin creation.")
		return nil // Skip admin creation if credentials not provided
	}

	log.Printf("Attempting to ensure admin user exists with email: %s", adminEmail)

	// Check if any admin user already exists
	users, _, err := s.GetUsers(ctx, 1, 100)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin users: %w", err)
	}

	for _, user := range users {
		if user.IsAdmin {
			return nil // Admin already exists, no need to create one
		}
	}

	// Check if the admin email already exists as a non-admin user
	existingUser, err := s.GetUserByEmail(ctx, adminEmail)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return fmt.Errorf("failed to check for existing user: %w", err)
	}

	if existingUser != nil {
		// User exists but is not an admin, promote to admin
		existingUser.IsAdmin = true
		if err := s.UpdateUser(ctx, existingUser); err != nil {
			return fmt.Errorf("failed to promote user to admin: %w", err)
		}
		log.Printf("Existing user %s promoted to admin", adminEmail)
		return nil
	}

	// Create a new admin user
	user, err := s.CreateUser(ctx, adminEmail, adminPassword, "Administrator")
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Update user to be admin
	user.IsAdmin = true
	if err := s.UpdateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to set admin privileges: %w", err)
	}

	log.Printf("Admin user initialized with email: %s", adminEmail)
	return nil
}
