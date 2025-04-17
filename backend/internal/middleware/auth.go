package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug logging
		log.Printf("AuthMiddleware: Processing request to %s %s", c.Request.Method, c.Request.URL.Path)

		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Printf("AuthMiddleware: No Authorization header for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("AuthMiddleware: Invalid Authorization header format for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := parts[1]

		// Validate the token
		token, err := authService.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			log.Printf("AuthMiddleware: Invalid or expired token for %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Get user from token
		user, err := authService.GetUserFromToken(tokenString)
		if err != nil || user == nil {
			log.Printf("AuthMiddleware: Invalid user in token for %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user in token"})
			c.Abort()
			return
		}

		// Set user in context
		log.Printf("AuthMiddleware: Setting user %s (IsAdmin=%v) in context for %s %s", user.ID, user.IsAdmin, c.Request.Method, c.Request.URL.Path)
		c.Set("user", user)
		c.Next()
	}
}

// AdminMiddleware creates a middleware for admin-only routes
func AdminMiddleware(userService services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug logging
		log.Printf("AdminMiddleware: Processing request to %s %s", c.Request.Method, c.Request.URL.Path)

		// Get user from context (set by AuthMiddleware)
		userObj, exists := c.Get("user")
		if !exists {
			log.Printf("AdminMiddleware: User not found in context for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			c.Abort()
			return
		}

		user, ok := userObj.(*models.User)
		if !ok {
			log.Printf("AdminMiddleware: Invalid user type in context for %s %s", c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
			c.Abort()
			return
		}

		// Check if user is admin
		if !user.IsAdmin {
			// Debug logging
			log.Printf("AdminMiddleware: User %s is not an admin (IsAdmin=%v) for %s %s", user.ID, user.IsAdmin, c.Request.Method, c.Request.URL.Path)
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		log.Printf("AdminMiddleware: User %s is admin, proceeding with %s %s", user.ID, c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}

// APIKeyMiddleware creates a middleware for API key authentication
func APIKeyMiddleware(agentService services.AgentService) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.Next() // No API key, let other auth try
			return
		}
		agent, err := agentService.GetAgentByAPIKey(c, apiKey)
		if err == nil && agent != nil {
			c.Set("agent", agent)
			c.Next()
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing API key"})
		c.Abort()
	}
}

// CompositeAuthMiddleware chains API key and JWT auth middlewares.
// If either sets an identity in context, the request proceeds.
func CompositeAuthMiddleware(agentService services.AgentService, authService services.AuthService) gin.HandlerFunc {
	apiKeyMW := APIKeyMiddleware(agentService)
	jwtMW := AuthMiddleware(authService)
	return func(c *gin.Context) {
		apiKeyMW(c)
		if c.IsAborted() || c.Keys["agent"] != nil {
			return
		}
		jwtMW(c)
	}
}
