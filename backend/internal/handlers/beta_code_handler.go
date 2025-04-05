package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// BetaCodeHandler handles beta code-related endpoints
type BetaCodeHandler struct {
	betaCodeService services.BetaCodeService
}

// NewBetaCodeHandler creates a new BetaCodeHandler
func NewBetaCodeHandler(betaCodeService services.BetaCodeService) *BetaCodeHandler {
	return &BetaCodeHandler{
		betaCodeService: betaCodeService,
	}
}

// CreateBetaCodeRequest represents the request body for creating multiple beta codes
type CreateBetaCodeRequest struct {
	Count int `json:"count" binding:"required,min=1,max=100"`
}

// ListBetaCodes returns all beta codes with pagination
func (h *BetaCodeHandler) ListBetaCodes(c *gin.Context) {
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

	// Only admin users can list beta codes
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to access beta codes"})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	if pageVal, err := strconv.Atoi(pageStr); err == nil && pageVal > 0 {
		page = pageVal
	}

	if pageSizeVal, err := strconv.Atoi(pageSizeStr); err == nil && pageSizeVal > 0 && pageSizeVal <= 100 {
		pageSize = pageSizeVal
	}

	// Get beta codes
	betaCodes, totalCount, err := h.betaCodeService.ListBetaCodes(c, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve beta codes"})
		return
	}

	// Format response
	response := make([]gin.H, len(betaCodes))
	for i, betaCode := range betaCodes {
		response[i] = gin.H{
			"id":         betaCode.ID,
			"code":       betaCode.Code,
			"is_used":    betaCode.IsUsed,
			"used_by_id": betaCode.UsedByID,
			"used_at":    betaCode.UsedAt,
			"created_at": betaCode.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"beta_codes":  response,
		"total_count": totalCount,
		"page":        page,
		"page_size":   pageSize,
	})
}

// CreateBetaCode creates one or more new beta codes
func (h *BetaCodeHandler) CreateBetaCode(c *gin.Context) {
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

	// Only admin users can create beta codes
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to create beta codes"})
		return
	}

	// Parse request body
	var req CreateBetaCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no request body, default to creating a single beta code
		if err.Error() == "EOF" {
			req.Count = 1
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Create beta codes
	var betaCodes []*models.BetaCode
	var err error

	if req.Count == 1 {
		// Create a single beta code
		betaCode, err := h.betaCodeService.CreateBetaCode(c)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create beta code"})
			return
		}
		betaCodes = []*models.BetaCode{betaCode}
	} else {
		// Create multiple beta codes
		betaCodes, err = h.betaCodeService.CreateMultipleBetaCodes(c, req.Count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create beta codes"})
			return
		}
	}

	// Format response
	response := make([]gin.H, len(betaCodes))
	for i, betaCode := range betaCodes {
		response[i] = gin.H{
			"id":         betaCode.ID,
			"code":       betaCode.Code,
			"created_at": betaCode.CreatedAt,
		}
	}

	c.JSON(http.StatusCreated, response)
}

// DeleteBetaCode deletes a beta code
func (h *BetaCodeHandler) DeleteBetaCode(c *gin.Context) {
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

	// Only admin users can delete beta codes
	if !user.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "You do not have permission to delete beta codes"})
		return
	}

	// Parse beta code ID from URL
	betaCodeIDStr := c.Param("id")
	betaCodeID, err := uuid.Parse(betaCodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid beta code ID format"})
		return
	}

	// Delete beta code
	err = h.betaCodeService.DeleteBetaCode(c, betaCodeID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrBetaCodeNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Beta code deleted successfully"})
}

// RegisterRoutes registers the beta code routes
func (h *BetaCodeHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	betaCodes := router.Group("/beta-codes")
	betaCodes.Use(authMiddleware)
	{
		betaCodes.GET("", h.ListBetaCodes)
		betaCodes.POST("", h.CreateBetaCode)
		betaCodes.DELETE("/:id", h.DeleteBetaCode)
	}
}
