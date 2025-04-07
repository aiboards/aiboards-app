package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"fmt"
)

// NotificationHandler handles notification-related endpoints
type NotificationHandler struct {
	notificationService services.NotificationService
}

// NewNotificationHandler creates a new NotificationHandler
func NewNotificationHandler(notificationService services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// GetNotification gets a notification by ID
func (h *NotificationHandler) GetNotification(c *gin.Context) {
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

	// Parse notification ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	// Get notification
	notification, err := h.notificationService.GetNotificationByID(c, notificationID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrNotificationNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		c.Error(fmt.Errorf("failed to get notification %s: %w", notificationID, err)) // Log the detailed error
		return
	}

	// Check if the notification belongs to the agent
	if notification.AgentID != agent.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          notification.ID,
		"agent_id":    notification.AgentID,
		"type":        notification.Type,
		"content":     notification.Content,
		"target_type": notification.TargetType,
		"target_id":   notification.TargetID,
		"is_read":     notification.IsRead,
		"created_at":  notification.CreatedAt,
		"read_at":     notification.ReadAt,
	})
}

// GetNotifications gets notifications for the current agent with pagination
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
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

	// Parse pagination parameters
	page := 1
	pageSize := 10

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	var err error
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

	// Get notifications
	notifications, total, err := h.notificationService.GetNotificationsByAgentID(c, agent.ID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve notifications"})
		c.Error(err) // Log the error
		return
	}

	// Prepare response
	notificationResponses := make([]gin.H, len(notifications))
	for i, notification := range notifications {
		notificationResponses[i] = gin.H{
			"id":          notification.ID,
			"agent_id":    notification.AgentID,
			"type":        notification.Type,
			"content":     notification.Content,
			"target_type": notification.TargetType,
			"target_id":   notification.TargetID,
			"is_read":     notification.IsRead,
			"created_at":  notification.CreatedAt,
			"read_at":     notification.ReadAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"notifications": notificationResponses,
		"total":         total,
		"page":          page,
		"page_size":     pageSize,
		"total_pages":   (total + pageSize - 1) / pageSize,
	})
}

// MarkAsRead marks a notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
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

	// Parse notification ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	// Get notification to check ownership
	notification, err := h.notificationService.GetNotificationByID(c, notificationID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrNotificationNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		c.Error(err) // Log the error
		return
	}

	// Check if the notification belongs to the agent
	if notification.AgentID != agent.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only mark your own notifications as read"})
		return
	}

	// Mark as read
	if err := h.notificationService.MarkAsRead(c, notificationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark notification as read"})
		c.Error(err) // Log the error
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}

// MarkAllAsRead marks all notifications for the current agent as read
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
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

	// Mark all as read
	if err := h.notificationService.MarkAllAsRead(c, agent.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all notifications as read"})
		c.Error(err) // Log the error
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

// DeleteNotification deletes a notification
func (h *NotificationHandler) DeleteNotification(c *gin.Context) {
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

	// Parse notification ID
	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	// Get notification to check ownership
	notification, err := h.notificationService.GetNotificationByID(c, notificationID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrNotificationNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		c.Error(err) // Log the error
		return
	}

	// Check if the notification belongs to the agent
	if notification.AgentID != agent.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own notifications"})
		return
	}

	// Delete notification
	if err := h.notificationService.DeleteNotification(c, notificationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete notification"})
		c.Error(err) // Log the error
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification deleted successfully"})
}

// GetUnreadCount gets the number of unread notifications for the current agent
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
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

	// Get unread count
	count, err := h.notificationService.CountUnread(c, agent.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count unread notifications"})
		c.Error(err) // Log the error
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// RegisterRoutes registers the notification routes
func (h *NotificationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	notifications := router.Group("/notifications")
	notifications.Use(authMiddleware)
	{
		notifications.GET("", h.GetNotifications)
		notifications.GET("/unread", h.GetUnreadCount)
		notifications.GET("/:id", h.GetNotification)
		notifications.PUT("/:id/read", h.MarkAsRead)
		notifications.PUT("/read-all", h.MarkAllAsRead)
		notifications.DELETE("/:id", h.DeleteNotification)
	}
}
