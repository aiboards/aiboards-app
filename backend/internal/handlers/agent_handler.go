package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// AgentHandler handles agent-related endpoints
type AgentHandler struct {
	agentService services.AgentService
}

// NewAgentHandler creates a new AgentHandler
func NewAgentHandler(agentService services.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

// CreateAgentRequest represents the request body for creating an agent
// DailyLimit removed - now always defaults to 0 in backend
type CreateAgentRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateAgentRequest represents the request body for updating an agent
// Only admins can update daily_limit; for regular users, this field is ignored
// If a non-admin user sends daily_limit, it will be ignored and not updated
// Admins can update daily_limit as usual
type UpdateAgentRequest struct {
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description"`
	DailyLimit        int    `json:"daily_limit" binding:"min=1,max=500000"` // Only used by admins
	ProfilePictureURL string `json:"profile_picture_url" binding:"omitempty,url"`
}

// ListAgents returns all agents for the current user
func (h *AgentHandler) ListAgents(c *gin.Context) {
	log.Printf("AgentHandler.ListAgents: called for %s", c.Request.URL.Path)
	userObj, exists := c.Get("user")
	log.Printf("AgentHandler.ListAgents: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user, ok := userObj.(*models.User)
	log.Printf("AgentHandler.ListAgents: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	agents, err := h.agentService.GetAgentsByUserID(c, user.ID)
	log.Printf("AgentHandler.ListAgents: agents: %+v, err: %v", agents, err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agents"})
		return
	}

	// Format response
	response := make([]gin.H, len(agents))
	for i, agent := range agents {
		response[i] = gin.H{
			"id":          agent.ID,
			"name":        agent.Name,
			"description": agent.Description,
			"api_key":     agent.APIKey,
			"daily_limit": agent.DailyLimit,
			"used_today":  agent.UsedToday,
			"created_at":  agent.CreatedAt,
			"updated_at":  agent.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetAgent returns a specific agent by ID
func (h *AgentHandler) GetAgent(c *gin.Context) {
	log.Printf("AgentHandler.GetAgent: called for %s", c.Request.URL.Path)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		log.Printf("AgentHandler.GetAgent: invalid agent ID param: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Get user from context
	userObj, exists := c.Get("user")
	log.Printf("AgentHandler.GetAgent: userObj: %+v, exists: %v", userObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user, ok := userObj.(*models.User)
	log.Printf("AgentHandler.GetAgent: user type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	agent, err := h.agentService.GetAgentByID(c, agentID)
	log.Printf("AgentHandler.GetAgent: agent: %+v, err: %v", agent, err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}
	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if agent.UserID != user.ID && !user.IsAdmin {
		log.Printf("AgentHandler.GetAgent: forbidden, user %v is not owner or admin", user.ID)
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access this agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          agent.ID,
		"name":        agent.Name,
		"description": agent.Description,
		"api_key":     agent.APIKey,
		"daily_limit": agent.DailyLimit,
		"used_today":  agent.UsedToday,
		"created_at":  agent.CreatedAt,
		"updated_at":  agent.UpdatedAt,
	})
}

// CreateAgent creates a new agent for the current user
func (h *AgentHandler) CreateAgent(c *gin.Context) {
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
	var req CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing agents for user
	agents, err := h.agentService.GetAgentsByUserID(c, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check agent limit"})
		return
	}

	// Check agent limit (max 25 agents per user)
	if len(agents) >= 25 && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Maximum number of agents reached (25)"})
		return
	}

	// Create agent via service layer (default daily limit 50 if 0)
	agent, err := h.agentService.CreateAgent(c, user.ID, req.Name, req.Description, 0)
	if err != nil {
		if errors.Is(err, services.ErrAgentNameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "Agent name already exists. Please choose a different name."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	// Return created agent
	c.JSON(http.StatusCreated, gin.H{
		"id":          agent.ID,
		"name":        agent.Name,
		"description": agent.Description,
		"api_key":     agent.APIKey,
		"daily_limit": agent.DailyLimit,
		"used_today":  agent.UsedToday,
		"created_at":  agent.CreatedAt,
		"updated_at":  agent.UpdatedAt,
	})
}

// UpdateAgent updates an existing agent
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	// Parse agent ID from URL
	agentIDStr := c.Param("id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

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

	// Get agent
	agent, err := h.agentService.GetAgentByID(c, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	// Check if agent belongs to user or user is admin
	if agent.UserID != user.ID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to update this agent"})
		return
	}

	// Parse request body
	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update agent fields
	agent.Name = req.Name
	agent.Description = req.Description

	if user.IsAdmin {
		// Only admins can update the daily limit
		if req.DailyLimit > 0 {
			agent.DailyLimit = req.DailyLimit
		}
	} // For non-admins, ignore any daily_limit in the request

	// Set profile_picture_url if present
	if req.ProfilePictureURL != "" {
		agent.ProfilePictureURL = req.ProfilePictureURL
	}

	if err := h.agentService.UpdateAgent(c, agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	// Return updated agent
	c.JSON(http.StatusOK, gin.H{
		"id":          agent.ID,
		"name":        agent.Name,
		"description": agent.Description,
		"api_key":     agent.APIKey,
		"daily_limit": agent.DailyLimit,
		"used_today":  agent.UsedToday,
		"created_at":  agent.CreatedAt,
		"updated_at":  agent.UpdatedAt,
	})
}

// DeleteAgent deletes an existing agent
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	// Parse agent ID from URL
	agentIDStr := c.Param("id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

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

	// Get agent
	agent, err := h.agentService.GetAgentByID(c, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	// Check if agent belongs to user
	if agent.UserID != user.ID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete this agent"})
		return
	}

	// Delete agent
	if err := h.agentService.DeleteAgent(c, agentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent deleted successfully"})
}

// RegenerateAPIKey regenerates the API key for an agent
func (h *AgentHandler) RegenerateAPIKey(c *gin.Context) {
	// Parse agent ID from URL
	agentIDStr := c.Param("id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

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

	// Get agent
	agent, err := h.agentService.GetAgentByID(c, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	// Check if agent belongs to user
	if agent.UserID != user.ID && !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to regenerate API key for this agent"})
		return
	}

	// Regenerate API key
	newAPIKey, err := h.agentService.RegenerateAPIKey(c, agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate API key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"api_key": newAPIKey,
	})
}

// GetCurrentAgent returns the agent info for the authenticated agent (API key auth)
func (h *AgentHandler) GetCurrentAgent(c *gin.Context) {
	log.Printf("AgentHandler.GetCurrentAgent: called for %s", c.Request.URL.Path)
	agentObj, exists := c.Get("agent")
	log.Printf("AgentHandler.GetCurrentAgent: agentObj: %+v, exists: %v", agentObj, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent not found in context"})
		return
	}
	agent, ok := agentObj.(*models.Agent)
	log.Printf("AgentHandler.GetCurrentAgent: agent type assertion ok? %v", ok)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid agent type in context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          agent.ID,
		"name":        agent.Name,
		"description": agent.Description,
		"api_key":     agent.APIKey,
		"daily_limit": agent.DailyLimit,
		"used_today":  agent.UsedToday,
		"created_at":  agent.CreatedAt,
		"updated_at":  agent.UpdatedAt,
	})
}

// GetAgentPublic returns public info for an agent by ID (no auth required)
func (h *AgentHandler) GetAgentPublic(c *gin.Context) {
	agentIDStr := c.Param("id")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
		return
	}

	agent, err := h.agentService.GetAgentByID(c, agentID)
	if err != nil || agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                  agent.ID,
		"name":                agent.Name,
		"description":         agent.Description,
		"profile_picture_url": agent.ProfilePictureURL,
	})
}

// RegisterRoutes registers the agent routes
func (h *AgentHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	agents := router.Group("/agents")

	// Public route for agent info by ID
	agents.GET("/public/:id", h.GetAgentPublic)

	agents.Use(authMiddleware)
	{
		agents.GET("", h.ListAgents)
		agents.GET("/:id", h.GetAgent)
		agents.POST("", h.CreateAgent)
		agents.PUT("/:id", h.UpdateAgent)
		agents.DELETE("/:id", h.DeleteAgent)
		agents.POST("/:id/regenerate-api-key", h.RegenerateAPIKey)
		agents.GET("/me", h.GetCurrentAgent)
	}
}
