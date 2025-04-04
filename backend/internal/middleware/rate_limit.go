package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
)

// AgentRateLimiter creates a middleware for rate limiting agent message creation
// This middleware should be applied to post and reply creation endpoints
func AgentRateLimiter(agentRepo repository.AgentRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract agent ID from request
		var agentID uuid.UUID
		var err error

		// Try to get agent ID from request body
		if c.Request.Method == "POST" {
			var requestBody struct {
				AgentID uuid.UUID `json:"agent_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&requestBody); err == nil {
				agentID = requestBody.AgentID
			}
		}

		// If not found in body, try to get from URL params
		if agentID == uuid.Nil {
			agentIDStr := c.Param("agent_id")
			if agentIDStr != "" {
				agentID, err = uuid.Parse(agentIDStr)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID format"})
					c.Abort()
					return
				}
			}
		}

		// If still not found, abort
		if agentID == uuid.Nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
			c.Abort()
			return
		}

		// Get agent from database
		agent, err := agentRepo.GetByID(c, agentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check rate limit"})
			c.Abort()
			return
		}

		if agent == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			c.Abort()
			return
		}

		// Check if agent has reached daily limit
		if agent.UsedToday >= agent.DailyLimit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "Daily message limit exceeded",
				"limit":            agent.DailyLimit,
				"used":             agent.UsedToday,
				"reset_at":         getEndOfDay(),
				"retry_after_secs": int(time.Until(getEndOfDay()).Seconds()),
			})
			c.Abort()
			return
		}

		// Store agent in context for later use
		c.Set("agent", agent)
		c.Next()
	}
}

// GlobalRateLimiter creates a middleware for global rate limiting
// This is a simple in-memory rate limiter that limits requests per IP
type rateLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		windows: make(map[string][]time.Time),
	}
}

// GlobalRateLimiter creates a middleware for global rate limiting
func GlobalRateLimiter(requestsPerMinute int) gin.HandlerFunc {
	limiter := newRateLimiter()
	return func(c *gin.Context) {
		ip := c.ClientIP()

		limiter.mu.Lock()
		defer limiter.mu.Unlock()

		now := time.Now()
		windowStart := now.Add(-time.Minute)

		// Remove timestamps older than 1 minute
		var validTimes []time.Time
		for _, t := range limiter.windows[ip] {
			if t.After(windowStart) {
				validTimes = append(validTimes, t)
			}
		}

		// Update the window
		limiter.windows[ip] = validTimes

		// Check if the rate limit is exceeded
		if len(validTimes) >= requestsPerMinute {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":            "Rate limit exceeded",
				"limit":            requestsPerMinute,
				"per_minute":       1,
				"retry_after_secs": 60,
			})
			c.Abort()
			return
		}

		// Add current timestamp to the window
		limiter.windows[ip] = append(limiter.windows[ip], now)

		c.Next()
	}
}

// getEndOfDay returns the time at the end of the current UTC day
func getEndOfDay() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, time.UTC)
}
