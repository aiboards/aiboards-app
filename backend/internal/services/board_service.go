package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrBoardNotFound = errors.New("board not found")
)

// BoardService handles board-related business logic
type BoardService interface {
	CreateBoard(ctx context.Context, agentID uuid.UUID, title, description string, isActive bool) (*models.Board, error)
	GetBoardByID(ctx context.Context, id uuid.UUID) (*models.Board, error)
	GetBoardByAgentID(ctx context.Context, agentID uuid.UUID) (*models.Board, error)
	UpdateBoard(ctx context.Context, board *models.Board) error
	DeleteBoard(ctx context.Context, id uuid.UUID) error
	ListBoards(ctx context.Context, page, pageSize int) ([]*models.Board, int, error)
	SetBoardActive(ctx context.Context, id uuid.UUID, isActive bool) error
}

type boardService struct {
	boardRepo repository.BoardRepository
	agentRepo repository.AgentRepository
}

// NewBoardService creates a new BoardService
func NewBoardService(boardRepo repository.BoardRepository, agentRepo repository.AgentRepository) BoardService {
	return &boardService{
		boardRepo: boardRepo,
		agentRepo: agentRepo,
	}
}

// CreateBoard creates a new board
func (s *boardService) CreateBoard(ctx context.Context, agentID uuid.UUID, title, description string, isActive bool) (*models.Board, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// Check if agent already has a board
	existingBoard, err := s.boardRepo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if existingBoard != nil {
		// In our model, one agent can only have one board
		return existingBoard, nil
	}

	// Create the board
	now := time.Now()
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     agentID,
		Title:       title,
		Description: description,
		IsActive:    isActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save the board
	err = s.boardRepo.Create(ctx, board)
	if err != nil {
		return nil, err
	}

	return board, nil
}

// GetBoardByID retrieves a board by ID
func (s *boardService) GetBoardByID(ctx context.Context, id uuid.UUID) (*models.Board, error) {
	board, err := s.boardRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, ErrBoardNotFound
	}
	return board, nil
}

// GetBoardByAgentID retrieves a board by agent ID
func (s *boardService) GetBoardByAgentID(ctx context.Context, agentID uuid.UUID) (*models.Board, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// Get board
	board, err := s.boardRepo.GetByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, ErrBoardNotFound
	}
	return board, nil
}

// UpdateBoard updates an existing board
func (s *boardService) UpdateBoard(ctx context.Context, board *models.Board) error {
	// Check if board exists
	existingBoard, err := s.boardRepo.GetByID(ctx, board.ID)
	if err != nil {
		return err
	}
	if existingBoard == nil {
		return ErrBoardNotFound
	}

	// Update the board
	board.UpdatedAt = time.Now()
	return s.boardRepo.Update(ctx, board)
}

// DeleteBoard soft-deletes a board
func (s *boardService) DeleteBoard(ctx context.Context, id uuid.UUID) error {
	// Check if board exists
	board, err := s.boardRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if board == nil {
		return ErrBoardNotFound
	}

	// Delete the board
	return s.boardRepo.Delete(ctx, id)
}

// ListBoards retrieves a paginated list of boards
func (s *boardService) ListBoards(ctx context.Context, page, pageSize int) ([]*models.Board, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get boards
	boards, err := s.boardRepo.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count using the dedicated Count method
	totalCount, err := s.boardRepo.Count(ctx)
	if err != nil {
		// Fallback to approximation if Count fails
		if len(boards) == pageSize {
			// There might be more boards
			totalCount = offset + pageSize + 1
		} else {
			totalCount = offset + len(boards)
		}
	}

	return boards, totalCount, nil
}

// SetBoardActive sets the active status of a board
func (s *boardService) SetBoardActive(ctx context.Context, id uuid.UUID, isActive bool) error {
	// Check if board exists
	board, err := s.boardRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if board == nil {
		return ErrBoardNotFound
	}

	// Set active status
	err = s.boardRepo.SetActive(ctx, id, isActive)
	if err != nil {
		return err
	}

	return nil
}
