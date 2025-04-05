package unit

import (
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/stretchr/testify/assert"
)

func TestCreateAgent(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user first
	testUser, err := models.NewUser("agent-test@example.com", "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Test creating an agent
	name := "Test Agent"
	description := "This is a test agent"
	dailyLimit := 100

	agent, err := env.AgentService.CreateAgent(env.Ctx, testUser.ID, name, description, dailyLimit)
	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.Equal(t, name, agent.Name)
	assert.Equal(t, description, agent.Description)
	assert.Equal(t, dailyLimit, agent.DailyLimit)
	assert.Equal(t, testUser.ID, agent.UserID)
	assert.NotEmpty(t, agent.APIKey)
}

func TestGetAgentByID(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	testUser, agent := createTestUserAndAgent(t, env)

	// Test getting agent by ID
	retrievedAgent, err := env.AgentService.GetAgentByID(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedAgent)
	assert.Equal(t, agent.ID, retrievedAgent.ID)
	assert.Equal(t, agent.Name, retrievedAgent.Name)
	assert.Equal(t, agent.Description, retrievedAgent.Description)
	assert.Equal(t, testUser.ID, retrievedAgent.UserID)
}

func TestGetAgentsByUserID(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user
	testUser, err := models.NewUser("agents-test@example.com", "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Create multiple agents for the user
	for i := 0; i < 3; i++ {
		_, err := env.AgentService.CreateAgent(
			env.Ctx,
			testUser.ID,
			"Agent "+string(rune(i+65)), // A, B, C
			"Description "+string(rune(i+65)),
			100,
		)
		assert.NoError(t, err)
	}

	// Test getting agents by user ID
	agents, err := env.AgentService.GetAgentsByUserID(env.Ctx, testUser.ID)
	assert.NoError(t, err)
	assert.Len(t, agents, 3)
}

func TestUpdateAgent(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	_, agent := createTestUserAndAgent(t, env)

	// Update agent
	agent.Name = "Updated Agent Name"
	agent.Description = "Updated description"
	agent.DailyLimit = 200

	err := env.AgentService.UpdateAgent(env.Ctx, agent)
	assert.NoError(t, err)

	// Verify update
	updatedAgent, err := env.AgentService.GetAgentByID(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Agent Name", updatedAgent.Name)
	assert.Equal(t, "Updated description", updatedAgent.Description)
	assert.Equal(t, 200, updatedAgent.DailyLimit)
}

func TestRegenerateAPIKey(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	_, agent := createTestUserAndAgent(t, env)

	// Store original API key
	originalAPIKey := agent.APIKey

	// Regenerate API key
	newAPIKey, err := env.AgentService.RegenerateAPIKey(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.NotEmpty(t, newAPIKey)
	assert.NotEqual(t, originalAPIKey, newAPIKey)

	// Verify the agent has the new API key
	updatedAgent, err := env.AgentService.GetAgentByID(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.Equal(t, newAPIKey, updatedAgent.APIKey)
}

func TestDeleteAgent(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	_, agent := createTestUserAndAgent(t, env)

	// Delete agent
	err := env.AgentService.DeleteAgent(env.Ctx, agent.ID)
	assert.NoError(t, err)

	// Verify deletion - agent should not be found after deletion
	_, err = env.AgentService.GetAgentByID(env.Ctx, agent.ID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrAgentNotFound, err)
}

func TestIncrementUsage(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	_, agent := createTestUserAndAgent(t, env)

	// Initial usage should be 0
	assert.Equal(t, 0, agent.UsedToday)

	// Increment usage
	err := env.AgentService.IncrementUsage(env.Ctx, agent.ID)
	assert.NoError(t, err)

	// Verify usage incremented
	updatedAgent, err := env.AgentService.GetAgentByID(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, updatedAgent.UsedToday)
}

func TestCheckRateLimit(t *testing.T) {
	// Create test environment
	env := utils.NewTestEnv(t)
	defer env.Cleanup()

	// Create a test user and agent with low daily limit
	testUser, err := models.NewUser("ratelimit-test@example.com", "password123", "Test User")
	assert.NoError(t, err)
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	agent, err := env.AgentService.CreateAgent(env.Ctx, testUser.ID, "Rate Limited Agent", "Test", 2)
	assert.NoError(t, err)

	// Initially should not be rate limited
	limited, err := env.AgentService.CheckRateLimit(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.False(t, limited)

	// Increment usage twice to reach limit
	err = env.AgentService.IncrementUsage(env.Ctx, agent.ID)
	assert.NoError(t, err)
	err = env.AgentService.IncrementUsage(env.Ctx, agent.ID)
	assert.NoError(t, err)

	// Now should be rate limited
	limited, err = env.AgentService.CheckRateLimit(env.Ctx, agent.ID)
	assert.NoError(t, err)
	assert.True(t, limited)
}

// Helper function to create a test user and agent
func createTestUserAndAgent(t *testing.T, env *utils.TestEnv) (*models.User, *models.Agent) {
	// Create a test user
	testUser, err := models.NewUser("agent-test@example.com", "password123", "Test User")
	assert.NoError(t, err)

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, testUser)
	assert.NoError(t, err)

	// Create an agent
	agent, err := env.AgentService.CreateAgent(
		env.Ctx,
		testUser.ID,
		"Test Agent",
		"This is a test agent",
		100,
	)
	assert.NoError(t, err)

	return testUser, agent
}
