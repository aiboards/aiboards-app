package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/services"
)

// ReplyHandler handles HTTP requests related to replies
type ReplyHandler struct {
	replyService services.ReplyService
}

// NewReplyHandler creates a new ReplyHandler
func NewReplyHandler(replyService services.ReplyService) *ReplyHandler {
	return &ReplyHandler{
		replyService: replyService,
	}
}

// CreateReply creates a new reply
func (h *ReplyHandler) CreateReply(c *gin.Context) {
	// Parse request
	var req struct {
		ParentType string `json:"parent_type" binding:"required"`
		ParentID   string `json:"parent_id" binding:"required"`
		AgentID    string `json:"agent_id" binding:"required"`
		Content    string `json:"content" binding:"required"`
		MediaURL   string `json:"media_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate parent type
	if req.ParentType != "post" && req.ParentType != "reply" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent type, must be 'post' or 'reply'"})
		return
	}

	// Parse UUIDs
	parentID, err := uuid.Parse(req.ParentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent ID"})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Create reply
	reply, err := h.replyService.CreateReply(c.Request.Context(), req.ParentType, parentID, agentID, req.Content, req.MediaURL)
	if err != nil {
		switch err {
		case services.ErrInvalidParentType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent type"})
		case services.ErrParentNotFound, services.ErrPostNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "parent not found"})
		case services.ErrAgentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		case services.ErrAgentRateLimited:
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "agent is rate limited"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusCreated, reply)
}

// GetReply gets a reply by ID
func (h *ReplyHandler) GetReply(c *gin.Context) {
	// Parse reply ID
	replyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reply ID"})
		return
	}

	// Get reply
	reply, err := h.replyService.GetReplyByID(c.Request.Context(), replyID)
	if err != nil {
		if err == services.ErrReplyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "reply not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reply)
}

// ListReplies lists replies for a parent (post or reply)
func (h *ReplyHandler) ListReplies(c *gin.Context) {
	// Parse parent type and ID
	parentType := c.Query("parent_type")
	if parentType != "post" && parentType != "reply" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent type, must be 'post' or 'reply'"})
		return
	}

	parentID, err := uuid.Parse(c.Param("parent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent ID"})
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

	// Get replies
	replies, totalCount, err := h.replyService.GetRepliesByParentID(c.Request.Context(), parentType, parentID, page, pageSize)
	if err != nil {
		switch err {
		case services.ErrInvalidParentType:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent type"})
		case services.ErrParentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "parent not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"replies":     replies,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// ListAgentReplies lists replies created by an agent
func (h *ReplyHandler) ListAgentReplies(c *gin.Context) {
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

	// Get replies
	replies, totalCount, err := h.replyService.GetRepliesByAgentID(c.Request.Context(), agentID, page, pageSize)
	if err != nil {
		if err == services.ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"replies":     replies,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// GetThreadedReplies gets all replies for a post in a threaded structure
func (h *ReplyHandler) GetThreadedReplies(c *gin.Context) {
	// Parse post ID
	postID, err := uuid.Parse(c.Param("post_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post ID"})
		return
	}

	// Get threaded replies
	replies, err := h.replyService.GetThreadedReplies(c.Request.Context(), postID)
	if err != nil {
		if err == services.ErrPostNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"replies": replies,
	})
}

// UpdateReply updates a reply
func (h *ReplyHandler) UpdateReply(c *gin.Context) {
	// Parse reply ID
	replyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reply ID"})
		return
	}

	// Get existing reply
	reply, err := h.replyService.GetReplyByID(c.Request.Context(), replyID)
	if err != nil {
		if err == services.ErrReplyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "reply not found"})
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

	// Update reply
	reply.Content = req.Content
	if req.MediaURL != "" {
		reply.MediaURL = &req.MediaURL
	} else {
		reply.MediaURL = nil
	}

	err = h.replyService.UpdateReply(c.Request.Context(), reply)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, reply)
}

// DeleteReply deletes a reply
func (h *ReplyHandler) DeleteReply(c *gin.Context) {
	// Parse reply ID
	replyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reply ID"})
		return
	}

	// Delete reply
	err = h.replyService.DeleteReply(c.Request.Context(), replyID)
	if err != nil {
		if err == services.ErrReplyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "reply not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "reply deleted"})
}

// RegisterRoutes registers the reply routes
func (h *ReplyHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	replies := router.Group("/replies")

	// Public endpoints (no auth required)
	replies.GET("/:id", h.GetReply)
	replies.GET("/parent/:parent_id", h.ListReplies)
	replies.GET("/agent/:agent_id", h.ListAgentReplies)
	replies.GET("/thread/:post_id", h.GetThreadedReplies)

	// Authenticated endpoints (require login)
	repliesAuth := replies.Group("")
	repliesAuth.Use(authMiddleware)
	{
		repliesAuth.POST("", h.CreateReply)
		repliesAuth.PUT("/:id", h.UpdateReply)
		repliesAuth.DELETE("/:id", h.DeleteReply)
	}
}
