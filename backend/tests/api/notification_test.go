package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestNotificationAPIEnv extends the TestEnv with notification-specific components
type TestNotificationAPIEnv struct {
	*utils.TestEnv
	NotificationRepository repository.NotificationRepository
	NotificationService    services.NotificationService
}

// NewTestNotificationAPIEnv creates a new test environment with notification components
func NewTestNotificationAPIEnv(t *testing.T) *TestNotificationAPIEnv {
	baseEnv := utils.NewTestEnv(t)

	// Create repositories
	notificationRepo := repository.NewNotificationRepository(baseEnv.DB)

	// Create notification service
	notificationService := services.NewNotificationService(
		notificationRepo,
		baseEnv.UserRepository,
		baseEnv.AgentRepository,
	)

	return &TestNotificationAPIEnv{
		TestEnv:                baseEnv,
		NotificationRepository: notificationRepo,
		NotificationService:    notificationService,
	}
}

// GenerateTokensForAgent generates JWT tokens for an agent
func (env *TestNotificationAPIEnv) GenerateTokensForAgent(agentID uuid.UUID) (*services.TokenPair, error) {
	// Get the agent
	agent, err := env.AgentRepository.GetByID(env.Ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Get the user
	user, err := env.UserRepository.GetByID(env.Ctx, agent.UserID)
	if err != nil {
		return nil, err
	}

	// Login with the user to get tokens
	_, tokens, err := env.AuthService.Login(env.Ctx, user.Email, "password123")
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func setupNotificationTestRouter(t *testing.T) (*gin.Engine, *TestNotificationAPIEnv) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test environment
	env := NewTestNotificationAPIEnv(t)

	// Create router with logging
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(env.AuthService)

	// Create a custom middleware to convert user to agent in context
	userToAgentMiddleware := func(c *gin.Context) {
		// Get user from context (set by AuthMiddleware)
		userObj, exists := c.Get("user")
		if !exists {
			log.Printf("User not found in context")
			c.Next()
			return
		}

		user, ok := userObj.(*models.User)
		if !ok {
			log.Printf("Invalid user type in context")
			c.Next()
			return
		}

		log.Printf("Found user %s in context, looking for agent", user.ID)

		// Get the agent directly from the test environment based on the user ID
		ctx := context.Background()
		
		// Try to get the agent from the repository
		agents, err := env.AgentRepository.GetByUserID(ctx, user.ID)
		if err != nil {
			log.Printf("Error getting agent for user %s: %v", user.ID, err)
			c.Next()
			return
		}
		
		if len(agents) == 0 {
			log.Printf("No agents found for user %s", user.ID)
			c.Next()
			return
		}
		
		agent := agents[0]
		log.Printf("Setting agent %s in context for user %s", agent.ID, user.ID)
		c.Set("agent", agent)
		c.Next()
	}

	// Debug logging for the auth middleware
	log.Printf("Created auth middleware for notification tests")

	// Create notification handler
	notificationHandler := handlers.NewNotificationHandler(env.NotificationService)

	// Setup API group
	api := router.Group("/api/v1")

	// Register notification routes with both middlewares
	notifications := api.Group("/notifications")
	notifications.Use(authMiddleware, userToAgentMiddleware)
	{
		notifications.GET("", notificationHandler.GetNotifications)
		notifications.GET("/unread", notificationHandler.GetUnreadCount)
		notifications.GET("/:id", notificationHandler.GetNotification)
		notifications.PUT("/:id/read", notificationHandler.MarkAsRead)
		notifications.PUT("/read-all", notificationHandler.MarkAllAsRead)
		notifications.DELETE("/:id", notificationHandler.DeleteNotification)
	}

	return router, env
}

// Helper function to create a test notification
func createTestNotification(t *testing.T, env *TestNotificationAPIEnv, agentID uuid.UUID) *models.Notification {
	notification := &models.Notification{
		ID:         uuid.New(),
		AgentID:    agentID,
		Type:       string(services.NotificationTypeSystem),
		Content:    "Test notification",
		TargetType: "post",
		TargetID:   uuid.New(),
		IsRead:     false,
		CreatedAt:  time.Now(),
	}

	err := env.NotificationRepository.Create(env.Ctx, notification)
	require.NoError(t, err)

	return notification
}

func TestGetNotificationEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create a notification for the agent
	notification := createTestNotification(t, env, agent.ID)

	// Create another user and agent (for testing forbidden access)
	_, otherUserID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	otherAgent := env.CreateTestAgent(otherUserID)

	// Get tokens for both users
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	otherTokenPair, err := env.GenerateTokensForAgent(otherAgent.ID)
	require.NoError(t, err)

	t.Run("User can get their own notification", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s", notification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

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

		// Verify notification data
		assert.Equal(t, notification.ID.String(), response["id"])
		assert.Equal(t, notification.AgentID.String(), response["agent_id"])
		assert.Equal(t, notification.Type, response["type"])
		assert.Equal(t, notification.Content, response["content"])
		assert.Equal(t, notification.TargetType, response["target_type"])
		assert.Equal(t, notification.TargetID.String(), response["target_id"])
		assert.Equal(t, notification.IsRead, response["is_read"])
	})

	t.Run("User cannot get another user's notification", func(t *testing.T) {
		// Create request with other user's token
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s", notification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", otherTokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Cannot get non-existent notification", func(t *testing.T) {
		// Create request with non-existent notification ID
		nonExistentID := uuid.New()
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s", nonExistentID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be not found
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("Unauthenticated user cannot get notification", func(t *testing.T) {
		// Create request without token
		req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/notifications/%s", notification.ID), nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestGetNotificationsEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create multiple notifications for the agent
	for i := 0; i < 15; i++ {
		createTestNotification(t, env, agent.ID)
	}

	// Get token for the user
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	t.Run("User can get their notifications with pagination", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/api/v1/notifications?page=1&page_size=10", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

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

		// Verify pagination data
		notifications, ok := response["notifications"].([]interface{})
		require.True(t, ok)
		assert.Len(t, notifications, 10)
		assert.Equal(t, float64(15), response["total"])
		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(10), response["page_size"])
	})

	t.Run("Unauthenticated user cannot get notifications", func(t *testing.T) {
		// Create request without token
		req := httptest.NewRequest("GET", "/api/v1/notifications", nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestMarkAsReadEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create a notification for the agent
	notification := createTestNotification(t, env, agent.ID)

	// Create another user and agent (for testing forbidden access)
	_, otherUserID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	otherAgent := env.CreateTestAgent(otherUserID)

	// Get tokens for both users
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	otherTokenPair, err := env.GenerateTokensForAgent(otherAgent.ID)
	require.NoError(t, err)

	t.Run("User can mark their own notification as read", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/notifications/%s/read", notification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify notification is marked as read
		updatedNotification, err := env.NotificationRepository.GetByID(env.Ctx, notification.ID)
		require.NoError(t, err)
		assert.True(t, updatedNotification.IsRead)
		assert.NotNil(t, updatedNotification.ReadAt)
	})

	t.Run("User cannot mark another user's notification as read", func(t *testing.T) {
		// Create a new notification for the first agent
		anotherNotification := createTestNotification(t, env, agent.ID)

		// Create request with other user's token
		req := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/notifications/%s/read", anotherNotification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", otherTokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)

		// Verify notification is still unread
		updatedNotification, err := env.NotificationRepository.GetByID(env.Ctx, anotherNotification.ID)
		require.NoError(t, err)
		assert.False(t, updatedNotification.IsRead)
	})
}

func TestMarkAllAsReadEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create multiple notifications for the agent
	for i := 0; i < 5; i++ {
		createTestNotification(t, env, agent.ID)
	}

	// Get token for the user
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	t.Run("User can mark all their notifications as read", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("PUT", "/api/v1/notifications/read-all", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify all notifications are marked as read
		unreadCount, err := env.NotificationService.CountUnread(env.Ctx, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, 0, unreadCount)
	})

	t.Run("Unauthenticated user cannot mark all notifications as read", func(t *testing.T) {
		// Create request without token
		req := httptest.NewRequest("PUT", "/api/v1/notifications/read-all", nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestDeleteNotificationEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create a notification for the agent
	notification := createTestNotification(t, env, agent.ID)

	// Create another user and agent (for testing forbidden access)
	_, otherUserID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	otherAgent := env.CreateTestAgent(otherUserID)

	// Get tokens for both users
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	otherTokenPair, err := env.GenerateTokensForAgent(otherAgent.ID)
	require.NoError(t, err)

	t.Run("User can delete their own notification", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/notifications/%s", notification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify notification is deleted (or not found)
		deletedNotification, err := env.NotificationRepository.GetByID(env.Ctx, notification.ID)
		assert.Error(t, err)
		assert.Nil(t, deletedNotification)
	})

	t.Run("User cannot delete another user's notification", func(t *testing.T) {
		// Create a new notification for the first agent
		anotherNotification := createTestNotification(t, env, agent.ID)

		// Create request with other user's token
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/notifications/%s", anotherNotification.ID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", otherTokenPair.AccessToken))

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be forbidden
		assert.Equal(t, http.StatusForbidden, w.Code)

		// Verify notification still exists
		existingNotification, err := env.NotificationRepository.GetByID(env.Ctx, anotherNotification.ID)
		require.NoError(t, err)
		assert.NotNil(t, existingNotification)
	})
}

func TestGetUnreadCountEndpoint(t *testing.T) {
	router, env := setupNotificationTestRouter(t)
	defer env.Cleanup()

	// Create a user and agent
	_, userID := utils.CreateRegularUserAndGetToken(t, env.TestEnv)
	agent := env.CreateTestAgent(userID)

	// Create multiple notifications for the agent
	for i := 0; i < 5; i++ {
		createTestNotification(t, env, agent.ID)
	}

	// Get token for the user
	tokenPair, err := env.GenerateTokensForAgent(agent.ID)
	require.NoError(t, err)

	t.Run("User can get their unread notification count", func(t *testing.T) {
		// Create request
		req := httptest.NewRequest("GET", "/api/v1/notifications/unread", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenPair.AccessToken))

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

		// Verify count
		assert.Equal(t, float64(5), response["count"])
	})

	t.Run("Unauthenticated user cannot get unread count", func(t *testing.T) {
		// Create request without token
		req := httptest.NewRequest("GET", "/api/v1/notifications/unread", nil)

		// Create response recorder
		w := httptest.NewRecorder()

		// Perform request
		router.ServeHTTP(w, req)

		// Check response - should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
