package services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
)

var (
	ErrPostNotFound  = errors.New("post not found")
	ErrBoardInactive = errors.New("board is inactive")
)

// PostService handles post-related business logic
type PostService interface {
	CreatePost(ctx context.Context, boardID, agentID uuid.UUID, content, mediaURL string) (*models.Post, error)
	GetPostByID(ctx context.Context, id uuid.UUID) (*models.Post, error)
	GetPostsByBoardID(ctx context.Context, boardID uuid.UUID, page, pageSize int) ([]*models.Post, int, error)
	GetPostsByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Post, int, error)
	UpdatePost(ctx context.Context, post *models.Post) error
	DeletePost(ctx context.Context, id uuid.UUID) error
}

type postService struct {
	postRepo  repository.PostRepository
	boardRepo repository.BoardRepository
	agentRepo repository.AgentRepository
	agentSvc  AgentService
}

// NewPostService creates a new PostService
func NewPostService(
	postRepo repository.PostRepository,
	boardRepo repository.BoardRepository,
	agentRepo repository.AgentRepository,
	agentSvc AgentService,
) PostService {
	return &postService{
		postRepo:  postRepo,
		boardRepo: boardRepo,
		agentRepo: agentRepo,
		agentSvc:  agentSvc,
	}
}

// CreatePost creates a new post
func (s *postService) CreatePost(ctx context.Context, boardID, agentID uuid.UUID, content, mediaURL string) (*models.Post, error) {
	// Check if board exists and is active
	board, err := s.boardRepo.GetByID(ctx, boardID)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, ErrBoardNotFound
	}
	if !board.IsActive {
		return nil, ErrBoardInactive
	}

	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, ErrAgentNotFound
	}

	// Check rate limit
	isLimited, err := s.agentSvc.CheckRateLimit(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if isLimited {
		return nil, ErrAgentRateLimited
	}

	// Create the post
	now := time.Now()
	post := &models.Post{
		ID:      uuid.New(),
		BoardID: boardID,
		AgentID: agentID,
		Content: content,
		MediaURL: func() *string {
			if mediaURL == "" {
				return nil
			} else {
				return &mediaURL
			}
		}(),
		VoteCount:  0,
		ReplyCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Execute operations in a transaction
	err = s.postRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Save the post
		if err := s.postRepo.Create(ctx, post); err != nil {
			return err
		}

		// Increment agent usage
		if err := s.agentRepo.IncrementUsage(ctx, agentID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return post, nil
}

// GetPostByID retrieves a post by ID
func (s *postService) GetPostByID(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}
	return post, nil
}

// GetPostsByBoardID retrieves posts for a board with pagination
func (s *postService) GetPostsByBoardID(ctx context.Context, boardID uuid.UUID, page, pageSize int) ([]*models.Post, int, error) {
	// Check if board exists
	board, err := s.boardRepo.GetByID(ctx, boardID)
	if err != nil {
		return nil, 0, err
	}
	if board == nil {
		return nil, 0, ErrBoardNotFound
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get posts
	posts, err := s.postRepo.GetByBoardID(ctx, boardID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.postRepo.CountByBoardID(ctx, boardID)
	if err != nil {
		return nil, 0, err
	}

	return posts, count, nil
}

// GetPostsByAgentID retrieves posts created by an agent with pagination
func (s *postService) GetPostsByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Post, int, error) {
	// Check if agent exists
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, 0, err
	}
	if agent == nil {
		return nil, 0, ErrAgentNotFound
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get posts
	posts, err := s.postRepo.GetByAgentID(ctx, agentID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.postRepo.CountByAgentID(ctx, agentID)
	if err != nil {
		return nil, 0, err
	}

	return posts, count, nil
}

// UpdatePost updates an existing post
func (s *postService) UpdatePost(ctx context.Context, post *models.Post) error {
	// Check if post exists
	existingPost, err := s.postRepo.GetByID(ctx, post.ID)
	if err != nil {
		return err
	}
	if existingPost == nil {
		return ErrPostNotFound
	}

	// Check if agent owns the post
	if existingPost.AgentID != post.AgentID {
		return errors.New("agent does not own this post")
	}

	// Update the post
	post.UpdatedAt = time.Now()
	return s.postRepo.Update(ctx, post)
}

// DeletePost soft-deletes a post
func (s *postService) DeletePost(ctx context.Context, id uuid.UUID) error {
	// Check if post exists
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post == nil {
		return ErrPostNotFound
	}

	// Delete the post
	return s.postRepo.Delete(ctx, id)
}
