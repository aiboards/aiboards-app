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

// TestVoteEnv extends the base test environment with vote-specific components
type TestVoteEnv struct {
	*utils.TestEnv
	VoteService         services.VoteService
	VoteRepository      repository.VoteRepository
	PostRepository      repository.PostRepository
	ReplyRepository     repository.ReplyRepository
	BoardRepository     repository.BoardRepository
	NotificationService services.NotificationService
}

// NewTestVoteEnv creates a new test environment with vote components
func NewTestVoteEnv(t *testing.T) *TestVoteEnv {
	baseEnv := utils.NewTestEnv(t)

	// Create repositories
	voteRepo := repository.NewVoteRepository(baseEnv.DB)
	postRepo := repository.NewPostRepository(baseEnv.DB)
	replyRepo := repository.NewReplyRepository(baseEnv.DB)
	boardRepo := repository.NewBoardRepository(baseEnv.DB)
	notificationRepo := repository.NewNotificationRepository(baseEnv.DB)
	userRepo := repository.NewUserRepository(baseEnv.DB)

	// Create notification service for vote notifications
	notificationService := services.NewNotificationService(
		notificationRepo,
		userRepo,
		baseEnv.AgentRepository,
	)

	// Create vote service
	voteService := services.NewVoteService(
		voteRepo,
		postRepo,
		replyRepo,
		baseEnv.AgentRepository,
	)

	return &TestVoteEnv{
		TestEnv:             baseEnv,
		VoteService:         voteService,
		VoteRepository:      voteRepo,
		PostRepository:      postRepo,
		ReplyRepository:     replyRepo,
		BoardRepository:     boardRepo,
		NotificationService: notificationService,
	}
}

// TestCreateVote_Integration tests creating a vote
func TestCreateVote_Integration(t *testing.T) {
	// Create test environment
	env := NewTestVoteEnv(t)
	defer env.Cleanup()

	// Create test users and agents
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	voterUserID, _ := env.CreateTestUser()
	voterAgent := env.CreateTestAgent(voterUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Test creating an upvote
	upvote, err := env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", post.ID, 1)
	require.NoError(t, err)
	require.NotNil(t, upvote)

	// Verify vote properties
	assert.Equal(t, voterAgent.ID, upvote.AgentID)
	assert.Equal(t, "post", upvote.TargetType)
	assert.Equal(t, post.ID, upvote.TargetID)
	assert.Equal(t, 1, upvote.Value)

	// Verify post vote count was updated
	updatedPost, err := env.PostRepository.GetByID(env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedPost.VoteCount)

	// Test creating a downvote on a reply
	reply := &models.Reply{
		ID:         uuid.New(),
		ParentType: "post",
		ParentID:   post.ID,
		AgentID:    postOwnerAgent.ID,
		Content:    "Test reply",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	err = env.ReplyRepository.Create(env.Ctx, reply)
	require.NoError(t, err)

	downvote, err := env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "reply", reply.ID, -1)
	require.NoError(t, err)
	require.NotNil(t, downvote)

	// Verify vote properties
	assert.Equal(t, voterAgent.ID, downvote.AgentID)
	assert.Equal(t, "reply", downvote.TargetType)
	assert.Equal(t, reply.ID, downvote.TargetID)
	assert.Equal(t, -1, downvote.Value)

	// Verify reply vote count was updated
	updatedReply, err := env.ReplyRepository.GetByID(env.Ctx, reply.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, updatedReply.VoteCount)

	// Test error case: already voted
	_, err = env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", post.ID, 1)
	assert.Equal(t, services.ErrAlreadyVoted, err)

	// Test error case: invalid target type
	_, err = env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "invalid", post.ID, 1)
	assert.Equal(t, services.ErrInvalidTargetType, err)

	// Test error case: target not found
	_, err = env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", uuid.New(), 1)
	assert.Equal(t, services.ErrTargetNotFound, err)
}

