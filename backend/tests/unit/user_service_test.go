package unit

import (
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetUserByID(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	testUser, err := models.NewUser("test@example.com", "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Test getting user by ID
	user, err := env.UserService.GetUserByID(env.Ctx, testUser.ID)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, testUser.Email, user.Email)
	assert.Equal(t, testUser.Name, user.Name)
}

func TestGetUserByEmail(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	email := "email-test@example.com"
	testUser, err := models.NewUser(email, "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Test getting user by email
	user, err := env.UserService.GetUserByEmail(env.Ctx, email)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, testUser.ID, user.ID)
	assert.Equal(t, email, user.Email)
}

func TestUpdateUser(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	testUser, err := models.NewUser("update-test@example.com", "password123", "Original Name")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Update user
	testUser.Name = "Updated Name"
	err = env.UserService.UpdateUser(env.Ctx, testUser)
	assert.NoError(t, err)

	// Verify update
	updatedUser, err := env.UserService.GetUserByID(env.Ctx, testUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Name", updatedUser.Name)
}

func TestChangePassword(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	email := "password-test@example.com"
	oldPassword := "oldpassword123"
	newPassword := "newpassword456"

	testUser, err := models.NewUser(email, oldPassword, "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Change password
	err = env.UserService.ChangePassword(env.Ctx, testUser.ID, oldPassword, newPassword)
	assert.NoError(t, err)

	// Verify password change by trying to login
	user, tokens, err := env.AuthService.Login(env.Ctx, email, newPassword)
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, tokens)
}

func TestDeleteUser(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	testUser, err := models.NewUser("delete-test@example.com", "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Delete user
	err = env.UserService.DeleteUser(env.Ctx, testUser.ID)
	assert.NoError(t, err)

	// Verify deletion
	user, err := env.UserService.GetUserByID(env.Ctx, testUser.ID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrUserNotFound, err)
	assert.Nil(t, user, "User should be nil after deletion")
}

func TestGetUserByID_NotFound(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Test with random UUID
	randomID := uuid.New()
	user, err := env.UserService.GetUserByID(env.Ctx, randomID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrUserNotFound, err)
	assert.Nil(t, user, "User should be nil for non-existent ID")
}
