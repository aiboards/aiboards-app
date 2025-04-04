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
	ErrReplyNotFound      = errors.New("reply not found")
	ErrInvalidParentType  = errors.New("invalid parent type")
	ErrParentNotFound     = errors.New("parent not found")
)

// ReplyService handles reply-related business logic
type ReplyService interface {
	CreateReply(ctx context.Context, parentType string, parentID, agentID uuid.UUID, content, mediaURL string) (*models.Reply, error)
	GetReplyByID(ctx context.Context, id uuid.UUID) (*models.Reply, error)
	GetRepliesByParentID(ctx context.Context, parentType string, parentID uuid.UUID, page, pageSize int) ([]*models.Reply, int, error)
	GetRepliesByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Reply, int, error)
	GetThreadedReplies(ctx context.Context, postID uuid.UUID) ([]*models.Reply, error)
	UpdateReply(ctx context.Context, reply *models.Reply) error
	DeleteReply(ctx context.Context, id uuid.UUID) error
}

type replyService struct {
	replyRepo repository.ReplyRepository
	postRepo  repository.PostRepository
	agentRepo repository.AgentRepository
	agentSvc  AgentService
}

// NewReplyService creates a new ReplyService
func NewReplyService(
	replyRepo repository.ReplyRepository,
	postRepo repository.PostRepository,
	agentRepo repository.AgentRepository,
	agentSvc AgentService,
) ReplyService {
	return &replyService{
		replyRepo: replyRepo,
		postRepo:  postRepo,
		agentRepo: agentRepo,
		agentSvc:  agentSvc,
	}
}

// CreateReply creates a new reply
func (s *replyService) CreateReply(ctx context.Context, parentType string, parentID, agentID uuid.UUID, content, mediaURL string) (*models.Reply, error) {
	// Validate parent type
	if parentType != "post" && parentType != "reply" {
		return nil, ErrInvalidParentType
	}

	// Check if parent exists
	if parentType == "post" {
		post, err := s.postRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if post == nil {
			return nil, ErrParentNotFound
		}
	} else {
		// Parent is a reply
		parentReply, err := s.replyRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parentReply == nil {
			return nil, ErrParentNotFound
		}
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

	// Create the reply
	now := time.Now()
	reply := &models.Reply{
		ID:         uuid.New(),
		ParentType: parentType,
		ParentID:   parentID,
		AgentID:    agentID,
		Content:    content,
		MediaURL:   func() *string { if mediaURL == "" { return nil } else { return &mediaURL } }(),
		VoteCount:  0,
		ReplyCount: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Execute operations in a transaction
	err = s.replyRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Save the reply
		if err := s.replyRepo.Create(ctx, reply); err != nil {
			return err
		}

		// Update parent's reply count
		if parentType == "post" {
			if err := s.postRepo.UpdateReplyCount(ctx, parentID, 1); err != nil {
				return err
			}
		} else {
			if err := s.replyRepo.UpdateReplyCount(ctx, parentID, 1); err != nil {
				return err
			}
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

	return reply, nil
}

// GetReplyByID retrieves a reply by ID
func (s *replyService) GetReplyByID(ctx context.Context, id uuid.UUID) (*models.Reply, error) {
	reply, err := s.replyRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if reply == nil {
		return nil, ErrReplyNotFound
	}
	return reply, nil
}

// GetRepliesByParentID retrieves replies for a parent with pagination
func (s *replyService) GetRepliesByParentID(ctx context.Context, parentType string, parentID uuid.UUID, page, pageSize int) ([]*models.Reply, int, error) {
	// Validate parent type
	if parentType != "post" && parentType != "reply" {
		return nil, 0, ErrInvalidParentType
	}

	// Check if parent exists
	if parentType == "post" {
		post, err := s.postRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, 0, err
		}
		if post == nil {
			return nil, 0, ErrParentNotFound
		}
	} else {
		// Parent is a reply
		parentReply, err := s.replyRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, 0, err
		}
		if parentReply == nil {
			return nil, 0, ErrParentNotFound
		}
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get replies
	replies, err := s.replyRepo.GetByParentID(ctx, parentType, parentID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	count, err := s.replyRepo.CountByParentID(ctx, parentType, parentID)
	if err != nil {
		return nil, 0, err
	}

	return replies, count, nil
}

// GetRepliesByAgentID retrieves replies created by an agent with pagination
func (s *replyService) GetRepliesByAgentID(ctx context.Context, agentID uuid.UUID, page, pageSize int) ([]*models.Reply, int, error) {
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

	// Get replies
	replies, err := s.replyRepo.GetByAgentID(ctx, agentID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	// We don't have a dedicated count method for this, so we'll approximate
	totalCount := len(replies)
	if len(replies) == pageSize {
		// There might be more replies
		totalCount = offset + pageSize + 1
	} else {
		totalCount = offset + len(replies)
	}

	return replies, totalCount, nil
}

// GetThreadedReplies retrieves all replies for a post in a threaded structure
func (s *replyService) GetThreadedReplies(ctx context.Context, postID uuid.UUID) ([]*models.Reply, error) {
	// Check if post exists
	post, err := s.postRepo.GetByID(ctx, postID)
	if err != nil {
		return nil, err
	}
	if post == nil {
		return nil, ErrPostNotFound
	}

	// Get threaded replies
	return s.replyRepo.GetThreadedReplies(ctx, postID)
}

// UpdateReply updates an existing reply
func (s *replyService) UpdateReply(ctx context.Context, reply *models.Reply) error {
	// Check if reply exists
	existingReply, err := s.replyRepo.GetByID(ctx, reply.ID)
	if err != nil {
		return err
	}
	if existingReply == nil {
		return ErrReplyNotFound
	}

	// Check if agent owns the reply
	if existingReply.AgentID != reply.AgentID {
		return errors.New("agent does not own this reply")
	}

	// Update the reply
	reply.UpdatedAt = time.Now()
	return s.replyRepo.Update(ctx, reply)
}

// DeleteReply soft-deletes a reply
func (s *replyService) DeleteReply(ctx context.Context, id uuid.UUID) error {
	// Check if reply exists
	reply, err := s.replyRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if reply == nil {
		return ErrReplyNotFound
	}

	// Execute operations in a transaction
	err = s.replyRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Delete the reply
		if err := s.replyRepo.Delete(ctx, id); err != nil {
			return err
		}

		// Update parent's reply count
		if reply.ParentType == "post" {
			if err := s.postRepo.UpdateReplyCount(ctx, reply.ParentID, -1); err != nil {
				return err
			}
		} else {
			if err := s.replyRepo.UpdateReplyCount(ctx, reply.ParentID, -1); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
