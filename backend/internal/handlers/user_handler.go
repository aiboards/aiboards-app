package handlers

import (
	"log"
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
	Name              string `json:"name" binding:"required"`
	ProfilePictureURL string `json:"profile_picture_url" binding:"omitempty,url"`
}

// UpdatePictureRequest represents the request body for updating a profile picture
type UpdatePictureRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// ChangePasswordRequest represents the request body for changing a password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// GetCurrentUser returns the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	log.Printf("GetCurrentUser: called for %s", c.Request.URL.Path)
	log.Printf("GetCurrentUser: c.Keys at entry: %+v", c.Keys)
	userObj, exists := c.Get("user")
	log.Printf("GetCurrentUser: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	log.Printf("GetCurrentUser: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	// Get full user details from database
	fullUser, err := h.userService.GetUserByID(c, user.ID)
	log.Printf("GetCurrentUser: fullUser: %+v, err: %v", fullUser, err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}

	// Return user info (excluding password hash)
	c.JSON(http.StatusOK, gin.H{
		"id":         fullUser.ID,
		"email":      fullUser.Email,
		"name":       fullUser.Name,
		"isAdmin":    fullUser.IsAdmin,
		"created_at": fullUser.CreatedAt,
		"updated_at": fullUser.UpdatedAt,
	})
}

// UpdateUser updates the current user's information
func (h *UserHandler) UpdateUser(c *gin.Context) {
	log.Printf("UpdateUser: called for %s", c.Request.URL.Path)
	userObj, exists := c.Get("user")
	log.Printf("UpdateUser: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	log.Printf("UpdateUser: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("UpdateUser: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fullUser, err := h.userService.GetUserByID(c, user.ID)
	log.Printf("UpdateUser: fullUser: %+v, err: %v", fullUser, err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user details"})
		return
	}

	fullUser.Name = req.Name
	if req.ProfilePictureURL != "" {
		fullUser.ProfilePictureURL = req.ProfilePictureURL
	}
	if err := h.userService.UpdateUser(c, fullUser); err != nil {
		log.Printf("UpdateUser: failed to update user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	log.Printf("UpdateUser: successfully updated user %v", fullUser.ID)
	c.JSON(http.StatusOK, gin.H{
		"id":         fullUser.ID,
		"email":      fullUser.Email,
		"name":       fullUser.Name,
		"isAdmin":    fullUser.IsAdmin,
		"created_at": fullUser.CreatedAt,
		"updated_at": fullUser.UpdatedAt,
	})
}

// ChangePassword changes the current user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	log.Printf("ChangePassword: called for %s", c.Request.URL.Path)
	userObj, exists := c.Get("user")
	log.Printf("ChangePassword: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	log.Printf("ChangePassword: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ChangePassword: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.userService.ChangePassword(c, user.ID, req.CurrentPassword, req.NewPassword)
	log.Printf("ChangePassword: result err: %v", err)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		log.Printf("ChangePassword: error response status %d: %v", status, err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	log.Printf("ChangePassword: password changed successfully for user %v", user.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DeleteUser deletes the current user's account
func (h *UserHandler) DeleteUser(c *gin.Context) {
	log.Printf("DeleteUser: called for %s", c.Request.URL.Path)
	userObj, exists := c.Get("user")
	log.Printf("DeleteUser: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	user, ok := userObj.(*models.User)
	log.Printf("DeleteUser: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	if err := h.userService.DeleteUser(c, user.ID); err != nil {
		log.Printf("DeleteUser: failed to delete user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	log.Printf("DeleteUser: user %v deleted successfully", user.ID)
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
