package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/handlers"
	"github.com/garrettallen/aiboards/backend/internal/middleware"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupReplyTestRouter(t *testing.T) (*gin.Engine, *utils.TestEnv, services.BoardService, services.PostService, services.ReplyService) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := utils.NewTestEnv(t)

	// Create repositories
	boardRepo := repository.NewBoardRepository(env.DB)
	postRepo := repository.NewPostRepository(env.DB)
	replyRepo := repository.NewReplyRepository(env.DB)
	agentRepo := repository.NewAgentRepository(env.DB)

	// Create services
	boardService := services.NewBoardService(boardRepo, agentRepo)
	postService := services.NewPostService(postRepo, boardRepo, agentRepo, env.AgentService)
	replyService := services.NewReplyService(replyRepo, postRepo, agentRepo, env.AgentService)

	// Create router
	router := gin.Default()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(env.AuthService)

	// Create reply handler
	replyHandler := handlers.NewReplyHandler(replyService)

	// Setup routes
	api := router.Group("/api/v1")
	replyHandler.RegisterRoutes(api, authMiddleware)

	return router, env, boardService, postService, replyService
}

func TestCreateReplyEndpoint(t *testing.T) {
	router, env, boardService, postService, _ := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board and post
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Test data
	parentType := string(models.ParentTypePost)
	jsonStr := []byte(`{
		"parent_type": "` + parentType + `",
		"parent_id": "` + post.ID.String() + `",
		"agent_id": "` + agentID.String() + `",
		"content": "Test reply content",
		"media_url": ""
	}`)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/replies", bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response data
	assert.Equal(t, "Test reply content", response["content"])
	assert.Equal(t, agentID.String(), response["agent_id"])
	assert.Equal(t, post.ID.String(), response["parent_id"])
	assert.Equal(t, string(models.ParentTypePost), response["parent_type"])
	assert.NotEmpty(t, response["id"])
}

func TestGetReplyEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board, post, and reply
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)
	parentType := string(models.ParentTypePost)
	reply, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, "Test Reply Content", "")
	require.NoError(t, err)

	// Create request
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/%s", reply.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response data
	assert.Equal(t, reply.ID.String(), response["id"])
	assert.Equal(t, reply.Content, response["content"])
	assert.Equal(t, reply.AgentID.String(), response["agent_id"])
	assert.Equal(t, reply.ParentID.String(), response["parent_id"])
	assert.Equal(t, string(reply.ParentType), response["parent_type"])
}

func TestUpdateReplyEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board, post, and reply
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)
	parentType := string(models.ParentTypePost)
	reply, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, "Original Content", "")
	require.NoError(t, err)

	// Test data for update
	jsonStr := []byte(`{
		"content": "Updated Content",
		"media_url": ""
	}`)

	// Create request
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/replies/%s", reply.ID), bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Get updated reply
	updatedReply, err := replyService.GetReplyByID(env.Ctx, reply.ID)
	require.NoError(t, err)

	// Check updated data
	assert.Equal(t, "Updated Content", updatedReply.Content)
}

func TestDeleteReplyEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board, post, and reply
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)
	parentType := string(models.ParentTypePost)
	reply, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, "Test Reply Content", "")
	require.NoError(t, err)

	// Create request
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/replies/%s", reply.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify reply is deleted
	_, err = replyService.GetReplyByID(env.Ctx, reply.ID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrReplyNotFound, err)
}

func TestListRepliesEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board and post
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Create multiple replies for the post
	parentType := string(models.ParentTypePost)
	for i := 0; i < 5; i++ {
		_, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, fmt.Sprintf("Reply %d", i), "")
		require.NoError(t, err)
	}

	// Create request with pagination
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/parent/%s?parent_type=post&page=1&page_size=3", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check pagination
	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(3), response["page_size"])
	assert.Equal(t, float64(5), response["total_count"])

	// Check replies list
	replies, ok := response["replies"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, replies, 3)
}

func TestListAgentRepliesEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board and post
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Create multiple replies for the agent
	parentType := string(models.ParentTypePost)
	for i := 0; i < 4; i++ {
		_, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, fmt.Sprintf("Reply %d", i), "")
		require.NoError(t, err)
	}

	// Create request with pagination
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/agent/%s?page=1&page_size=3", agentID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check pagination
	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(3), response["page_size"])
	assert.Equal(t, float64(4), response["total_count"])

	// Check replies list
	replies, ok := response["replies"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, replies, 3)
}

func TestGetThreadedRepliesEndpoint(t *testing.T) {
	router, env, boardService, postService, replyService := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board and post
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Create a thread of replies (post -> reply1 -> reply2 -> reply3)
	parentType := string(models.ParentTypePost)
	reply1, err := replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, "Reply 1", "")
	require.NoError(t, err)

	replyParentType := string(models.ParentTypeReply)
	reply2, err := replyService.CreateReply(env.Ctx, replyParentType, reply1.ID, agentID, "Reply 2", "")
	require.NoError(t, err)

	_, err = replyService.CreateReply(env.Ctx, replyParentType, reply2.ID, agentID, "Reply 3", "")
	require.NoError(t, err)

	// Also create some direct replies to the post
	_, err = replyService.CreateReply(env.Ctx, parentType, post.ID, agentID, "Another direct reply", "")
	require.NoError(t, err)

	// Create request
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/thread/%s", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check threaded replies
	replies, ok := response["replies"].([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(replies), 2) // At least 2 direct replies to post
}

func TestReplyEndpointErrors(t *testing.T) {
	router, env, boardService, postService, _ := setupReplyTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board and post
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	t.Run("Get non-existent reply returns 404", func(t *testing.T) {
		randomID := uuid.New()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/%s", randomID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid reply ID format returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/replies/invalid-uuid", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create reply with missing fields returns 400", func(t *testing.T) {
		// Missing required fields
		jsonStr := []byte(`{
			"content": "Test Reply",
			"media_url": ""
			// Missing agent_id, parent_id, and parent_type
		}`)

		req, _ := http.NewRequest("POST", "/api/v1/replies", bytes.NewBuffer(jsonStr))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create reply with non-existent parent returns 404", func(t *testing.T) {
		randomID := uuid.New()
		jsonStr := []byte(`{
			"parent_type": "post",
			"parent_id": "` + randomID.String() + `",
			"agent_id": "` + agentID.String() + `",
			"content": "Test reply content",
			"media_url": ""
		}`)

		req, _ := http.NewRequest("POST", "/api/v1/replies", bytes.NewBuffer(jsonStr))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Create reply with invalid parent type returns 400", func(t *testing.T) {
		jsonStr := []byte(`{
			"parent_type": "invalid_type",
			"parent_id": "` + post.ID.String() + `",
			"content": "Test reply content",
			"media_url": ""
		}`)

		req, _ := http.NewRequest("POST", "/api/v1/replies", bytes.NewBuffer(jsonStr))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Unauthenticated request returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/replies/parent/%s", post.ID), nil)
		// No auth token

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
