package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/services"
)

// BoardHandler handles HTTP requests related to boards
type BoardHandler struct {
	boardService services.BoardService
}

// NewBoardHandler creates a new BoardHandler
func NewBoardHandler(boardService services.BoardService) *BoardHandler {
	return &BoardHandler{
		boardService: boardService,
	}
}

// CreateBoard creates a new board
func (h *BoardHandler) CreateBoard(c *gin.Context) {
	// Parse request
	var req struct {
		AgentID     string `json:"agent_id" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse agent ID
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Create board
	board, err := h.boardService.CreateBoard(c.Request.Context(), agentID, req.Title, req.Description, req.IsActive)
	if err != nil {
		if err == services.ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, board)
}

// GetBoard gets a board by ID
func (h *BoardHandler) GetBoard(c *gin.Context) {
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Get board
	board, err := h.boardService.GetBoardByID(c.Request.Context(), boardID)
	if err != nil {
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

// GetBoardByAgent gets a board by agent ID
func (h *BoardHandler) GetBoardByAgent(c *gin.Context) {
	// Parse agent ID
	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get board
	board, err := h.boardService.GetBoardByAgentID(c.Request.Context(), agentID)
	if err != nil {
		if err == services.ErrAgentNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

// UpdateBoard updates a board
func (h *BoardHandler) UpdateBoard(c *gin.Context) {
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Parse request
	var req struct {
		AgentID     string `json:"agent_id" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse agent ID
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get existing board
	board, err := h.boardService.GetBoardByID(c.Request.Context(), boardID)
	if err != nil {
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Update board
	board.AgentID = agentID
	board.Title = req.Title
	board.Description = req.Description
	board.IsActive = req.IsActive

	err = h.boardService.UpdateBoard(c.Request.Context(), board)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

// DeleteBoard deletes a board
func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Delete board
	err = h.boardService.DeleteBoard(c.Request.Context(), boardID)
	if err != nil {
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "board deleted"})
}

// ListBoards lists all boards
func (h *BoardHandler) ListBoards(c *gin.Context) {
	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// Get boards
	boards, totalCount, err := h.boardService.ListBoards(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"boards":      boards,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// SetBoardActive sets the active status of a board
func (h *BoardHandler) SetBoardActive(c *gin.Context) {
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Parse request body directly
	var requestMap map[string]interface{}
	if err := c.ShouldBindJSON(&requestMap); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract isActive value
	isActiveValue, exists := requestMap["is_active"]
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active field is required"})
		return
	}

	// Convert to boolean
	isActive, ok := isActiveValue.(bool)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active must be a boolean"})
		return
	}

	// Set active status
	err = h.boardService.SetBoardActive(c.Request.Context(), boardID, isActive)
	if err != nil {
		if err == services.ErrBoardNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "board not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "board active status updated"})
}

// SearchBoards searches for boards by title or description
func (h *BoardHandler) SearchBoards(c *gin.Context) {
	// Get search query
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
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
	
	// Search boards
	boards, totalCount, err := h.boardService.SearchBoards(c.Request.Context(), query, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"boards":      boards,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
		"query":       query,
	})
}

// RegisterRoutes registers the board routes
func (h *BoardHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	boards := router.Group("/boards")

	// Public endpoints (no auth required)
	boards.GET("", h.ListBoards)
	boards.GET("/search", h.SearchBoards)
	boards.GET("/:id", h.GetBoard)
	boards.GET("/agent/:agent_id", h.GetBoardByAgent)

	// Authenticated endpoints (require login)
	boardsAuth := boards.Group("")
	boardsAuth.Use(authMiddleware)
	{
		boardsAuth.POST("", h.CreateBoard)
		boardsAuth.PUT("/:id", h.UpdateBoard)
		boardsAuth.DELETE("/:id", h.DeleteBoard)
		boardsAuth.PUT("/:id/active", h.SetBoardActive)
	}
}
