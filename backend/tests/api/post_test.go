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

func setupPostTestRouter(t *testing.T) (*gin.Engine, *utils.TestEnv, services.BoardService, services.PostService) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := utils.NewTestEnv(t)

	// Create repositories
	boardRepo := repository.NewBoardRepository(env.DB)
	postRepo := repository.NewPostRepository(env.DB)
	agentRepo := repository.NewAgentRepository(env.DB)

	// Create services
	boardService := services.NewBoardService(boardRepo, agentRepo)
	postService := services.NewPostService(postRepo, boardRepo, agentRepo, env.AgentService)

	// Create router
	router := gin.Default()

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(env.AuthService)

	// Create post handler
	postHandler := handlers.NewPostHandler(postService)

	// Setup routes
	api := router.Group("/api/v1")
	postHandler.RegisterRoutes(api, authMiddleware)

	return router, env, boardService, postService
}

func TestCreatePostEndpoint(t *testing.T) {
	router, env, boardService, _ := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Test data
	jsonStr := []byte(`{
		"agent_id": "` + agentID.String() + `",
		"board_id": "` + board.ID.String() + `",
		"content": "Test post content",
		"media_url": ""
	}`)

	// Create request
	req, _ := http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify response
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "Test post content", response["content"])
}

func TestGetPostEndpoint(t *testing.T) {
	router, env, boardService, postService := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Create a post
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Create request
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/posts/%s", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response
	var postResponse models.Post
	err = json.Unmarshal(w.Body.Bytes(), &postResponse)
	require.NoError(t, err)
	assert.Equal(t, post.ID, postResponse.ID)
	assert.Equal(t, "Test Content", postResponse.Content)
}

func TestUpdatePostEndpoint(t *testing.T) {
	router, env, boardService, postService := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Create a post
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Original Content", "")
	require.NoError(t, err)

	// Update post
	jsonStr := []byte(`{
		"id": "` + post.ID.String() + `",
		"agent_id": "` + agentID.String() + `",
		"content": "Updated post content",
		"media_url": "https://example.com/image.jpg"
	}`)

	// Create request
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/posts/%s", post.ID), bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify response
	var updatedPost models.Post
	err = json.Unmarshal(w.Body.Bytes(), &updatedPost)
	require.NoError(t, err)
	assert.Equal(t, post.ID, updatedPost.ID)
	assert.Equal(t, "Updated post content", updatedPost.Content)
	assert.NotNil(t, updatedPost.MediaURL)
	assert.Equal(t, "https://example.com/image.jpg", *updatedPost.MediaURL)
}

func TestDeletePostEndpoint(t *testing.T) {
	router, env, boardService, postService := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Create a post
	post, err := postService.CreatePost(env.Ctx, board.ID, agentID, "Test Content", "")
	require.NoError(t, err)

	// Create request
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/posts/%s", post.ID), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Create response recorder
	w := httptest.NewRecorder()

	// Perform request
	router.ServeHTTP(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify post is deleted
	_, err = postService.GetPostByID(env.Ctx, post.ID)
	assert.Error(t, err)
	assert.Equal(t, services.ErrPostNotFound, err)
}

func TestListBoardPostsEndpoint(t *testing.T) {
	router, env, boardService, postService := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Create multiple posts
	for i := 0; i < 5; i++ {
		_, err := postService.CreatePost(env.Ctx, board.ID, agentID, fmt.Sprintf("Test Content %d", i), "")
		require.NoError(t, err)
	}

	// Create request with pagination
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/posts/board/%s?page=1&page_size=3", board.ID), nil)
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

	// Check posts list
	posts, ok := response["posts"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, posts, 3)
}

func TestListAgentPostsEndpoint(t *testing.T) {
	router, env, boardService, postService := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	// Create multiple posts
	for i := 0; i < 4; i++ {
		_, err := postService.CreatePost(env.Ctx, board.ID, agentID, fmt.Sprintf("Test Content %d", i), "")
		require.NoError(t, err)
	}

	// Create request with pagination
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/posts/agent/%s?page=1&page_size=3", agentID), nil)
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

	// Check posts list
	posts, ok := response["posts"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, posts, 3)
}

func TestPostEndpointErrors(t *testing.T) {
	router, env, boardService, _ := setupPostTestRouter(t)
	defer env.Cleanup()

	// Create user, agent and get token
	token, _, agentID := createUserAgentAndGetToken(t, env)

	// Create a board
	board, err := boardService.CreateBoard(env.Ctx, agentID, "Test Board", "Test Description", true)
	require.NoError(t, err)

	t.Run("Get non-existent post returns 404", func(t *testing.T) {
		randomID := uuid.New()
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/posts/%s", randomID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Invalid post ID format returns 400", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/posts/invalid-uuid", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create post with missing fields returns 400", func(t *testing.T) {
		// Missing required fields
		jsonStr := []byte(`{
			"title": "Test Post"
			// Missing agent_id, board_id, and content
		}`)

		req, _ := http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonStr))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Create post with non-existent board returns 404", func(t *testing.T) {
		randomID := uuid.New()
		jsonStr := []byte(`{
			"agent_id": "` + agentID.String() + `",
			"board_id": "` + randomID.String() + `",
			"content": "Test post content",
			"media_url": ""
		}`)

		req, _ := http.NewRequest("POST", "/api/v1/posts", bytes.NewBuffer(jsonStr))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Unauthenticated request returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/posts/board/%s", board.ID), nil)
		// No auth token

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
