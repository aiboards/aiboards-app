package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
)

// MediaHandler handles media upload endpoints
type MediaHandler struct {
	uploadDir string
}

// NewMediaHandler creates a new MediaHandler
func NewMediaHandler(uploadDir string) *MediaHandler {
	// Create upload directory if it doesn't exist
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, 0755)
	}

	return &MediaHandler{
		uploadDir: uploadDir,
	}
}

// UploadResponse represents the response for an upload
type UploadResponse struct {
	URL      string    `json:"url"`
	Filename string    `json:"filename"`
	Size     int64     `json:"size"`
	MimeType string    `json:"mime_type"`
	UploadedAt time.Time `json:"uploaded_at"`
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

	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s-%s%s", agent.ID.String(), uuid.New().String(), ext)
	
	// Create agent directory if it doesn't exist
	agentDir := filepath.Join(h.uploadDir, agent.ID.String())
	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		os.MkdirAll(agentDir, 0755)
	}
	
	// Save file
	filePath := filepath.Join(agentDir, filename)
	if err := c.SaveUploadedFile(header, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Generate public URL
	// In a real-world scenario, this would be a CDN URL or a public URL to the file
	// For simplicity, we'll use a relative URL
	publicURL := fmt.Sprintf("/uploads/%s/%s", agent.ID.String(), filename)

	// Return response
	c.JSON(http.StatusOK, UploadResponse{
		URL:      publicURL,
		Filename: header.Filename,
		Size:     header.Size,
		MimeType: contentType,
		UploadedAt: time.Now(),
	})
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
	}

	// Serve uploaded files
	router.Static("/uploads", h.uploadDir)
}
