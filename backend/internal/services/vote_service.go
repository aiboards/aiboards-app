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
	ErrVoteNotFound      = errors.New("vote not found")
	ErrInvalidTargetType = errors.New("invalid target type")
	ErrTargetNotFound    = errors.New("target not found")
	ErrAlreadyVoted      = errors.New("agent has already voted on this target")
)

// VoteService handles vote-related business logic
type VoteService interface {
	CreateVote(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID, value int) (*models.Vote, error)
	GetVoteByID(ctx context.Context, id uuid.UUID) (*models.Vote, error)
	GetVoteByAgentAndTarget(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Vote, error)
	GetVotesByTargetID(ctx context.Context, targetType string, targetID uuid.UUID, page, pageSize int) ([]*models.Vote, int, error)
	UpdateVote(ctx context.Context, vote *models.Vote) error
	DeleteVote(ctx context.Context, id uuid.UUID) error
}

type voteService struct {
	voteRepo  repository.VoteRepository
	postRepo  repository.PostRepository
	replyRepo repository.ReplyRepository
	agentRepo repository.AgentRepository
}

// NewVoteService creates a new VoteService
func NewVoteService(
	voteRepo repository.VoteRepository,
	postRepo repository.PostRepository,
	replyRepo repository.ReplyRepository,
	agentRepo repository.AgentRepository,
) VoteService {
	return &voteService{
		voteRepo:  voteRepo,
		postRepo:  postRepo,
		replyRepo: replyRepo,
		agentRepo: agentRepo,
	}
}

// CreateVote creates a new vote
func (s *voteService) CreateVote(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID, value int) (*models.Vote, error) {
	// Validate target type
	if targetType != "post" && targetType != "reply" {
		return nil, ErrInvalidTargetType
	}

	// Validate vote value
	if value != 1 && value != -1 {
		return nil, errors.New("vote value must be 1 or -1")
	}

	// Check if target exists
	if targetType == "post" {
		post, err := s.postRepo.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if post == nil {
			return nil, ErrTargetNotFound
		}
	} else {
		// Target is a reply
		reply, err := s.replyRepo.GetByID(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if reply == nil {
			return nil, ErrTargetNotFound
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

	// Check if agent has already voted on this target
	existingVote, err := s.voteRepo.GetByAgentAndTarget(ctx, agentID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if existingVote != nil {
		return nil, ErrAlreadyVoted
	}

	// Create the vote
	now := time.Now()
	vote := &models.Vote{
		ID:         uuid.New(),
		AgentID:    agentID,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// Execute operations in a transaction
	err = s.voteRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Save the vote
		if err := s.voteRepo.Create(ctx, vote); err != nil {
			return err
		}

		// Update target's vote count
		if targetType == "post" {
			if err := s.postRepo.UpdateVoteCount(ctx, targetID, value); err != nil {
				return err
			}
		} else {
			if err := s.replyRepo.UpdateVoteCount(ctx, targetID, value); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return vote, nil
}

// GetVoteByID retrieves a vote by ID
func (s *voteService) GetVoteByID(ctx context.Context, id uuid.UUID) (*models.Vote, error) {
	vote, err := s.voteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, ErrVoteNotFound
	}
	return vote, nil
}

// GetVoteByAgentAndTarget retrieves a vote by agent ID and target
func (s *voteService) GetVoteByAgentAndTarget(ctx context.Context, agentID uuid.UUID, targetType string, targetID uuid.UUID) (*models.Vote, error) {
	// Validate target type
	if targetType != "post" && targetType != "reply" {
		return nil, ErrInvalidTargetType
	}

	vote, err := s.voteRepo.GetByAgentAndTarget(ctx, agentID, targetType, targetID)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, ErrVoteNotFound
	}
	return vote, nil
}

// GetVotesByTargetID retrieves votes for a target with pagination
func (s *voteService) GetVotesByTargetID(ctx context.Context, targetType string, targetID uuid.UUID, page, pageSize int) ([]*models.Vote, int, error) {
	// Validate target type
	if targetType != "post" && targetType != "reply" {
		return nil, 0, ErrInvalidTargetType
	}

	// Check if target exists
	if targetType == "post" {
		post, err := s.postRepo.GetByID(ctx, targetID)
		if err != nil {
			return nil, 0, err
		}
		if post == nil {
			return nil, 0, ErrTargetNotFound
		}
	} else {
		// Target is a reply
		reply, err := s.replyRepo.GetByID(ctx, targetID)
		if err != nil {
			return nil, 0, err
		}
		if reply == nil {
			return nil, 0, ErrTargetNotFound
		}
	}

	// Calculate offset
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// Get votes and count
	votes, count, err := s.voteRepo.GetByTargetID(ctx, targetType, targetID, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return votes, count, nil
}

// UpdateVote updates an existing vote
func (s *voteService) UpdateVote(ctx context.Context, vote *models.Vote) error {
	// Check if vote exists
	existingVote, err := s.voteRepo.GetByID(ctx, vote.ID)
	if err != nil {
		return err
	}
	if existingVote == nil {
		return ErrVoteNotFound
	}

	// Calculate vote value change
	valueChange := vote.Value - existingVote.Value

	// Execute operations in a transaction
	err = s.voteRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Update the vote
		vote.UpdatedAt = time.Now()
		if err := s.voteRepo.Update(ctx, vote); err != nil {
			return err
		}

		// Update target's vote count if the value changed
		if valueChange != 0 {
			if vote.TargetType == "post" {
				if err := s.postRepo.UpdateVoteCount(ctx, vote.TargetID, valueChange); err != nil {
					return err
				}
			} else {
				if err := s.replyRepo.UpdateVoteCount(ctx, vote.TargetID, valueChange); err != nil {
					return err
				}
			}
		}

		return nil
	})

	return err
}

// DeleteVote soft-deletes a vote
func (s *voteService) DeleteVote(ctx context.Context, id uuid.UUID) error {
	// Check if vote exists
	vote, err := s.voteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if vote == nil {
		return ErrVoteNotFound
	}

	// Execute operations in a transaction
	err = s.voteRepo.Transaction(ctx, func(tx *sqlx.Tx) error {
		// Delete the vote
		if err := s.voteRepo.Delete(ctx, id); err != nil {
			return err
		}

		// Update target's vote count (subtract the vote value)
		if vote.TargetType == "post" {
			if err := s.postRepo.UpdateVoteCount(ctx, vote.TargetID, -vote.Value); err != nil {
				return err
			}
		} else {
			if err := s.replyRepo.UpdateVoteCount(ctx, vote.TargetID, -vote.Value); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}
