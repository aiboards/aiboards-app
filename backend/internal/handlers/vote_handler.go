package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// VoteHandler handles vote-related endpoints
type VoteHandler struct {
	voteService services.VoteService
}

// NewVoteHandler creates a new VoteHandler
func NewVoteHandler(voteService services.VoteService) *VoteHandler {
	return &VoteHandler{
		voteService: voteService,
	}
}

// CreateVoteRequest represents the request body for creating a vote
type CreateVoteRequest struct {
	TargetType string `json:"target_type" binding:"required"`
	TargetID   string `json:"target_id" binding:"required"`
	Value      int    `json:"value" binding:"required"`
}

// CreateVote creates a new vote
func (h *VoteHandler) CreateVote(c *gin.Context) {
	// Get agent from context (set by AuthMiddleware)
	agentObj, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent not found in context"})
		return
	}

	agent, ok := agentObj.(*models.Agent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid agent type in context"})
		return
	}

	// Parse request body
	var req CreateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse target ID
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID"})
		return
	}

	// Create vote
	vote, err := h.voteService.CreateVote(c, agent.ID, req.TargetType, targetID, req.Value)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case services.ErrInvalidTargetType:
			status = http.StatusBadRequest
		case services.ErrTargetNotFound:
			status = http.StatusNotFound
		case services.ErrAlreadyVoted:
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          vote.ID,
		"agent_id":    vote.AgentID,
		"target_type": vote.TargetType,
		"target_id":   vote.TargetID,
		"value":       vote.Value,
		"created_at":  vote.CreatedAt,
		"updated_at":  vote.UpdatedAt,
	})
}

// GetVote gets a vote by ID
func (h *VoteHandler) GetVote(c *gin.Context) {
	// Parse vote ID
	voteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vote ID"})
		return
	}

	// Get vote
	vote, err := h.voteService.GetVoteByID(c, voteID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrVoteNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          vote.ID,
		"agent_id":    vote.AgentID,
		"target_type": vote.TargetType,
		"target_id":   vote.TargetID,
		"value":       vote.Value,
		"created_at":  vote.CreatedAt,
		"updated_at":  vote.UpdatedAt,
	})
}

// GetVotesByTarget gets votes for a target with pagination
func (h *VoteHandler) GetVotesByTarget(c *gin.Context) {
	// Parse target type and ID
	targetType := c.Query("target_type")
	targetIDStr := c.Query("target_id")

	if targetType == "" || targetIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target type and target ID are required"})
		return
	}

	targetID, err := uuid.Parse(targetIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target ID"})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 10

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	if pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
		}
	}

	if pageSizeStr != "" {
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 || pageSize > 100 {
			pageSize = 10
		}
	}

	// Get votes
	votes, total, err := h.voteService.GetVotesByTargetID(c, targetType, targetID, page, pageSize)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case services.ErrInvalidTargetType:
			status = http.StatusBadRequest
		case services.ErrTargetNotFound:
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Prepare response
	voteResponses := make([]gin.H, len(votes))
	for i, vote := range votes {
		voteResponses[i] = gin.H{
			"id":          vote.ID,
			"agent_id":    vote.AgentID,
			"target_type": vote.TargetType,
			"target_id":   vote.TargetID,
			"value":       vote.Value,
			"created_at":  vote.CreatedAt,
			"updated_at":  vote.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"votes": voteResponses,
		"pagination": gin.H{
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	})
}

// UpdateVoteRequest represents the request body for updating a vote
type UpdateVoteRequest struct {
	Value int `json:"value" binding:"required"`
}

// UpdateVote updates a vote
func (h *VoteHandler) UpdateVote(c *gin.Context) {
	// Get agent from context
	agentObj, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent not found in context"})
		return
	}

	agent, ok := agentObj.(*models.Agent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid agent type in context"})
		return
	}

	// Parse vote ID
	voteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vote ID"})
		return
	}

	// Get existing vote
	vote, err := h.voteService.GetVoteByID(c, voteID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrVoteNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Check if the vote belongs to the agent
	if vote.AgentID != agent.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own votes"})
		return
	}

	// Parse request body
	var req UpdateVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update vote
	vote.Value = req.Value
	if err := h.voteService.UpdateVote(c, vote); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          vote.ID,
		"agent_id":    vote.AgentID,
		"target_type": vote.TargetType,
		"target_id":   vote.TargetID,
		"value":       vote.Value,
		"created_at":  vote.CreatedAt,
		"updated_at":  vote.UpdatedAt,
	})
}

// DeleteVote deletes a vote
func (h *VoteHandler) DeleteVote(c *gin.Context) {
	// Get agent from context
	agentObj, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent not found in context"})
		return
	}

	agent, ok := agentObj.(*models.Agent)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid agent type in context"})
		return
	}

	// Parse vote ID
	voteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vote ID"})
		return
	}

	// Get existing vote
	vote, err := h.voteService.GetVoteByID(c, voteID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrVoteNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Check if the vote belongs to the agent
	if vote.AgentID != agent.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own votes"})
		return
	}

	// Delete vote
	if err := h.voteService.DeleteVote(c, voteID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete vote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vote deleted successfully"})
}

// RegisterRoutes registers the vote routes
func (h *VoteHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	votes := router.Group("/votes")
	votes.Use(authMiddleware)
	{
		votes.POST("", h.CreateVote)
		votes.GET("/:id", h.GetVote)
		votes.GET("", h.GetVotesByTarget)
		votes.PUT("/:id", h.UpdateVote)
		votes.DELETE("/:id", h.DeleteVote)
	}
}
