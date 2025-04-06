package integration

import (
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPostTest(t *testing.T) (*utils.TestEnv, services.BoardService, services.PostService) {
	// Create test environment
	env := utils.NewTestEnv(t)

	// Add repositories to test environment
	boardRepo := repository.NewBoardRepository(env.DB)
	postRepo := repository.NewPostRepository(env.DB)

	// Create services
	boardService := services.NewBoardService(boardRepo, env.AgentRepository)
	postService := services.NewPostService(postRepo, boardRepo, env.AgentRepository, env.AgentService)

	return env, boardService, postService
}

func createUserAndAgent(t *testing.T, env *utils.TestEnv) (*models.User, *models.Agent) {
	// Create a test user
	user := &models.User{
		ID: uuid.New(),
		// Add other required fields
	}
	userID, _ := env.CreateTestUser()

	// Create a test agent
	agent := env.CreateTestAgent(userID)

	return user, agent
}

func TestPostService(t *testing.T) {
	env, boardService, postService := setupPostTest(t)
	defer env.Cleanup()

	// Create a user and agent for testing
	_, agent := createUserAndAgent(t, env)
	agentID := agent.ID

	// Create a board for testing
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	boardID := board.ID

	t.Run("CreatePost", func(t *testing.T) {
		// Test creating a post
		post, err := postService.CreatePost(env.Ctx, boardID, agentID, "Test Post Content", "")
		require.NoError(t, err)
		assert.NotNil(t, post)
		assert.Equal(t, boardID, post.BoardID)
		assert.Equal(t, agentID, post.AgentID)
		assert.Equal(t, "Test Post Content", post.Content)
		assert.Nil(t, post.MediaURL)
		assert.Equal(t, 0, post.VoteCount)
		assert.Equal(t, 0, post.ReplyCount)
		assert.NotZero(t, post.CreatedAt)
		assert.NotZero(t, post.UpdatedAt)
		assert.Nil(t, post.DeletedAt)
	})

	t.Run("GetPostByID", func(t *testing.T) {
		// Create a post
		post, err := postService.CreatePost(env.Ctx, boardID, agentID, "Test Get Post", "")
		require.NoError(t, err)

		// Get the post by ID
		retrievedPost, err := postService.GetPostByID(env.Ctx, post.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedPost)
		assert.Equal(t, post.ID, retrievedPost.ID)
		assert.Equal(t, post.BoardID, retrievedPost.BoardID)
		assert.Equal(t, post.AgentID, retrievedPost.AgentID)
		assert.Equal(t, post.Content, retrievedPost.Content)
	})

	t.Run("GetPostByID_NotFound", func(t *testing.T) {
		// Try to get a non-existent post
		_, err := postService.GetPostByID(env.Ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, services.ErrPostNotFound, err)
	})

	t.Run("UpdatePost", func(t *testing.T) {
		// Create a post
		post, err := postService.CreatePost(env.Ctx, boardID, agentID, "Original Content", "")
		require.NoError(t, err)

		// Update the post
		post.Content = "Updated Content"
		mediaURL := "https://example.com/image.jpg"
		post.MediaURL = &mediaURL

		err = postService.UpdatePost(env.Ctx, post)
		require.NoError(t, err)

		// Get the updated post
		updatedPost, err := postService.GetPostByID(env.Ctx, post.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Content", updatedPost.Content)
		assert.NotNil(t, updatedPost.MediaURL)
		assert.Equal(t, mediaURL, *updatedPost.MediaURL)
	})

	t.Run("DeletePost", func(t *testing.T) {
		// Create a post
		post, err := postService.CreatePost(env.Ctx, boardID, agentID, "Post to Delete", "")
		require.NoError(t, err)

		// Delete the post
		err = postService.DeletePost(env.Ctx, post.ID)
		require.NoError(t, err)

		// Try to get the deleted post
		_, err = postService.GetPostByID(env.Ctx, post.ID)
		assert.Error(t, err)
		assert.Equal(t, services.ErrPostNotFound, err)
	})

	t.Run("GetPostsByBoardID", func(t *testing.T) {
		// Create multiple posts for the board
		for i := 0; i < 5; i++ {
			_, err := postService.CreatePost(env.Ctx, boardID, agentID, "Board Post", "")
			require.NoError(t, err)
		}

		// Get posts with pagination
		posts, count, err := postService.GetPostsByBoardID(env.Ctx, boardID, 1, 3)
		require.NoError(t, err)
		assert.Len(t, posts, 3)
		assert.GreaterOrEqual(t, count, 5)

		// Get next page
		morePosts, _, err := postService.GetPostsByBoardID(env.Ctx, boardID, 2, 3)
		require.NoError(t, err)
		assert.NotEmpty(t, morePosts)
	})

	t.Run("GetPostsByAgentID", func(t *testing.T) {
		// Create multiple posts for the agent
		for i := 0; i < 5; i++ {
			_, err := postService.CreatePost(env.Ctx, boardID, agentID, "Agent Post", "")
			require.NoError(t, err)
		}

		// Get posts with pagination
		posts, count, err := postService.GetPostsByAgentID(env.Ctx, agentID, 1, 3)
		require.NoError(t, err)
		assert.Len(t, posts, 3)
		assert.GreaterOrEqual(t, count, 5)

		// Get next page
		morePosts, _, err := postService.GetPostsByAgentID(env.Ctx, agentID, 2, 3)
		require.NoError(t, err)
		assert.NotEmpty(t, morePosts)
	})

	t.Run("CreatePost_InvalidBoard", func(t *testing.T) {
		// Try to create a post with a non-existent board
		_, err := postService.CreatePost(env.Ctx, uuid.New(), agentID, "Invalid Board Post", "")
		assert.Error(t, err)
		assert.Equal(t, services.ErrBoardNotFound, err)
	})

	t.Run("CreatePost_InvalidAgent", func(t *testing.T) {
		// Try to create a post with a non-existent agent
		_, err := postService.CreatePost(env.Ctx, boardID, uuid.New(), "Invalid Agent Post", "")
		assert.Error(t, err)
		assert.Equal(t, services.ErrAgentNotFound, err)
	})

	t.Run("CreatePost_InactiveBoard", func(t *testing.T) {
		// Create a board first
		inactiveBoard, err := boardService.CreateBoard(env.Ctx, agentID, "Inactive Board", "Description", false)
		require.NoError(t, err)
		
		// Explicitly set the board to inactive to ensure it overrides any default values
		err = boardService.SetBoardActive(env.Ctx, inactiveBoard.ID, false)
		require.NoError(t, err)
		
		// Verify the board is actually inactive by retrieving it
		board, err := boardService.GetBoardByID(env.Ctx, inactiveBoard.ID)
		require.NoError(t, err)
		require.NotNil(t, board)
		require.False(t, board.IsActive, "Board should be inactive")

		// Try to create a post on an inactive board
		_, err = postService.CreatePost(env.Ctx, inactiveBoard.ID, agentID, "Post on Inactive Board", "")
		assert.Error(t, err)
		assert.Equal(t, services.ErrBoardInactive, err)
	})
}
