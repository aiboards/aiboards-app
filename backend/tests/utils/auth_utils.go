package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// CreateAdminUserAndGetToken creates an admin user and returns the auth token and user ID
func CreateAdminUserAndGetToken(t *testing.T, env *TestEnv) (string, uuid.UUID) {
	// Create a user
	user, err := models.NewUser("admin@example.com", "password123", "Admin User")
	require.NoError(t, err)

	// Make the user an admin
	user.IsAdmin = true

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, user)
	require.NoError(t, err)

	// Login to get token
	_, tokens, err := env.AuthService.Login(env.Ctx, "admin@example.com", "password123")
	require.NoError(t, err)

	return tokens.AccessToken, user.ID
}

// CreateRegularUserAndGetToken creates a regular user and returns the auth token and user ID
func CreateRegularUserAndGetToken(t *testing.T, env *TestEnv) (string, uuid.UUID) {
	// Create a unique email with a timestamp to avoid conflicts
	email := fmt.Sprintf("user_%d@example.com", time.Now().UnixNano())

	// Create a user
	user, err := models.NewUser(email, "password123", "Regular User")
	require.NoError(t, err)

	// Explicitly ensure the user is not an admin
	user.IsAdmin = false

	// Save user to database
	err = env.UserRepository.Create(env.Ctx, user)
	require.NoError(t, err)

	// Login to get token
	_, tokens, err := env.AuthService.Login(env.Ctx, email, "password123")
	require.NoError(t, err)

	return tokens.AccessToken, user.ID
}

// CreateTestPost creates a test post and returns it
func CreateTestPost(t *testing.T, env *TestEnv, agentID uuid.UUID) *models.Post {
	// Create a test board first
	board := models.NewBoard(agentID, "Test Board", "Test board description")

	// Insert board directly into database
	query := `
		INSERT INTO boards (id, agent_id, title, description, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := env.DB.Exec(
		query,
		board.ID,
		board.AgentID,
		board.Title,
		board.Description,
		board.IsActive,
		board.CreatedAt,
		board.UpdatedAt,
	)
	require.NoError(t, err)

	// Create a post
	post := models.NewPost(board.ID, agentID, "Test post content", nil)

	// Insert post directly into database
	query = `
		INSERT INTO posts (id, board_id, agent_id, content, vote_count, reply_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = env.DB.Exec(
		query,
		post.ID,
		post.BoardID,
		post.AgentID,
		post.Content,
		post.VoteCount,
		post.ReplyCount,
		post.CreatedAt,
		post.UpdatedAt,
	)
	require.NoError(t, err)

	return post
}

// CreateTestReply creates a test reply and returns it
func CreateTestReply(t *testing.T, env *TestEnv, agentID uuid.UUID, postID uuid.UUID) *models.Reply {
	// Create a reply
	reply := models.NewReply(string(models.ParentTypePost), postID, agentID, "Test reply content", nil)

	// Insert reply directly into database
	query := `
		INSERT INTO replies (id, parent_type, parent_id, agent_id, content, vote_count, reply_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := env.DB.Exec(
		query,
		reply.ID,
		reply.ParentType,
		reply.ParentID,
		reply.AgentID,
		reply.Content,
		reply.VoteCount,
		reply.ReplyCount,
		reply.CreatedAt,
		reply.UpdatedAt,
	)
	require.NoError(t, err)

	return reply
}
