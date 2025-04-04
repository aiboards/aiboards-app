package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrBetaCodeNotFound = errors.New("beta code not found")
	ErrBetaCodeUsed     = errors.New("beta code has already been used")
	ErrInvalidBetaCode  = errors.New("invalid beta code")
)

// BetaCodeService handles beta code-related business logic
type BetaCodeService interface {
	CreateBetaCode(ctx context.Context) (*models.BetaCode, error)
	CreateMultipleBetaCodes(ctx context.Context, count int) ([]*models.BetaCode, error)
	GetBetaCodeByID(ctx context.Context, id uuid.UUID) (*models.BetaCode, error)
	GetBetaCodeByCode(ctx context.Context, code string) (*models.BetaCode, error)
	ListBetaCodes(ctx context.Context, page, pageSize int) ([]*models.BetaCode, int, error)
	VerifyAndUseBetaCode(ctx context.Context, code string, userID uuid.UUID) error
	DeleteBetaCode(ctx context.Context, id uuid.UUID) error
	CountActiveBetaCodes(ctx context.Context) (int, error)
}

type betaCodeService struct {
	betaCodeRepo repository.BetaCodeRepository
	userRepo     repository.UserRepository
}

// NewBetaCodeService creates a new BetaCodeService
func NewBetaCodeService(
	betaCodeRepo repository.BetaCodeRepository,
	userRepo repository.UserRepository,
) BetaCodeService {
	return &betaCodeService{
		betaCodeRepo: betaCodeRepo,
		userRepo:     userRepo,
	}
}

// generateBetaCode creates a new random beta code
func generateBetaCode() (string, error) {
	bytes := make([]byte, 8) // 8 bytes = 16 hex chars
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Convert to base64 and clean it up
	code := base64.URLEncoding.EncodeToString(bytes)
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, "_", "")

	// Truncate to 12 characters and uppercase
	if len(code) > 12 {
		code = code[:12]
	}
	return strings.ToUpper(code), nil
}

// CreateBetaCode creates a new beta code
func (s *betaCodeService) CreateBetaCode(ctx context.Context) (*models.BetaCode, error) {
	// Generate a unique beta code
	code, err := generateBetaCode()
	if err != nil {
		return nil, err
	}

	// Ensure the code is unique
	existingCode, err := s.betaCodeRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if existingCode != nil {
		// Try again with a new code
		return s.CreateBetaCode(ctx)
	}

	// Create the beta code
	now := time.Now()
	betaCode := &models.BetaCode{
		ID:        uuid.New(),
		Code:      code,
		IsUsed:    false,
		CreatedAt: now,
	}

	// Save the beta code
	err = s.betaCodeRepo.Create(ctx, betaCode)
	if err != nil {
		return nil, err
	}

	return betaCode, nil
}

// CreateMultipleBetaCodes creates multiple beta codes
func (s *betaCodeService) CreateMultipleBetaCodes(ctx context.Context, count int) ([]*models.BetaCode, error) {
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}

	betaCodes := make([]*models.BetaCode, 0, count)
	for i := 0; i < count; i++ {
		betaCode, err := s.CreateBetaCode(ctx)
		if err != nil {
			return betaCodes, err
		}
		betaCodes = append(betaCodes, betaCode)
	}

	return betaCodes, nil
}

// GetBetaCodeByID retrieves a beta code by ID
func (s *betaCodeService) GetBetaCodeByID(ctx context.Context, id uuid.UUID) (*models.BetaCode, error) {
	betaCode, err := s.betaCodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if betaCode == nil {
		return nil, ErrBetaCodeNotFound
	}
	return betaCode, nil
}

// GetBetaCodeByCode retrieves a beta code by code string
func (s *betaCodeService) GetBetaCodeByCode(ctx context.Context, code string) (*models.BetaCode, error) {
	// Normalize the code
	code = strings.ToUpper(strings.TrimSpace(code))

	betaCode, err := s.betaCodeRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if betaCode == nil {
		return nil, ErrBetaCodeNotFound
	}
	return betaCode, nil
}

// ListBetaCodes retrieves a paginated list of beta codes
func (s *betaCodeService) ListBetaCodes(ctx context.Context, page, pageSize int) ([]*models.BetaCode, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get beta codes
	betaCodes, err := s.betaCodeRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get active count as an approximation of total
	activeCount, err := s.betaCodeRepo.CountActive(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Add the number of used codes we retrieved to get a better approximation
	usedCount := 0
	for _, betaCode := range betaCodes {
		if betaCode.IsUsed {
			usedCount++
		}
	}
	totalCount := activeCount + usedCount

	return betaCodes, totalCount, nil
}

// VerifyAndUseBetaCode verifies a beta code and marks it as used
func (s *betaCodeService) VerifyAndUseBetaCode(ctx context.Context, code string, userID uuid.UUID) error {
	// Normalize the code
	code = strings.ToUpper(strings.TrimSpace(code))

	// Check if code exists
	betaCode, err := s.betaCodeRepo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if betaCode == nil {
		return ErrInvalidBetaCode
	}

	// Check if code has already been used
	if betaCode.IsUsed {
		return ErrBetaCodeUsed
	}

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Mark code as used
	return s.betaCodeRepo.MarkAsUsed(ctx, betaCode.ID, userID)
}

// DeleteBetaCode soft-deletes a beta code
func (s *betaCodeService) DeleteBetaCode(ctx context.Context, id uuid.UUID) error {
	// Check if beta code exists
	betaCode, err := s.betaCodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if betaCode == nil {
		return ErrBetaCodeNotFound
	}

	// Delete the beta code
	return s.betaCodeRepo.Delete(ctx, id)
}

// CountActiveBetaCodes counts the number of unused beta codes
func (s *betaCodeService) CountActiveBetaCodes(ctx context.Context) (int, error) {
	return s.betaCodeRepo.CountActive(ctx)
}
