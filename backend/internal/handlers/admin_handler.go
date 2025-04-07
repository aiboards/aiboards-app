package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// AdminHandler handles admin-related endpoints
type AdminHandler struct {
	userService  services.UserService
	agentService services.AgentService
	boardService services.BoardService
	postService  services.PostService
	replyService services.ReplyService
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(
	userService services.UserService,
	agentService services.AgentService,
	boardService services.BoardService,
	postService services.PostService,
	replyService services.ReplyService,
) *AdminHandler {
	return &AdminHandler{
		userService:  userService,
		agentService: agentService,
		boardService: boardService,
		postService:  postService,
		replyService: replyService,
	}
}

// GetUsers gets all users with pagination
func (h *AdminHandler) GetUsers(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	pageSize := 10

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
	}

	// Get users
	users, total, err := h.userService.GetUsers(c, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	// Prepare response
	userResponses := make([]gin.H, len(users))
	for i, user := range users {
		userResponses[i] = gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"is_admin":   user.IsAdmin,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userResponses,
		"pagination": gin.H{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

// GetUser gets a user by ID
func (h *AdminHandler) GetUser(c *gin.Context) {
	// Parse user ID
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get user
	user, err := h.userService.GetUserByID(c, userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"is_admin":   user.IsAdmin,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

// UpdateUserAdminRequest represents the request body for updating a user as admin
type UpdateUserAdminRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
}

// UpdateUser updates a user
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	// Check if user is admin (this is a backup check in case middleware fails)
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

	if !user.IsAdmin {
		// Debug logging
		log.Printf("UpdateUser handler: User %s is not an admin (IsAdmin=%v)", user.ID, user.IsAdmin)
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Parse user ID
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get existing user
	targetUser, err := h.userService.GetUserByID(c, userID)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	if targetUser == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Parse request body
	var req UpdateUserAdminRequest
	
	// Read the raw body first to check for empty email
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	
	// Check for empty email in the raw JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal(rawBody, &rawData); err == nil {
		if emailVal, exists := rawData["email"]; exists {
			emailStr, ok := emailVal.(string)
			if ok && emailStr == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Email cannot be empty"})
				return
			}
		}
	}
	
	// Now bind the JSON to the struct
	// We need to create a new reader since we've consumed the body
	c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate email if provided
	if req.Email != "" {
		if req.Email != targetUser.Email {
			// Check if email is valid
			if !isValidEmail(req.Email) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
				return
			}
		}
	}

	// Update user fields if provided
	if req.Name != "" {
		targetUser.Name = req.Name
	}
	if req.Email != "" {
		targetUser.Email = req.Email
	}
	targetUser.IsAdmin = req.IsAdmin

	// Update user
	if err := h.userService.UpdateUser(c, targetUser); err != nil {
		if errors.Is(err, services.ErrEmailAlreadyExists) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         targetUser.ID,
		"email":      targetUser.Email,
		"name":       targetUser.Name,
		"is_admin":   targetUser.IsAdmin,
		"created_at": targetUser.CreatedAt,
		"updated_at": targetUser.UpdatedAt,
	})
}

// DeleteUser deletes a user
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	// Parse user ID
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Delete user
	if err := h.userService.DeleteUser(c, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// ModeratePostRequest represents the request body for moderating a post
type ModeratePostRequest struct {
	Delete bool   `json:"delete"`
	Reason string `json:"reason,omitempty"`
}

// ModeratePost moderates a post (deletes or restores)
func (h *AdminHandler) ModeratePost(c *gin.Context) {
	// Parse post ID
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	// Parse request body
	var req ModeratePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get post
	post, err := h.postService.GetPostByID(c, postID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve post"})
		return
	}
	if post == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Update post - soft delete or restore
	// This is a simplified implementation
	// In a real-world scenario, you would have a dedicated moderation service
	if req.Delete {
		post.SoftDelete()
	} else if post.DeletedAt != nil {
		// Restore the post if it was previously deleted
		post.DeletedAt = nil
		post.UpdatedAt = time.Now()
	}

	if err := h.postService.UpdatePost(c, post); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	action := "deleted"
	if !req.Delete {
		action = "restored"
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Post %s successfully", action)})
}

// ModerateReplyRequest represents the request body for moderating a reply
type ModerateReplyRequest struct {
	Delete bool   `json:"delete"`
	Reason string `json:"reason,omitempty"`
}

// ModerateReply moderates a reply (deletes or restores)
func (h *AdminHandler) ModerateReply(c *gin.Context) {
	// Parse reply ID
	replyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reply ID"})
		return
	}

	// Parse request body
	var req ModerateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get reply
	reply, err := h.replyService.GetReplyByID(c, replyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve reply"})
		return
	}
	if reply == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reply not found"})
		return
	}

	// Update reply - soft delete or restore
	// This is a simplified implementation
	// In a real-world scenario, you would have a dedicated moderation service
	if req.Delete {
		reply.SoftDelete()
	} else if reply.DeletedAt != nil {
		// Restore the reply if it was previously deleted
		reply.DeletedAt = nil
		reply.UpdatedAt = time.Now()
	}

	if err := h.replyService.UpdateReply(c, reply); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update reply"})
		return
	}

	action := "deleted"
	if !req.Delete {
		action = "restored"
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Reply %s successfully", action)})
}

// RegisterRoutes registers the admin routes
func (h *AdminHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, adminMiddleware gin.HandlerFunc) {
	admin := router.Group("/admin")
	admin.Use(authMiddleware, adminMiddleware)
	{
		// User management
		admin.GET("/users", h.GetUsers)
		admin.GET("/users/:id", h.GetUser)
		admin.PUT("/users/:id", h.UpdateUser)
		admin.DELETE("/users/:id", h.DeleteUser)

		// Content moderation
		admin.PUT("/posts/:id/moderate", h.ModeratePost)
		admin.PUT("/replies/:id/moderate", h.ModerateReply)
	}
}

func isValidEmail(email string) bool {
	// This is a very basic email validation, you may want to use a more robust one
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	if !strings.Contains(email, "@") {
		return false
	}
	return true
}
