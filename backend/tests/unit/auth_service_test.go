package unit

import (
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Test data
	email := "register-test@example.com"
	password := "securePassword123"
	name := "Register Test User"

	// Call the Register method
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

	// Verify beta code was marked as used
	retrievedBetaCode, err := env.BetaCodeRepository.GetByCode(env.Ctx, betaCode)
	require.NoError(t, err)
	require.NotNil(t, retrievedBetaCode)
	assert.True(t, retrievedBetaCode.IsUsed)
	assert.Equal(t, user.ID, *retrievedBetaCode.UsedByID)
}

func TestRegister_InvalidEmail(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Test with invalid email
	_, _, err := env.AuthService.Register(env.Ctx, "invalid-email", "password123", "Test User", betaCode)

	// Assert error
	assert.Error(t, err)
	// The exact error might vary based on implementation
}

func TestRegister_WeakPassword(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Test with weak password
	_, _, err := env.AuthService.Register(env.Ctx, "test@example.com", "weak", "Test User", betaCode)

	// Assert error
	assert.Error(t, err)
	// The exact error might vary based on implementation
}

func TestRegister_InvalidBetaCode(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Test with invalid beta code
	_, _, err := env.AuthService.Register(env.Ctx, "test@example.com", "password123", "Test User", "invalid-code")

	// Assert error
	assert.Error(t, err)
	assert.Equal(t, services.ErrInvalidBetaCode, err)
}

func TestRegister_UsedBetaCode(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Use the beta code once
	_, _, err := env.AuthService.Register(env.Ctx, "first@example.com", "password123", "First User", betaCode)
	require.NoError(t, err)

	// Try to use the same beta code again
	_, _, err = env.AuthService.Register(env.Ctx, "second@example.com", "password123", "Second User", betaCode)

	// Assert error
	assert.Error(t, err)
	assert.Equal(t, services.ErrInvalidBetaCode, err)
}

func TestRegister_EmailAlreadyExists(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Create a test beta code
	betaCode := env.CreateTestBetaCode()

	// Try to register with the same email
	_, _, err = env.AuthService.Register(env.Ctx, user.Email, "password123", "Test User", betaCode)

	// Assert error
	assert.Error(t, err)
	assert.Equal(t, services.ErrUserAlreadyExists, err)
}

func TestLogin_Success(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Call the Login method
	loggedInUser, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, loggedInUser)
	require.NotNil(t, tokens)

	assert.Equal(t, user.ID, loggedInUser.ID)
	assert.Equal(t, user.Email, loggedInUser.Email)
	assert.Equal(t, user.Name, loggedInUser.Name)

	// Verify tokens were generated
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
}

func TestLogin_UserNotFound(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Call the Login method with non-existent email
	_, _, err := env.AuthService.Login(env.Ctx, "nonexistent@example.com", "password123")

	// Assert error
	assert.Error(t, err)
	assert.Equal(t, services.ErrInvalidCredentials, err)
}

func TestLogin_InvalidPassword(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Call the Login method with wrong password
	_, _, err = env.AuthService.Login(env.Ctx, user.Email, "wrong-password")

	// Assert error
	assert.Error(t, err)
	assert.Equal(t, services.ErrInvalidCredentials, err)
}

func TestRefreshTokens_Success(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Login to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// Call the RefreshTokens method
	newTokens, err := env.AuthService.RefreshTokens(env.Ctx, tokens.RefreshToken)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, newTokens)

	// Verify new tokens were generated
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
}

func TestRefreshTokens_InvalidToken(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Call the RefreshTokens method with invalid token
	_, err := env.AuthService.RefreshTokens(env.Ctx, "invalid-token")

	// Assert error
	assert.Error(t, err)
}

func TestValidateToken_Success(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Login to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// Call the ValidateToken method
	token, err := env.AuthService.ValidateToken(tokens.AccessToken)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, token)

	// Verify token claims (specific assertions may vary based on implementation)
	assert.NotNil(t, token)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Call the ValidateToken method with invalid token
	_, err := env.AuthService.ValidateToken("invalid-token")

	// Assert error
	assert.Error(t, err)
}

func TestGetUserFromToken_Success(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Login to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// Call the GetUserFromToken method
	tokenUser, err := env.AuthService.GetUserFromToken(tokens.AccessToken)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, tokenUser)

	assert.Equal(t, user.ID, tokenUser.ID)
	assert.Equal(t, user.Email, tokenUser.Email)
	assert.Equal(t, user.Name, tokenUser.Name)
}

func TestGetUserFromToken_InvalidToken(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Call the GetUserFromToken method with invalid token
	_, err := env.AuthService.GetUserFromToken("invalid-token")

	// Assert error
	assert.Error(t, err)
}

func TestGetUserFromToken_UserNotFound(t *testing.T) {
	// Create a test environment with real repositories
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	userID, password := env.CreateTestUser()
	user, err := env.UserRepository.GetByID(env.Ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, user)

	// Login to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, password)
	require.NoError(t, err)
	require.NotNil(t, tokens)

	// Delete the user
	err = env.UserRepository.Delete(env.Ctx, user.ID)
	require.NoError(t, err)

	// Call the GetUserFromToken method
	_, err = env.AuthService.GetUserFromToken(tokens.AccessToken)

	// Assert error
	assert.Error(t, err)
}
