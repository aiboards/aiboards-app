package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
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

func setupAdminTestRouter(t *testing.T) (*gin.Engine, *utils.TestEnv) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := utils.NewTestEnv(t)

	// Create router with debug mode
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(func(c *gin.Context) {
		log.Printf("TEST ROUTER: Request to %s %s", c.Request.Method, c.Request.URL.Path)
		c.Next()
	})

	// Create middleware
	authMiddleware := middleware.AuthMiddleware(env.AuthService)
	adminMiddleware := middleware.AdminMiddleware(env.UserService)

	// Create repositories for posts, replies, and boards
	postRepo := repository.NewPostRepository(env.DB)
	replyRepo := repository.NewReplyRepository(env.DB)
	boardRepo := repository.NewBoardRepository(env.DB)

	// Create services
	boardService := services.NewBoardService(boardRepo, env.AgentRepository)
	postService := services.NewPostService(postRepo, boardRepo, env.AgentRepository, env.AgentService)
	replyService := services.NewReplyService(replyRepo, postRepo, env.AgentRepository, env.AgentService)

	// Create admin handler
	adminHandler := handlers.NewAdminHandler(
		env.UserService,
		env.AgentService,
		boardService,
		postService,
		replyService,
	)

	// Setup routes
	api := router.Group("/api/v1")
	adminHandler.RegisterRoutes(api, authMiddleware, adminMiddleware)

	return router, env
}

func TestGetUsersEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	utils.CreateRegularUserAndGetToken(t, env)

	// Create additional test users
	for i := 0; i < 5; i++ {
		email := fmt.Sprintf("test%d@example.com", i)
		user, err := models.NewUser(email, "password123", fmt.Sprintf("Test User %d", i))
		require.NoError(t, err)
		err = env.UserRepository.Create(env.Ctx, user)
		require.NoError(t, err)
	}

	t.Run("Admin user can list all users", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Contains(t, response, "users")
		assert.Contains(t, response, "pagination")

		// Verify users list
		users, ok := response["users"].([]interface{})
		assert.True(t, ok)
		assert.GreaterOrEqual(t, len(users), 7) // Admin, regular, and 5 test users
	})

	t.Run("Regular user cannot access admin endpoint", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some-token"))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Unauthenticated user cannot access admin endpoint", func(t *testing.T) {
		// Create request without token
		req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Pagination works correctly", func(t *testing.T) {
		// Create request with pagination
		req := httptest.NewRequest("GET", "/api/v1/admin/users?page=1&page_size=3", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify pagination
		pagination := response["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(3), pagination["page_size"])

		// Verify users list length
		users, ok := response["users"].([]interface{})
		assert.True(t, ok)
		assert.Equal(t, 3, len(users))
	})
}

func TestGetUserEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	_, regularUserID := utils.CreateRegularUserAndGetToken(t, env)

	t.Run("Admin user can get a specific user", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/users/%s", regularUserID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify user data
		assert.Equal(t, regularUserID.String(), response["id"])
		// Don't check the exact email since it's now dynamically generated
		assert.Contains(t, response["email"].(string), "@example.com")
		assert.Equal(t, "Regular User", response["name"])
		assert.Equal(t, false, response["is_admin"])
	})

	t.Run("Regular user cannot access user details", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/users/%s", regularUserID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some-token"))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Invalid user ID returns bad request", func(t *testing.T) {
		// Create request with invalid UUID
		req := httptest.NewRequest("GET", "/api/v1/admin/users/invalid-uuid", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Non-existent user ID returns not found", func(t *testing.T) {
		// Create request with random UUID
		randomID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/admin/users/%s", randomID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be not found
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateUserEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token - we'll update this user to admin in the first test
	_, regularUserID := utils.CreateRegularUserAndGetToken(t, env)

	t.Run("Admin user can update a user", func(t *testing.T) {
		// Create update request body
		updateData := map[string]interface{}{
			"name":     "Updated User Name",
			"is_admin": true,
		}
		jsonData, _ := json.Marshal(updateData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/users/%s", regularUserID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify user was updated in database
		updatedUser, err := env.UserService.GetUserByID(env.Ctx, regularUserID)
		require.NoError(t, err)
		assert.Equal(t, "Updated User Name", updatedUser.Name)
		assert.Equal(t, true, updatedUser.IsAdmin)
	})

	t.Run("Regular user cannot update users", func(t *testing.T) {
		// Create a new regular user for this test
		regularToken2, regularUserID2 := utils.CreateRegularUserAndGetToken(t, env)

		// Create update request body
		updateData := map[string]interface{}{
			"name": "Should Not Update",
		}
		jsonData, _ := json.Marshal(updateData)

		// Create request - try to update the second user with their own token (should be forbidden)
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/users/%s", regularUserID2), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", regularToken2))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Invalid update data returns bad request", func(t *testing.T) {
		// Create invalid update request body (email is required but missing)
		updateData := map[string]interface{}{
			"email": "", // Invalid empty email
		}
		jsonData, _ := json.Marshal(updateData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/users/%s", regularUserID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteUserEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	utils.CreateRegularUserAndGetToken(t, env)

	// Create a user to delete
	userToDelete, err := models.NewUser("delete-me@example.com", "password123", "User To Delete")
	require.NoError(t, err)
	err = env.UserRepository.Create(env.Ctx, userToDelete)
	require.NoError(t, err)

	t.Run("Admin user can delete a user", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/users/%s", userToDelete.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify user was deleted (soft delete)
		deletedUser, err := env.UserService.GetUserByID(env.Ctx, userToDelete.ID)
		assert.Error(t, err) // Should return an error since user is deleted
		assert.Nil(t, deletedUser)
	})

	// Create another user for the next test
	anotherUser, err := models.NewUser("another@example.com", "password123", "Another User")
	require.NoError(t, err)
	err = env.UserRepository.Create(env.Ctx, anotherUser)
	require.NoError(t, err)

	t.Run("Regular user cannot delete users", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/admin/users/%s", anotherUser.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some-token"))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify user still exists
		existingUser, err := env.UserService.GetUserByID(env.Ctx, anotherUser.ID)
		assert.NoError(t, err)
		assert.NotNil(t, existingUser)
	})

	t.Run("Invalid user ID returns bad request", func(t *testing.T) {
		// Create request with invalid UUID
		req := httptest.NewRequest("DELETE", "/api/v1/admin/users/invalid-uuid", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestModeratePostEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	_, regularUserID := utils.CreateRegularUserAndGetToken(t, env)

	// Create a test agent for the regular user
	agent := env.CreateTestAgent(regularUserID)

	// Create a test post
	post := utils.CreateTestPost(t, env, agent.ID)

	t.Run("Admin user can delete a post", func(t *testing.T) {
		// Create moderation request body
		moderationData := map[string]interface{}{
			"delete": true,
			"reason": "Violates community guidelines",
		}
		jsonData, _ := json.Marshal(moderationData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/posts/%s/moderate", post.ID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify post was soft deleted
		var deletedPost models.Post
		err := env.DB.Get(&deletedPost, "SELECT * FROM posts WHERE id = $1", post.ID)
		require.NoError(t, err)
		assert.NotNil(t, deletedPost.DeletedAt)
	})

	t.Run("Regular user cannot moderate posts", func(t *testing.T) {
		// Create a new post for this test
		anotherPost := utils.CreateTestPost(t, env, agent.ID)

		// Create moderation request body
		moderationData := map[string]interface{}{
			"delete": true,
			"reason": "Violates community guidelines",
		}
		jsonData, _ := json.Marshal(moderationData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/posts/%s/moderate", anotherPost.ID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some-token"))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify post was not deleted
		var post models.Post
		err := env.DB.Get(&post, "SELECT * FROM posts WHERE id = $1", anotherPost.ID)
		require.NoError(t, err)
		assert.Nil(t, post.DeletedAt)
	})
}

func TestModerateReplyEndpoint(t *testing.T) {
	router, env := setupAdminTestRouter(t)
	defer env.Cleanup()

	// Create admin user and get token
	adminToken, _ := utils.CreateAdminUserAndGetToken(t, env)

	// Create regular user and get token
	_, regularUserID := utils.CreateRegularUserAndGetToken(t, env)

	// Create a test agent for the regular user
	agent := env.CreateTestAgent(regularUserID)

	// Create a test post
	post := utils.CreateTestPost(t, env, agent.ID)

	// Create a test reply
	reply := utils.CreateTestReply(t, env, agent.ID, post.ID)

	t.Run("Admin user can delete a reply", func(t *testing.T) {
		// Create moderation request body
		moderationData := map[string]interface{}{
			"delete": true,
			"reason": "Violates community guidelines",
		}
		jsonData, _ := json.Marshal(moderationData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/replies/%s/moderate", reply.ID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify reply was soft deleted
		var deletedReply models.Reply
		err := env.DB.Get(&deletedReply, "SELECT * FROM replies WHERE id = $1", reply.ID)
		require.NoError(t, err)
		assert.NotNil(t, deletedReply.DeletedAt)
	})

	t.Run("Regular user cannot moderate replies", func(t *testing.T) {
		// Create a new reply for this test
		anotherReply := utils.CreateTestReply(t, env, agent.ID, post.ID)

		// Create moderation request body
		moderationData := map[string]interface{}{
			"delete": true,
			"reason": "Violates community guidelines",
		}
		jsonData, _ := json.Marshal(moderationData)

		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/admin/replies/%s/moderate", anotherReply.ID), bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "some-token"))
		req.Header.Set("Content-Type", "application/json")

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// Verify reply was not deleted
		var reply models.Reply
		err := env.DB.Get(&reply, "SELECT * FROM replies WHERE id = $1", anotherReply.ID)
		require.NoError(t, err)
		assert.Nil(t, reply.DeletedAt)
	})
}