// TestGetVoteByID_Integration tests retrieving a vote by ID
func TestGetVoteByID_Integration(t *testing.T) {
	// Create test environment
	env := NewTestVoteEnv(t)
	defer env.Cleanup()

	// Create test users and agents
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	voterUserID, _ := env.CreateTestUser()
	voterAgent := env.CreateTestAgent(voterUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create a vote
	vote, err := env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Test retrieving the vote by ID
	retrievedVote, err := env.VoteService.GetVoteByID(env.Ctx, vote.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedVote)

	// Verify vote properties
	assert.Equal(t, vote.ID, retrievedVote.ID)
	assert.Equal(t, voterAgent.ID, retrievedVote.AgentID)
	assert.Equal(t, "post", retrievedVote.TargetType)
	assert.Equal(t, post.ID, retrievedVote.TargetID)
	assert.Equal(t, 1, retrievedVote.Value)

	// Test error case: vote not found
	_, err = env.VoteService.GetVoteByID(env.Ctx, uuid.New())
	assert.Equal(t, services.ErrVoteNotFound, err)
}

// TestGetVotesByTargetID_Integration tests retrieving votes for a target
func TestGetVotesByTargetID_Integration(t *testing.T) {
	// Create test environment
	env := NewTestVoteEnv(t)
	defer env.Cleanup()

	// Create test users and agents
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	// Create voter agents
	var voterAgents []*models.Agent
	for i := 0; i < 5; i++ {
		voterUserID, _ := env.CreateTestUser()
		voterAgent := env.CreateTestAgent(voterUserID)
		voterAgents = append(voterAgents, voterAgent)
	}

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create votes
	for i, agent := range voterAgents {
		value := 1
		if i%2 == 1 {
			value = -1 // Alternate between upvotes and downvotes
		}
		_, err := env.VoteService.CreateVote(env.Ctx, agent.ID, "post", post.ID, value)
		require.NoError(t, err)
	}

	// Test retrieving votes with pagination
	votes, count, err := env.VoteService.GetVotesByTargetID(env.Ctx, "post", post.ID, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, count) // Total count
	assert.Len(t, votes, 3)   // Page size

	// Test second page
	votes, count, err = env.VoteService.GetVotesByTargetID(env.Ctx, "post", post.ID, 2, 3)
	require.NoError(t, err)
	assert.Equal(t, 5, count) // Total count
	assert.Len(t, votes, 2)   // Remaining items

	// Test error case: invalid target type
	_, _, err = env.VoteService.GetVotesByTargetID(env.Ctx, "invalid", post.ID, 1, 10)
	assert.Equal(t, services.ErrInvalidTargetType, err)

	// Test error case: target not found
	_, _, err = env.VoteService.GetVotesByTargetID(env.Ctx, "post", uuid.New(), 1, 10)
	assert.Equal(t, services.ErrTargetNotFound, err)
}

// TestUpdateVote_Integration tests updating a vote
func TestUpdateVote_Integration(t *testing.T) {
	// Create test environment
	env := NewTestVoteEnv(t)
	defer env.Cleanup()

	// Create test users and agents
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	voterUserID, _ := env.CreateTestUser()
	voterAgent := env.CreateTestAgent(voterUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create a vote
	vote, err := env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Verify initial post vote count
	updatedPost, err := env.PostRepository.GetByID(env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedPost.VoteCount)

	// Update the vote (change from upvote to downvote)
	vote.Value = -1
	err = env.VoteService.UpdateVote(env.Ctx, vote)
	require.NoError(t, err)

	// Verify vote was updated
	updatedVote, err := env.VoteService.GetVoteByID(env.Ctx, vote.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, updatedVote.Value)

	// Verify post vote count was updated
	updatedPost, err = env.PostRepository.GetByID(env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, updatedPost.VoteCount) // Changed from +1 to -1

	// Test error case: vote not found
	nonExistentVote := &models.Vote{
		ID:         uuid.New(),
		AgentID:    voterAgent.ID,
		TargetType: "post",
		TargetID:   post.ID,
		Value:      1,
	}
	err = env.VoteService.UpdateVote(env.Ctx, nonExistentVote)
	assert.Equal(t, services.ErrVoteNotFound, err)
}

// TestDeleteVote_Integration tests deleting a vote
func TestDeleteVote_Integration(t *testing.T) {
	// Create test environment
	env := NewTestVoteEnv(t)
	defer env.Cleanup()

	// Create test users and agents
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	voterUserID, _ := env.CreateTestUser()
	voterAgent := env.CreateTestAgent(voterUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create a vote
	vote, err := env.VoteService.CreateVote(env.Ctx, voterAgent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Verify initial post vote count
	updatedPost, err := env.PostRepository.GetByID(env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, updatedPost.VoteCount)

	// Delete the vote
	err = env.VoteService.DeleteVote(env.Ctx, vote.ID)
	require.NoError(t, err)

	// Verify vote was deleted (soft delete)
	_, err = env.VoteService.GetVoteByID(env.Ctx, vote.ID)
	assert.Equal(t, services.ErrVoteNotFound, err)

	// Verify post vote count was updated
	updatedPost, err = env.PostRepository.GetByID(env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updatedPost.VoteCount) // Back to 0 after vote deletion

	// Test error case: vote not found
	err = env.VoteService.DeleteVote(env.Ctx, uuid.New())
	assert.Equal(t, services.ErrVoteNotFound, err)
}
