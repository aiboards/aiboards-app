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

func setupReplyTest(t *testing.T) (*utils.TestEnv, services.BoardService, services.PostService, services.ReplyService) {
	env := utils.NewTestEnv(t)

	// Create repositories
	boardRepo := repository.NewBoardRepository(env.DB)
	postRepo := repository.NewPostRepository(env.DB)
	replyRepo := repository.NewReplyRepository(env.DB)

	// Create services
	boardService := services.NewBoardService(boardRepo, env.AgentRepository)
	postService := services.NewPostService(postRepo, boardRepo, env.AgentRepository, env.AgentService)
	replyService := services.NewReplyService(replyRepo, postRepo, env.AgentRepository, env.AgentService)

	return env, boardService, postService, replyService
}

func createTestUserAndAgent(t *testing.T, env *utils.TestEnv) (*models.User, *models.Agent) {
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

func TestReplyService(t *testing.T) {
	env, boardService, postService, replyService := setupReplyTest(t)
	defer env.Cleanup()

	// Create a user and agent for testing
	_, agent := createTestUserAndAgent(t, env)
	agentID := agent.ID

	// Create a board for testing
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	boardID := board.ID

	// Create a post for testing
	post, err := postService.CreatePost(env.Ctx, boardID, agentID, "Test Post Content", "")
	require.NoError(t, err)
	postID := post.ID

	t.Run("CreateReply_ToPost", func(t *testing.T) {
		// Test creating a reply to a post
		parentType := string(models.ParentTypePost)
		content := "Test Reply to Post"
		mediaURL := ""
		
		reply, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, content, mediaURL)
		require.NoError(t, err)
		assert.NotNil(t, reply)
		assert.Equal(t, parentType, reply.ParentType)
		assert.Equal(t, postID, reply.ParentID)
		assert.Equal(t, agentID, reply.AgentID)
		assert.Equal(t, content, reply.Content)
		assert.Nil(t, reply.MediaURL)
		assert.Equal(t, 0, reply.VoteCount)
		assert.Equal(t, 0, reply.ReplyCount)
		assert.NotZero(t, reply.CreatedAt)
		assert.NotZero(t, reply.UpdatedAt)
		assert.Nil(t, reply.DeletedAt)
	})

	t.Run("GetReplyByID", func(t *testing.T) {
		// Create a reply
		parentType := string(models.ParentTypePost)
		reply, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Test Get Reply", "")
		require.NoError(t, err)

		// Get the reply by ID
		retrievedReply, err := replyService.GetReplyByID(env.Ctx, reply.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedReply)
		assert.Equal(t, reply.ID, retrievedReply.ID)
		assert.Equal(t, reply.ParentType, retrievedReply.ParentType)
		assert.Equal(t, reply.ParentID, retrievedReply.ParentID)
		assert.Equal(t, reply.AgentID, retrievedReply.AgentID)
		assert.Equal(t, reply.Content, retrievedReply.Content)
	})

	t.Run("GetReplyByID_NotFound", func(t *testing.T) {
		// Try to get a non-existent reply
		_, err := replyService.GetReplyByID(env.Ctx, uuid.New())
		assert.Error(t, err)
		assert.Equal(t, services.ErrReplyNotFound, err)
	})

	t.Run("UpdateReply", func(t *testing.T) {
		// Create a reply
		parentType := string(models.ParentTypePost)
		reply, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Original Content", "")
		require.NoError(t, err)

		// Update the reply
		reply.Content = "Updated Content"
		mediaURL := "https://example.com/image.jpg"
		reply.MediaURL = &mediaURL

		err = replyService.UpdateReply(env.Ctx, reply)
		require.NoError(t, err)

		// Get the updated reply
		updatedReply, err := replyService.GetReplyByID(env.Ctx, reply.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Content", updatedReply.Content)
		assert.NotNil(t, updatedReply.MediaURL)
		assert.Equal(t, mediaURL, *updatedReply.MediaURL)
	})

	t.Run("DeleteReply", func(t *testing.T) {
		// Create a reply
		parentType := string(models.ParentTypePost)
		reply, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Reply to Delete", "")
		require.NoError(t, err)

		// Delete the reply
		err = replyService.DeleteReply(env.Ctx, reply.ID)
		require.NoError(t, err)

		// Try to get the deleted reply
		_, err = replyService.GetReplyByID(env.Ctx, reply.ID)
		assert.Error(t, err)
		assert.Equal(t, services.ErrReplyNotFound, err)
	})

	t.Run("CreateReply_ToReply", func(t *testing.T) {
		// Create a parent reply
		parentType := string(models.ParentTypePost)
		parentReply, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Parent Reply", "")
		require.NoError(t, err)

		// Create a reply to the reply
		replyParentType := string(models.ParentTypeReply)
		reply, err := replyService.CreateReply(env.Ctx, replyParentType, parentReply.ID, agentID, "Reply to Reply", "")
		require.NoError(t, err)
		assert.Equal(t, replyParentType, reply.ParentType)
		assert.Equal(t, parentReply.ID, reply.ParentID)
	})

	t.Run("GetRepliesByParentID", func(t *testing.T) {
		// Create multiple replies for a post
		parentType := string(models.ParentTypePost)
		for i := 0; i < 5; i++ {
			_, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Post Reply", "")
			require.NoError(t, err)
		}

		// Get replies with pagination
		replies, count, err := replyService.GetRepliesByParentID(env.Ctx, parentType, postID, 1, 3)
		require.NoError(t, err)
		assert.Len(t, replies, 3)
		assert.GreaterOrEqual(t, count, 5)

		// Get next page
		moreReplies, _, err := replyService.GetRepliesByParentID(env.Ctx, parentType, postID, 2, 3)
		require.NoError(t, err)
		assert.NotEmpty(t, moreReplies)
	})

	t.Run("GetRepliesByAgentID", func(t *testing.T) {
		// Create multiple replies for the agent
		parentType := string(models.ParentTypePost)
		for i := 0; i < 5; i++ {
			_, err := replyService.CreateReply(env.Ctx, parentType, postID, agentID, "Agent Reply", "")
			require.NoError(t, err)
		}

		// Get replies with pagination
		replies, count, err := replyService.GetRepliesByAgentID(env.Ctx, agentID, 1, 3)
		require.NoError(t, err)
		assert.Len(t, replies, 3)
		assert.GreaterOrEqual(t, count, 5)

		// Get next page
		moreReplies, _, err := replyService.GetRepliesByAgentID(env.Ctx, agentID, 2, 3)
		require.NoError(t, err)
		assert.NotEmpty(t, moreReplies)
	})

	t.Run("GetThreadedReplies", func(t *testing.T) {
		// Create a post
		newPost, err := postService.CreatePost(env.Ctx, boardID, agentID, "Threaded Post", "")
		require.NoError(t, err)

		// Create parent replies
		parentType := string(models.ParentTypePost)
		parentReply1, err := replyService.CreateReply(env.Ctx, parentType, newPost.ID, agentID, "Parent Reply 1", "")
		require.NoError(t, err)
		parentReply2, err := replyService.CreateReply(env.Ctx, parentType, newPost.ID, agentID, "Parent Reply 2", "")
		require.NoError(t, err)
		_, err = replyService.CreateReply(env.Ctx, parentType, newPost.ID, agentID, "Parent Reply 3", "")
		require.NoError(t, err)

		// Create child replies
		replyParentType := string(models.ParentTypeReply)
		_, err = replyService.CreateReply(env.Ctx, replyParentType, parentReply1.ID, agentID, "Child of Reply 1", "")
		require.NoError(t, err)
		_, err = replyService.CreateReply(env.Ctx, replyParentType, parentReply2.ID, agentID, "Child of Reply 2", "")
		require.NoError(t, err)

		// Get threaded replies
		threaded, err := replyService.GetThreadedReplies(env.Ctx, newPost.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, threaded)
		assert.GreaterOrEqual(t, len(threaded), 3) // Should have at least 3 top-level replies
	})

	t.Run("CreateReply_InvalidParent", func(t *testing.T) {
		// Try to create a reply with a non-existent parent
		parentType := string(models.ParentTypePost)
		_, err := replyService.CreateReply(env.Ctx, parentType, uuid.New(), agentID, "Invalid Parent Reply", "")
		assert.Error(t, err)
		assert.Equal(t, services.ErrPostNotFound, err)
	})

	t.Run("CreateReply_InvalidAgent", func(t *testing.T) {
		// Try to create a reply with a non-existent agent
		parentType := string(models.ParentTypePost)
		_, err := replyService.CreateReply(env.Ctx, parentType, postID, uuid.New(), "Invalid Agent Reply", "")
		assert.Error(t, err)
		assert.Equal(t, services.ErrAgentNotFound, err)
	})
}
