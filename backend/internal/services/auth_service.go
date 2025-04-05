package services

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrWeakPassword       = errors.New("password is too weak")
	ErrInvalidBetaCode    = errors.New("invalid or used beta code")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
)

// Minimum password length
const MinPasswordLength = 8

// Email validation regex
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"-"` // Not returned in JSON
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthService handles authentication-related business logic
type AuthService interface {
	Register(ctx context.Context, email, password, name, betaCode string) (*models.User, *TokenPair, error)
	Login(ctx context.Context, email, password string) (*models.User, *TokenPair, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetUserFromToken(tokenString string) (*models.User, error)
}

type authService struct {
	userRepo     repository.UserRepository
	betaCodeRepo repository.BetaCodeRepository
	jwtSecret    []byte
	accessExp    time.Duration
	refreshExp   time.Duration
}

// NewAuthService creates a new AuthService
func NewAuthService(
	userRepo repository.UserRepository,
	betaCodeRepo repository.BetaCodeRepository,
	jwtSecret string,
	accessExp time.Duration,
	refreshExp time.Duration,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		betaCodeRepo: betaCodeRepo,
		jwtSecret:    []byte(jwtSecret),
		accessExp:    accessExp,
		refreshExp:   refreshExp,
	}
}

// validateEmail checks if the email format is valid
func validateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// validatePassword checks if the password meets minimum requirements
func validatePassword(password string) bool {
	return len(password) >= MinPasswordLength
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, email, password, name, betaCode string) (*models.User, *TokenPair, error) {
	// Validate email format
	if !validateEmail(email) {
		return nil, nil, ErrInvalidEmail
	}

	// Validate password strength
	if !validatePassword(password) {
		return nil, nil, ErrWeakPassword
	}

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if existingUser != nil {
		return nil, nil, ErrUserAlreadyExists
	}

	// Validate beta code
	code, err := s.betaCodeRepo.GetByCode(ctx, betaCode)
	if err != nil {
		return nil, nil, err
	}
	if code == nil || code.IsUsed {
		return nil, nil, ErrInvalidBetaCode
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}

	// Create user
	now := time.Now()
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
		IsAdmin:      false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Execute in transaction
	err = s.userRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Create user
		if err := s.userRepo.Create(ctx, user); err != nil {
			return err
		}

		// Mark beta code as used
		code.IsUsed = true
		code.UsedByID = &user.ID
		code.UsedAt = &now
		return s.betaCodeRepo.Update(ctx, code)
	})

	if err != nil {
		return nil, nil, err
	}

	// Generate tokens
	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// Login authenticates a user and returns tokens
func (s *authService) Login(ctx context.Context, email, password string) (*models.User, *TokenPair, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Generate tokens
	tokens, err := s.generateTokens(user.ID)
	if err != nil {
		return nil, nil, err
	}

	return user, tokens, nil
}

// RefreshTokens generates new tokens using a refresh token
func (s *authService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Parse and validate refresh token
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Check token type
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	// Extract user ID
	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidToken
	}

	// Generate new tokens
	return s.generateTokens(userID)
}

// ValidateToken validates a JWT token
func (s *authService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
}

// GetUserFromToken extracts user information from a token
func (s *authService) GetUserFromToken(tokenString string) (*models.User, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	userIDStr, ok := claims["sub"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// Fetch the user from the database
	ctx := context.Background()
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// generateTokens creates a new access and refresh token pair
func (s *authService) generateTokens(userID uuid.UUID) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(s.accessExp)
	refreshExpiry := now.Add(s.refreshExp)

	// Create access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID.String(),
		"exp":  accessExpiry.Unix(),
		"iat":  now.Unix(),
		"type": "access",
	})

	accessTokenString, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	// Create refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  userID.String(),
		"exp":  refreshExpiry.Unix(),
		"iat":  now.Unix(),
		"type": "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    accessExpiry,
	}, nil
}
