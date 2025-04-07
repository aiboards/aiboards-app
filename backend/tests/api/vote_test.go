package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/handlers"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVoteAPI contains all components needed for vote API testing
type TestVoteAPI struct {
	Router      *gin.Engine
	Env         *utils.TestEnv
	VoteHandler *handlers.VoteHandler
	AuthToken   string
	UserID      uuid.UUID
	Agent       *models.Agent
}

// setupVoteAPITest sets up the test environment for vote API tests
func setupVoteAPITest(t *testing.T) *TestVoteAPI {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test environment
	env := utils.NewTestEnv(t)

	// Create repositories
	voteRepo := repository.NewVoteRepository(env.DB)
	postRepo := repository.NewPostRepository(env.DB)
	replyRepo := repository.NewReplyRepository(env.DB)
	repository.NewBoardRepository(env.DB)

	// Create services
	voteService := services.NewVoteService(
		voteRepo,
		postRepo,
		replyRepo,
		env.AgentRepository,
	)

	// Create handler
	voteHandler := handlers.NewVoteHandler(voteService)

	// Create router
	router := gin.Default()
	api := router.Group("/api")

	// Create auth token and user
	authToken, userID := utils.CreateRegularUserAndGetToken(t, env)

	// Create agent for the user
	agent := env.CreateTestAgent(userID)

	// Create a custom auth middleware that sets both user and agent in the context
	customAuthMiddleware := func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := parts[1]

		// Validate the token
		token, err := env.AuthService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from token
		user, err := env.AuthService.GetUserFromToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user in token"})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", user)

		// Get agent for the user
		agents, err := env.AgentRepository.GetByUserID(c, user.ID)
		if err != nil || len(agents) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No agent found for user"})
			c.Abort()
			return
		}

		// Set agent in context
		c.Set("agent", agents[0])
		c.Next()
	}

	// Register routes with custom middleware
	votes := api.Group("/votes")
	votes.Use(customAuthMiddleware)
	votes.POST("", voteHandler.CreateVote)
	votes.GET("/:id", voteHandler.GetVote)
	votes.GET("", voteHandler.GetVotesByTarget)
	votes.PUT("/:id", voteHandler.UpdateVote)
	votes.DELETE("/:id", voteHandler.DeleteVote)

	return &TestVoteAPI{
		Router:      router,
		Env:         env,
		VoteHandler: voteHandler,
		AuthToken:   authToken,
		UserID:      userID,
		Agent:       agent,
	}
}

