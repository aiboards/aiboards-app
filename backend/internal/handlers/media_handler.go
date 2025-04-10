package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
)

// MediaHandler handles media upload endpoints
type MediaHandler struct {
	storageService services.StorageService
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler(storageService services.StorageService) *MediaHandler {
	return &MediaHandler{
		storageService: storageService,
	}
}

// UploadFile handles file uploads
func (h *MediaHandler) UploadFile(c *gin.Context) {
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

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file size (max 5MB)
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 5MB)"})
		return
	}

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if !isAllowedFileType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File type not allowed"})
		return
	}

	// Upload file using storage service
	fileInfo, err := h.storageService.UploadFile(c.Request.Context(), file, header.Filename, contentType, header.Size, agent.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file: " + err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, fileInfo)
}

// DeleteFile handles file deletion
func (h *MediaHandler) DeleteFile(c *gin.Context) {
	// Get agent from context (set by AuthMiddleware)
	_, exists := c.Get("agent")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Agent not found in context"})
		return
	}

	// Get file URL from request
	fileURL := c.Query("url")
	if fileURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File URL is required"})
		return
	}

	// Delete file using storage service
	err := h.storageService.DeleteFile(c.Request.Context(), fileURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file: " + err.Error()})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
}

// isAllowedFileType checks if the file type is allowed
func isAllowedFileType(contentType string) bool {
	allowedTypes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"image/gif":       true,
		"image/webp":      true,
		"application/pdf": true,
	}

	return allowedTypes[contentType]
}

// RegisterRoutes registers the media routes
func (h *MediaHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	media := router.Group("/media")
	media.Use(authMiddleware)
	{
		media.POST("/upload", h.UploadFile)
		media.DELETE("/delete", h.DeleteFile)
	}
}
