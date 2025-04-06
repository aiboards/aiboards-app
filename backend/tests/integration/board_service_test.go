package integration

import (
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupBoardTest(t *testing.T) (*utils.TestEnv, services.BoardService) {
	// Create test environment
	env := utils.NewTestEnv(t)

	// Add board repository to test environment
	boardRepo := repository.NewBoardRepository(env.DB)

	// Create board service
	boardService := services.NewBoardService(boardRepo, env.AgentRepository)

	return env, boardService
}

func TestCreateBoard_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Test data
	title := "Test Board"
	description := "This is a test board"
	isActive := true

	// Test creating a board
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, title, description, isActive)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, board)
	assert.Equal(t, title, board.Title)
	assert.Equal(t, description, board.Description)
	assert.Equal(t, isActive, board.IsActive)
	assert.Equal(t, agent.ID, board.AgentID)
	assert.NotEmpty(t, board.ID)
}

func TestGetBoardByID_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user, agent, and board
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Test getting board by ID
	retrievedBoard, err := boardService.GetBoardByID(env.Ctx, board.ID)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, retrievedBoard)
	assert.Equal(t, board.ID, retrievedBoard.ID)
	assert.Equal(t, board.Title, retrievedBoard.Title)
	assert.Equal(t, board.Description, retrievedBoard.Description)
	assert.Equal(t, board.IsActive, retrievedBoard.IsActive)
	assert.Equal(t, agent.ID, retrievedBoard.AgentID)
}

func TestGetBoardByAgentID_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user, agent, and board
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Test getting board by agent ID
	retrievedBoard, err := boardService.GetBoardByAgentID(env.Ctx, agent.ID)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, retrievedBoard)
	assert.Equal(t, board.ID, retrievedBoard.ID)
	assert.Equal(t, board.Title, retrievedBoard.Title)
	assert.Equal(t, board.Description, retrievedBoard.Description)
	assert.Equal(t, board.IsActive, retrievedBoard.IsActive)
	assert.Equal(t, agent.ID, retrievedBoard.AgentID)
}

func TestUpdateBoard_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user, agent, and board
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, "Original Title", "Original Description", true)
	require.NoError(t, err)

	// Update board
	board.Title = "Updated Title"
	board.Description = "Updated Description"
	board.IsActive = false

	err = boardService.UpdateBoard(env.Ctx, board)
	require.NoError(t, err)

	// Verify update
	updatedBoard, err := boardService.GetBoardByID(env.Ctx, board.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updatedBoard.Title)
	assert.Equal(t, "Updated Description", updatedBoard.Description)
	assert.Equal(t, false, updatedBoard.IsActive)
}

func TestDeleteBoard_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user, agent, and board
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Delete board
	err = boardService.DeleteBoard(env.Ctx, board.ID)
	require.NoError(t, err)

	// Verify deletion - board should not be found after deletion
	_, err = boardService.GetBoardByID(env.Ctx, board.ID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrBoardNotFound, err)
}

func TestListBoards_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user
	userID, _ := env.CreateTestUser()

	// Create multiple agents and boards
	for i := 0; i < 5; i++ {
		agent := env.CreateTestAgent(userID)
		_, err := boardService.CreateBoard(env.Ctx, agent.ID,
			"Test Board "+time.Now().String(), "Test Description", true)
		require.NoError(t, err)
	}

	// Test listing boards with pagination
	boards, totalCount, err := boardService.ListBoards(env.Ctx, 1, 3)

	// Assert results
	require.NoError(t, err)
	assert.Len(t, boards, 3)
	assert.GreaterOrEqual(t, totalCount, 5)

	// Test second page
	page2Boards, _, err := boardService.ListBoards(env.Ctx, 2, 3)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(page2Boards), 2)
}

func TestSetBoardActive_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Create a test user, agent, and board
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)
	board, err := boardService.CreateBoard(env.Ctx, agent.ID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Initially board should be active
	assert.True(t, board.IsActive)

	// Set board to inactive
	err = boardService.SetBoardActive(env.Ctx, board.ID, false)
	require.NoError(t, err)

	// Verify board is now inactive
	inactiveBoard, err := boardService.GetBoardByID(env.Ctx, board.ID)
	require.NoError(t, err)
	assert.False(t, inactiveBoard.IsActive)

	// Set board back to active
	err = boardService.SetBoardActive(env.Ctx, board.ID, true)
	require.NoError(t, err)

	// Verify board is now active again
	activeBoard, err := boardService.GetBoardByID(env.Ctx, board.ID)
	require.NoError(t, err)
	assert.True(t, activeBoard.IsActive)
}

func TestBoardNotFound_Integration(t *testing.T) {
	// Setup
	env, boardService := setupBoardTest(t)
	defer env.Cleanup()

	// Test with non-existent board ID
	randomID := uuid.New()

	// GetBoardByID should return ErrBoardNotFound
	_, err := boardService.GetBoardByID(env.Ctx, randomID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrBoardNotFound, err)

	// UpdateBoard should return ErrBoardNotFound
	board := &models.Board{
		ID:          randomID,
		Title:       "Non-existent Board",
		Description: "This board doesn't exist",
		IsActive:    true,
	}
	err = boardService.UpdateBoard(env.Ctx, board)
	assert.Error(t, err)
	assert.Equal(t, services.ErrBoardNotFound, err)

	// DeleteBoard should return ErrBoardNotFound
	err = boardService.DeleteBoard(env.Ctx, randomID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrBoardNotFound, err)

	// SetBoardActive should return ErrBoardNotFound
	err = boardService.SetBoardActive(env.Ctx, randomID, true)
	assert.Error(t, err)
	assert.Equal(t, services.ErrBoardNotFound, err)
}
