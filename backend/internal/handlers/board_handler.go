package handlers

import (
	"log"
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
	log.Printf("CreateBoard: called for %s", c.Request.URL.Path)
	// Parse request
	var req struct {
		AgentID     string `json:"agent_id" binding:"required"`
		Title       string `json:"title" binding:"required"`
		Description string `json:"description" binding:"required"`
		IsActive    bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("CreateBoard: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse agent ID
	agentID, err := uuid.Parse(req.AgentID)
	log.Printf("CreateBoard: agentID: %s, err: %v", req.AgentID, err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Create board
	board, err := h.boardService.CreateBoard(c.Request.Context(), agentID, req.Title, req.Description, req.IsActive)
	log.Printf("CreateBoard: created board: %+v, err: %v", board, err)
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
	log.Printf("GetBoard: called for %s", c.Request.URL.Path)
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	log.Printf("GetBoard: boardID param: %s, err: %v", c.Param("id"), err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Get board
	board, err := h.boardService.GetBoardByID(c.Request.Context(), boardID)
	log.Printf("GetBoard: board: %+v, err: %v", board, err)
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
	log.Printf("GetBoardByAgent: called for %s", c.Request.URL.Path)
	// Parse agent ID
	agentID, err := uuid.Parse(c.Param("agent_id"))
	log.Printf("GetBoardByAgent: agentID param: %s, err: %v", c.Param("agent_id"), err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get board
	board, err := h.boardService.GetBoardByAgentID(c.Request.Context(), agentID)
	log.Printf("GetBoardByAgent: board: %+v, err: %v", board, err)
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
	log.Printf("UpdateBoard: called for %s", c.Request.URL.Path)
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	log.Printf("UpdateBoard: boardID param: %s, err: %v", c.Param("id"), err)
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
		log.Printf("UpdateBoard: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse agent ID
	agentID, err := uuid.Parse(req.AgentID)
	log.Printf("UpdateBoard: agentID: %s, err: %v", req.AgentID, err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Get existing board
	board, err := h.boardService.GetBoardByID(c.Request.Context(), boardID)
	log.Printf("UpdateBoard: existing board: %+v, err: %v", board, err)
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
	log.Printf("UpdateBoard: updated board: %+v, err: %v", board, err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, board)
}

// DeleteBoard deletes a board
func (h *BoardHandler) DeleteBoard(c *gin.Context) {
	log.Printf("DeleteBoard: called for %s", c.Request.URL.Path)
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	log.Printf("DeleteBoard: boardID param: %s, err: %v", c.Param("id"), err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Delete board
	err = h.boardService.DeleteBoard(c.Request.Context(), boardID)
	log.Printf("DeleteBoard: deleted board: %v, err: %v", boardID, err)
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
	log.Printf("ListBoards: called for %s", c.Request.URL.Path)
	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	log.Printf("ListBoards: page param: %s, err: %v", c.DefaultQuery("page", "1"), err)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	log.Printf("ListBoards: page_size param: %s, err: %v", c.DefaultQuery("page_size", "10"), err)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	// Get boards
	boards, totalCount, err := h.boardService.ListBoards(c.Request.Context(), page, pageSize)
	log.Printf("ListBoards: boards: %+v, totalCount: %d, err: %v", boards, totalCount, err)
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
	log.Printf("SetBoardActive: called for %s", c.Request.URL.Path)
	// Parse board ID
	boardID, err := uuid.Parse(c.Param("id"))
	log.Printf("SetBoardActive: boardID param: %s, err: %v", c.Param("id"), err)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid board ID"})
		return
	}

	// Parse request body directly
	var requestMap map[string]interface{}
	if err := c.ShouldBindJSON(&requestMap); err != nil {
		log.Printf("SetBoardActive: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract isActive value
	isActiveValue, exists := requestMap["is_active"]
	log.Printf("SetBoardActive: isActiveValue: %+v, exists: %v", isActiveValue, exists)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active field is required"})
		return
	}

	// Convert to boolean
	isActive, ok := isActiveValue.(bool)
	log.Printf("SetBoardActive: isActive: %v, ok: %v", isActive, ok)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "is_active must be a boolean"})
		return
	}

	// Set active status
	err = h.boardService.SetBoardActive(c.Request.Context(), boardID, isActive)
	log.Printf("SetBoardActive: set active status: %v, err: %v", isActive, err)
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
	log.Printf("SearchBoards: called for %s", c.Request.URL.Path)
	// Get search query
	query := c.Query("q")
	log.Printf("SearchBoards: query param: %s", query)
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}
	
	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	log.Printf("SearchBoards: page param: %s, err: %v", c.DefaultQuery("page", "1"), err)
	if err != nil || page < 1 {
		page = 1
	}
	
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	log.Printf("SearchBoards: page_size param: %s, err: %v", c.DefaultQuery("page_size", "10"), err)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}
	
	// Search boards
	boards, totalCount, err := h.boardService.SearchBoards(c.Request.Context(), query, page, pageSize)
	log.Printf("SearchBoards: boards: %+v, totalCount: %d, err: %v", boards, totalCount, err)
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
