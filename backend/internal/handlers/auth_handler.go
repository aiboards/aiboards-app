package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/garrettallen/aiboards/backend/internal/services"
)

// AuthHandler handles authentication-related endpoints
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
	BetaCode string `json:"beta_code" binding:"required"`
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TokenResponse represents the response for authentication endpoints
type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        gin.H     `json:"user"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Register(c, req.Email, req.Password, req.Name, req.BetaCode)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case services.ErrUserAlreadyExists:
			status = http.StatusConflict
		case services.ErrInvalidBetaCode:
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Set refresh token as HTTP-only cookie
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(time.Until(tokens.ExpiresAt.Add(24*time.Hour*7)).Seconds()), // 7 days
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HTTP only
	)

	// Return access token and user info
	c.JSON(http.StatusCreated, TokenResponse{
		AccessToken: tokens.AccessToken,
		ExpiresAt:   tokens.ExpiresAt,
		User: gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Login(c, req.Email, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Set refresh token as HTTP-only cookie
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(time.Until(tokens.ExpiresAt.Add(24*time.Hour*7)).Seconds()), // 7 days
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HTTP only
	)

	// Return access token and user info
	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: tokens.AccessToken,
		ExpiresAt:   tokens.ExpiresAt,
		User: gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token not found"})
		return
	}

	// Refresh tokens
	tokens, err := h.authService.RefreshTokens(c, refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Set new refresh token as HTTP-only cookie
	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(time.Until(tokens.ExpiresAt.Add(24*time.Hour*7)).Seconds()), // 7 days
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HTTP only
	)

	// Return new access token
	c.JSON(http.StatusOK, gin.H{
		"access_token": tokens.AccessToken,
		"expires_at":   tokens.ExpiresAt,
	})
}

// RegisterRoutes registers the auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/signup", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
	}
}