// createTestPost creates a test post for vote tests
func (api *TestVoteAPI) createTestPost(t *testing.T) *models.Post {
	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     api.Agent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	boardRepo := repository.NewBoardRepository(api.Env.DB)
	err := boardRepo.Create(api.Env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   api.Agent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	postRepo := repository.NewPostRepository(api.Env.DB)
	err = postRepo.Create(api.Env.Ctx, post)
	require.NoError(t, err)

	return post
}

// // createTestReply creates a test reply for vote tests
// func (api *TestVoteAPI) createTestReply(t *testing.T, post *models.Post) *models.Reply {
// 	// Create a test reply
// 	reply := &models.Reply{
// 		ID:         uuid.New(),
// 		ParentType: "post",
// 		ParentID:   post.ID,
// 		AgentID:    api.Agent.ID,
// 		Content:    "Test reply",
// 		CreatedAt:  time.Now(),
// 		UpdatedAt:  time.Now(),
// 	}

// 	replyRepo := repository.NewReplyRepository(api.Env.DB)
// 	err := replyRepo.Create(api.Env.Ctx, reply)
// 	require.NoError(t, err)

// 	return reply
// }

// TestCreateVoteEndpoint tests the POST /api/votes endpoint
func TestCreateVoteEndpoint(t *testing.T) {
	api := setupVoteAPITest(t)
	defer api.Env.Cleanup()

	// Create a test post
	post := api.createTestPost(t)

	// Test data
	requestBody := map[string]interface{}{
		"target_type": "post",
		"target_id":   post.ID.String(),
		"value":       1,
	}
	jsonData, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("POST", "/api/votes", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	api.Router.ServeHTTP(w, req)

	// Print response body for debugging
	fmt.Printf("Response body: %s\n", w.Body.String())

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	assert.Equal(t, api.Agent.ID.String(), response["agent_id"])
	assert.Equal(t, "post", response["target_type"])
	assert.Equal(t, post.ID.String(), response["target_id"])
	assert.Equal(t, float64(1), response["value"])
	assert.NotEmpty(t, response["created_at"])

	// Test error case: already voted - create a new request with the same data
	jsonData, _ = json.Marshal(requestBody)
	req = httptest.NewRequest("POST", "/api/votes", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	
	// Print response body for debugging
	fmt.Printf("Already voted response: %s\n", w.Body.String())
	assert.Equal(t, http.StatusConflict, w.Code)

	// Test error case: invalid target type
	invalidRequestBody := map[string]interface{}{
		"target_type": "invalid",
		"target_id":   post.ID.String(),
		"value":       1,
	}
	jsonData, _ = json.Marshal(invalidRequestBody)
	req = httptest.NewRequest("POST", "/api/votes", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test error case: target not found
	nonExistentID := uuid.New()
	notFoundRequestBody := map[string]interface{}{
		"target_type": "post",
		"target_id":   nonExistentID.String(),
		"value":       1,
	}
	jsonData, _ = json.Marshal(notFoundRequestBody)
	req = httptest.NewRequest("POST", "/api/votes", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test error case: unauthorized
	req = httptest.NewRequest("POST", "/api/votes", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGetVoteEndpoint tests the GET /api/votes/:id endpoint
func TestGetVoteEndpoint(t *testing.T) {
	api := setupVoteAPITest(t)
	defer api.Env.Cleanup()

	// Create a test post
	post := api.createTestPost(t)

	// Create vote service
	voteService := services.NewVoteService(
		repository.NewVoteRepository(api.Env.DB),
		repository.NewPostRepository(api.Env.DB),
		repository.NewReplyRepository(api.Env.DB),
		api.Env.AgentRepository,
	)

	// Create a vote using the vote service instead of directly via repository
	vote, err := voteService.CreateVote(api.Env.Ctx, api.Agent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/votes/%s", vote.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	api.Router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	assert.Equal(t, vote.ID.String(), response["id"])
	assert.Equal(t, api.Agent.ID.String(), response["agent_id"])
	assert.Equal(t, "post", response["target_type"])
	assert.Equal(t, post.ID.String(), response["target_id"])
	assert.Equal(t, float64(1), response["value"])

	// Test error case: vote not found
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/votes/%s", uuid.New()), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test error case: unauthorized
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/votes/%s", vote.ID), nil)
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGetVotesByTargetEndpoint tests the GET /api/votes endpoint with target query params
func TestGetVotesByTargetEndpoint(t *testing.T) {
	api := setupVoteAPITest(t)
	defer api.Env.Cleanup()

	// Create a test post
	post := api.createTestPost(t)

	// Create vote service
	voteService := services.NewVoteService(
		repository.NewVoteRepository(api.Env.DB),
		repository.NewPostRepository(api.Env.DB),
		repository.NewReplyRepository(api.Env.DB),
		api.Env.AgentRepository,
	)

	// Create multiple votes from different agents using the service
	for i := 0; i < 5; i++ {
		// Create a new user and agent for each vote
		_, otherUserID := utils.CreateRegularUserAndGetToken(t, api.Env)
		otherAgent := api.Env.CreateTestAgent(otherUserID)

		_, err := voteService.CreateVote(api.Env.Ctx, otherAgent.ID, "post", post.ID, 1)
		require.NoError(t, err)
	}

	// Create request with pagination
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/votes?target_type=post&target_id=%s&page=1&page_size=3", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	api.Router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	votes := response["votes"].([]interface{})
	pagination := response["pagination"].(map[string]interface{})

	assert.Len(t, votes, 3) // Page size
	assert.Equal(t, float64(5), pagination["total"])
	assert.Equal(t, float64(1), pagination["page"])
	assert.Equal(t, float64(3), pagination["page_size"])
	assert.Equal(t, float64(2), pagination["total_pages"])

	// Test second page
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/votes?target_type=post&target_id=%s&page=2&page_size=3", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	votes = response["votes"].([]interface{})
	assert.Len(t, votes, 2) // Remaining items

	// Test error case: missing parameters
	req = httptest.NewRequest("GET", "/api/votes", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test error case: invalid target type
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/votes?target_type=invalid&target_id=%s", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Test error case: target not found
	req = httptest.NewRequest("GET", fmt.Sprintf("/api/votes?target_type=post&target_id=%s", uuid.New()), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateVoteEndpoint tests the PUT /api/votes/:id endpoint
func TestUpdateVoteEndpoint(t *testing.T) {
	api := setupVoteAPITest(t)
	defer api.Env.Cleanup()

	// Create a test post
	post := api.createTestPost(t)

	// Create vote service
	voteService := services.NewVoteService(
		repository.NewVoteRepository(api.Env.DB),
		repository.NewPostRepository(api.Env.DB),
		repository.NewReplyRepository(api.Env.DB),
		api.Env.AgentRepository,
	)

	// Create a vote using the service
	vote, err := voteService.CreateVote(api.Env.Ctx, api.Agent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Test data for update
	requestBody := map[string]interface{}{
		"value": -1, // Change from upvote to downvote
	}
	jsonData, _ := json.Marshal(requestBody)

	// Create request
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/votes/%s", vote.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	api.Router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check response data
	assert.Equal(t, vote.ID.String(), response["id"])
	assert.Equal(t, float64(-1), response["value"]) // Changed to downvote

	// Verify post vote count was updated
	postRepo := repository.NewPostRepository(api.Env.DB)
	updatedPost, err := postRepo.GetByID(api.Env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, -1, updatedPost.VoteCount) // Changed from +1 to -1

	// Test error case: vote not found
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/votes/%s", uuid.New()), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test error case: unauthorized
	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/votes/%s", vote.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test error case: forbidden (trying to update someone else's vote)
	// Create another user and agent
	otherAuthToken, otherUserID := utils.CreateRegularUserAndGetToken(t, api.Env)
	api.Env.CreateTestAgent(otherUserID)

	req = httptest.NewRequest("PUT", fmt.Sprintf("/api/votes/%s", vote.ID), bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", otherAuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// TestDeleteVoteEndpoint tests the DELETE /api/votes/:id endpoint
func TestDeleteVoteEndpoint(t *testing.T) {
	api := setupVoteAPITest(t)
	defer api.Env.Cleanup()

	// Create a test post
	post := api.createTestPost(t)

	// Create vote service
	voteService := services.NewVoteService(
		repository.NewVoteRepository(api.Env.DB),
		repository.NewPostRepository(api.Env.DB),
		repository.NewReplyRepository(api.Env.DB),
		api.Env.AgentRepository,
	)

	// Create a vote using the service
	vote, err := voteService.CreateVote(api.Env.Ctx, api.Agent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	// Create request
	req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/votes/%s", vote.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	api.Router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify vote was deleted
	voteRepo := repository.NewVoteRepository(api.Env.DB)
	deletedVote, err := voteRepo.GetByID(api.Env.Ctx, vote.ID)
	require.NoError(t, err)
	assert.Nil(t, deletedVote) // Should be nil after deletion

	// Verify post vote count was updated
	postRepo := repository.NewPostRepository(api.Env.DB)
	updatedPost, err := postRepo.GetByID(api.Env.Ctx, post.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, updatedPost.VoteCount) // Back to 0 after vote deletion

	// Test error case: vote not found
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/votes/%s", uuid.New()), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test error case: unauthorized
	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/votes/%s", vote.ID), nil)
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Test error case: forbidden (trying to delete someone else's vote)
	// Create another vote from a different agent
	_, otherUserID := utils.CreateRegularUserAndGetToken(t, api.Env)
	otherAgent := api.Env.CreateTestAgent(otherUserID)

	otherVote, err := voteService.CreateVote(api.Env.Ctx, otherAgent.ID, "post", post.ID, 1)
	require.NoError(t, err)

	req = httptest.NewRequest("DELETE", fmt.Sprintf("/api/votes/%s", otherVote.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", api.AuthToken))
	w = httptest.NewRecorder()
	api.Router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}
