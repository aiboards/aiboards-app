package handlers

import (
	"log"
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
	User  gin.H `json:"user"`
	Token gin.H `json:"token"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	log.Printf("AuthHandler.Register: called for %s", c.Request.URL.Path)
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("AuthHandler.Register: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Register(c, req.Email, req.Password, req.Name, req.BetaCode)
	log.Printf("AuthHandler.Register: user: %+v, tokens: %+v, err: %v", user, tokens, err)
	if err != nil {
		status := http.StatusInternalServerError
		switch err {
		case services.ErrUserAlreadyExists:
			status = http.StatusConflict
		case services.ErrInvalidBetaCode:
			status = http.StatusBadRequest
		}
		log.Printf("AuthHandler.Register: error response status %d: %v", status, err)
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
	log.Printf("AuthHandler.Register: returning user ID %v", user.ID)
	c.JSON(http.StatusOK, TokenResponse{
		User: gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
		Token: gin.H{
			"access_token": tokens.AccessToken,
			"expires_at":   tokens.ExpiresAt,
		},
	})
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	log.Printf("AuthHandler.Login: called for %s", c.Request.URL.Path)
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("AuthHandler.Login: failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, tokens, err := h.authService.Login(c, req.Email, req.Password)
	log.Printf("AuthHandler.Login: user: %+v, tokens: %+v, err: %v", user, tokens, err)
	if err != nil {
		status := http.StatusInternalServerError
		if err == services.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		log.Printf("AuthHandler.Login: error response status %d: %v", status, err)
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
	log.Printf("AuthHandler.Login: returning user ID %v", user.ID)
	c.JSON(http.StatusOK, TokenResponse{
		User: gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
		},
		Token: gin.H{
			"access_token": tokens.AccessToken,
			"expires_at":   tokens.ExpiresAt,
		},
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	log.Printf("AuthHandler.RefreshToken: called for %s", c.Request.URL.Path)
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		log.Printf("AuthHandler.RefreshToken: no refresh token cookie: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token not found"})
		return
	}

	tokens, err := h.authService.RefreshTokens(c, refreshToken)
	log.Printf("AuthHandler.RefreshToken: tokens: %+v, err: %v", tokens, err)
	if err != nil {
		log.Printf("AuthHandler.RefreshToken: invalid refresh token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.SetCookie(
		"refresh_token",
		tokens.RefreshToken,
		int(time.Until(tokens.ExpiresAt.Add(24*time.Hour*7)).Seconds()), // 7 days
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HTTP only
	)

	log.Printf("AuthHandler.RefreshToken: returning new access token")
	c.JSON(http.StatusOK, gin.H{
		"token": gin.H{
			"access_token": tokens.AccessToken,
			"expires_at":   tokens.ExpiresAt,
		},
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
