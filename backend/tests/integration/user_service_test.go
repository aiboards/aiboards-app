package integration

import (
	"testing"

	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Test data
	email := "create-test@example.com"
	password := "securePassword123"
	name := "Create Test User"

	// Create a new user
	user, err := env.UserService.CreateUser(env.Ctx, email, password, name)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, name, user.Name)
	assert.False(t, user.IsAdmin)
	assert.NotEmpty(t, user.ID)

	// Verify user can be retrieved from database
	retrievedUser, err := env.UserRepository.GetByEmail(env.Ctx, email)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, email, retrievedUser.Email)
}

func TestGetUserByID_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()

	// Get the user by ID
	user, err := env.UserService.GetUserByID(env.Ctx, userID)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
}

func TestGetUserByEmail_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()

	// Get the original user to get the email
	originalUser, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, originalUser)

	// Get the user by email
	user, err := env.UserService.GetUserByEmail(env.Ctx, originalUser.Email)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, originalUser.Email, user.Email)
}

func TestUpdateUser_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()

	// Get the original user
	originalUser, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, originalUser)

	// Update user data
	updatedName := "Updated Name"
	originalUser.Name = updatedName

	// Update the user
	err = env.UserService.UpdateUser(env.Ctx, originalUser)
	require.NoError(t, err)

	// Get the updated user
	updatedUser, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)

	// Assert results
	assert.Equal(t, updatedName, updatedUser.Name)
	assert.Equal(t, originalUser.Email, updatedUser.Email)
}

func TestDeleteUser_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()

	// Delete the user
	err := env.UserService.DeleteUser(env.Ctx, userID)
	require.NoError(t, err)

	// Try to get the deleted user
	deletedUser, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	assert.Nil(t, deletedUser, "User should be soft-deleted")
}
