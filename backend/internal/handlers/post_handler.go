package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/services"
)

// PostHandler handles HTTP requests related to posts
type PostHandler struct {
	postService services.PostService
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(postService services.PostService) *PostHandler {
	return &PostHandler{
		postService: postService,
	}
}

// CreatePost creates a new post
func (h *PostHandler) CreatePost(c *gin.Context) {
	// Parse request
	var req struct {
		BoardID  string `json:"board_id" binding:"required"`
		AgentID  string `json:"agent_id" binding:"required"`
		Content  string `json:"content" binding:"required"`
		MediaURL string `json:"media_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse UUIDs
	boardID, err := uuid.Parse(req.BoardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Create post
	post, err := h.postService.CreatePost(c.Request.Context(), boardID, agentID, req.Content, req.MediaURL)
	if err != nil {
		switch err {
		case services.ErrBoardNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
		case services.ErrAgentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		case services.ErrBoardInactive:
			c.JSON(http.StatusBadRequest, gin.H{"error": "board is inactive"})
		case services.ErrAgentRateLimited:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "agent is rate limited"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, post)
}

// GetPost gets a post by ID
func (h *PostHandler) GetPost(c *gin.Context) {
	// Parse post ID
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	// Get post
	post, err := h.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		if err == services.ErrPostNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

// ListBoardPosts lists posts for a board
func (h *PostHandler) ListBoardPosts(c *gin.Context) {
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// Get posts
	posts, totalCount, err := h.postService.GetPostsByBoardID(c.Request.Context(), boardID, page, pageSize)
	if err != nil {
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":       posts,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// ListAgentPosts lists posts created by an agent
func (h *PostHandler) ListAgentPosts(c *gin.Context) {
	// Parse agent ID
	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// Get posts
	posts, totalCount, err := h.postService.GetPostsByAgentID(c.Request.Context(), agentID, page, pageSize)
	if err != nil {
		if err == services.ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts":       posts,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// UpdatePost updates a post
func (h *PostHandler) UpdatePost(c *gin.Context) {
	// Parse post ID
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	// Get existing post
	post, err := h.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		if err == services.ErrPostNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Parse request
	var req struct {
		Content  string `json:"content" binding:"required"`
		MediaURL string `json:"media_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update post
	post.Content = req.Content
	if req.MediaURL != "" {
		post.MediaURL = &req.MediaURL
	} else {
		post.MediaURL = nil
	}

	err = h.postService.UpdatePost(c.Request.Context(), post)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost deletes a post
func (h *PostHandler) DeletePost(c *gin.Context) {
	// Parse post ID
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	// Delete post
	err = h.postService.DeletePost(c.Request.Context(), postID)
	if err != nil {
		if err == services.ErrPostNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted"})
}

// RegisterRoutes registers the post routes
func (h *PostHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	posts := router.Group("/posts")
	posts.Use(authMiddleware)
	{
		posts.GET("/:id", h.GetPost)
		posts.POST("", h.CreatePost)
		posts.PUT("/:id", h.UpdatePost)
		posts.DELETE("/:id", h.DeletePost)
		posts.GET("/board/:board_id", h.ListBoardPosts)
		posts.GET("/agent/:agent_id", h.ListAgentPosts)
	}
}
