package integration

import (
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Test data
	email := "test@example.com"
	password := "securePassword123"
	name := "Test User"

	// Call the Register method with real repositories
	user, tokens, err := env.AuthService.Register(env.Ctx, email, password, name, betaCode)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, tokens)

	assert.Equal(t, email, user.Email)
	assert.Equal(t, name, user.Name)
	assert.NotEmpty(t, user.ID)

	// Verify tokens were generated
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)

	// Verify user can be retrieved from database
	retrievedUser, err := env.UserRepository.GetByEmail(env.Ctx, email)
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)
	assert.Equal(t, email, retrievedUser.Email)

	// Verify beta code was marked as used
	retrievedBetaCode, err := env.BetaCodeRepository.GetByCode(env.Ctx, betaCode)
	require.NoError(t, err)
	require.NotNil(t, retrievedBetaCode)
	assert.True(t, retrievedBetaCode.IsUsed)
	assert.Equal(t, user.ID, *retrievedBetaCode.UsedByID)
}

func TestLogin_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()

	// Get the user from the database to get their email
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Call the Login method with real repositories
	loggedInUser, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, loggedInUser)
	require.NotNil(t, tokens)

	assert.Equal(t, user.Email, loggedInUser.Email)
	assert.Equal(t, user.Name, loggedInUser.Name)
	assert.Equal(t, user.ID, loggedInUser.ID)

	// Verify tokens were generated
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestRefreshTokens_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()

	// Get the user from the database to get their email
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Login to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// Store the original tokens
	originalAccessToken := tokens.AccessToken
	originalRefreshToken := tokens.RefreshToken

	// Wait a short time to ensure tokens will be different
	time.Sleep(1 * time.Second)

	// Call the RefreshTokens method with real repositories
	newTokens, err := env.AuthService.RefreshTokens(env.Ctx, tokens.RefreshToken)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, newTokens)

	// Verify new tokens were generated and are different from the old ones
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, originalAccessToken, newTokens.AccessToken, "Access token should be different after refresh")
	assert.NotEqual(t, originalRefreshToken, newTokens.RefreshToken, "Refresh token should be different after refresh")
}
