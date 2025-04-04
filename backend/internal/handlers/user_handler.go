package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// UserHandler handles user-related endpoints
type UserHandler struct {
	userService services.UserService
	authService services.AuthService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService services.UserService, authService services.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Name string `json:"name" binding:"required"`
}

// ChangePasswordRequest represents the request body for changing a password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// GetCurrentUser returns the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user from context (set by AuthMiddleware)
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	// Get full user details from database
	fullUser, err := h.userService.GetUserByID(c, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}

	// Return user info (excluding password hash)
	c.JSON(http.StatusOK, gin.H{
		"id":         fullUser.ID,
		"email":      fullUser.Email,
		"name":       fullUser.Name,
		"is_admin":   fullUser.IsAdmin,
		"created_at": fullUser.CreatedAt,
		"updated_at": fullUser.UpdatedAt,
	})
}

// UpdateUser updates the current user's information
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Get user from context
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	// Parse request body
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get full user details from database
	fullUser, err := h.userService.GetUserByID(c, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}

	// Update user
	fullUser.Name = req.Name
	if err := h.userService.UpdateUser(c, fullUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	// Return updated user info
	c.JSON(http.StatusOK, gin.H{
		"id":         fullUser.ID,
		"email":      fullUser.Email,
		"name":       fullUser.Name,
		"is_admin":   fullUser.IsAdmin,
		"created_at": fullUser.CreatedAt,
		"updated_at": fullUser.UpdatedAt,
	})
}

// ChangePassword changes the current user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	// Get user from context
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	// Parse request body
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Change password
	err := h.userService.ChangePassword(c, user.ID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DeleteUser deletes the current user's account
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// Get user from context
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	// Delete user
	if err := h.userService.DeleteUser(c, user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// RegisterRoutes registers the user routes
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	users := router.Group("/users")
	users.Use(authMiddleware)
	{
		users.GET("/me", h.GetCurrentUser)
		users.PUT("/me", h.UpdateUser)
		users.POST("/me/change-password", h.ChangePassword)
		users.DELETE("/me", h.DeleteUser)
	}
}
