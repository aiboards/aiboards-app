package integration

import (
	"testing"
	"time"

	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/models"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/tests/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotificationEnv extends the TestEnv with notification-specific components
type TestNotificationEnv struct {
	*utils.TestEnv
	NotificationRepository repository.NotificationRepository
	NotificationService    services.NotificationService
	PostRepository         repository.PostRepository
	ReplyRepository        repository.ReplyRepository
	VoteRepository         repository.VoteRepository
	BoardRepository        repository.BoardRepository
}

// NewTestNotificationEnv creates a new test environment with notification components
func NewTestNotificationEnv(t *testing.T) *TestNotificationEnv {
	baseEnv := utils.NewTestEnv(t)

	// Create repositories
	notificationRepo := repository.NewNotificationRepository(baseEnv.DB)
	postRepo := repository.NewPostRepository(baseEnv.DB)
	replyRepo := repository.NewReplyRepository(baseEnv.DB)
	voteRepo := repository.NewVoteRepository(baseEnv.DB)
	boardRepo := repository.NewBoardRepository(baseEnv.DB)

	// Create notification service
	notificationService := services.NewNotificationService(
		notificationRepo,
		baseEnv.UserRepository,
		baseEnv.AgentRepository,
	)

	return &TestNotificationEnv{
		TestEnv:                baseEnv,
		NotificationRepository: notificationRepo,
		NotificationService:    notificationService,
		PostRepository:         postRepo,
		ReplyRepository:        replyRepo,
		VoteRepository:         voteRepo,
		BoardRepository:        boardRepo,
	}
}

func TestCreateNotification_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Test data
	notificationType := services.NotificationTypeSystem
	content := "Test notification content"
	targetType := "post"
	targetID := uuid.New()

	// Call the CreateNotification method
	notification, err := env.NotificationService.CreateNotification(
		env.Ctx,
		agent.ID,
		notificationType,
		content,
		targetType,
		targetID,
	)

	// Assert results
	require.NoError(t, err)
	require.NotNil(t, notification)

	assert.Equal(t, agent.ID, notification.AgentID)
	assert.Equal(t, string(notificationType), notification.Type)
	assert.Equal(t, content, notification.Content)
	assert.Equal(t, targetType, notification.TargetType)
	assert.Equal(t, targetID, notification.TargetID)
	assert.False(t, notification.IsRead)
	assert.NotEmpty(t, notification.ID)
	assert.NotEmpty(t, notification.CreatedAt)
	assert.Nil(t, notification.ReadAt)

	// Verify notification can be retrieved from database
	retrievedNotification, err := env.NotificationRepository.GetByID(env.Ctx, notification.ID)
	require.NoError(t, err)
	require.NotNil(t, retrievedNotification)
	assert.Equal(t, notification.ID, retrievedNotification.ID)
	assert.Equal(t, notification.AgentID, retrievedNotification.AgentID)
}

func TestGetNotificationsByAgentID_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Create multiple test notifications
	for i := 0; i < 15; i++ {
		_, err := env.NotificationService.CreateNotification(
			env.Ctx,
			agent.ID,
			services.NotificationTypeSystem,
			"Test notification "+time.Now().String(),
			"post",
			uuid.New(),
		)
		require.NoError(t, err)
	}

	// Test pagination - page 1
	notifications1, total1, err := env.NotificationService.GetNotificationsByAgentID(env.Ctx, agent.ID, 1, 5)
	require.NoError(t, err)
	assert.Len(t, notifications1, 5)
	assert.GreaterOrEqual(t, total1, 15)

	// Test pagination - page 2
	notifications2, total2, err := env.NotificationService.GetNotificationsByAgentID(env.Ctx, agent.ID, 2, 5)
	require.NoError(t, err)
	assert.Len(t, notifications2, 5)
	assert.Equal(t, total1, total2)

	// Verify different pages return different notifications
	assert.NotEqual(t, notifications1[0].ID, notifications2[0].ID)
}

func TestMarkAsRead_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Create a test notification
	notification, err := env.NotificationService.CreateNotification(
		env.Ctx,
		agent.ID,
		services.NotificationTypeSystem,
		"Test notification",
		"post",
		uuid.New(),
	)
	require.NoError(t, err)
	require.NotNil(t, notification)
	assert.False(t, notification.IsRead)

	// Mark as read
	err = env.NotificationService.MarkAsRead(env.Ctx, notification.ID)
	require.NoError(t, err)

	// Verify notification is marked as read
	updatedNotification, err := env.NotificationRepository.GetByID(env.Ctx, notification.ID)
	require.NoError(t, err)
	require.NotNil(t, updatedNotification)
	assert.True(t, updatedNotification.IsRead)
	assert.NotNil(t, updatedNotification.ReadAt)
}

func TestMarkAllAsRead_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Create multiple test notifications
	for i := 0; i < 5; i++ {
		_, err := env.NotificationService.CreateNotification(
			env.Ctx,
			agent.ID,
			services.NotificationTypeSystem,
			"Test notification "+time.Now().String(),
			"post",
			uuid.New(),
		)
		require.NoError(t, err)
	}

	// Verify there are unread notifications
	unreadCount, err := env.NotificationService.CountUnread(env.Ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, unreadCount)

	// Mark all as read
	err = env.NotificationService.MarkAllAsRead(env.Ctx, agent.ID)
	require.NoError(t, err)

	// Verify all notifications are marked as read
	unreadCount, err = env.NotificationService.CountUnread(env.Ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, unreadCount)
}

func TestDeleteNotification_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Create a test notification
	notification, err := env.NotificationService.CreateNotification(
		env.Ctx,
		agent.ID,
		services.NotificationTypeSystem,
		"Test notification",
		"post",
		uuid.New(),
	)
	require.NoError(t, err)
	require.NotNil(t, notification)

	// Delete the notification
	err = env.NotificationService.DeleteNotification(env.Ctx, notification.ID)
	require.NoError(t, err)

	// Verify notification is deleted
	deletedNotification, err := env.NotificationRepository.GetByID(env.Ctx, notification.ID)
	assert.Error(t, err)
	assert.Nil(t, deletedNotification)
}

func TestCountUnread_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent
	userID, _ := env.CreateTestUser()
	agent := env.CreateTestAgent(userID)

	// Create multiple test notifications
	for i := 0; i < 5; i++ {
		_, err := env.NotificationService.CreateNotification(
			env.Ctx,
			agent.ID,
			services.NotificationTypeSystem,
			"Test notification "+time.Now().String(),
			"post",
			uuid.New(),
		)
		require.NoError(t, err)
	}

	// Count unread notifications
	unreadCount, err := env.NotificationService.CountUnread(env.Ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, 5, unreadCount)

	// Mark one notification as read
	notifications, _, err := env.NotificationService.GetNotificationsByAgentID(env.Ctx, agent.ID, 1, 1)
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	err = env.NotificationService.MarkAsRead(env.Ctx, notifications[0].ID)
	require.NoError(t, err)

	// Verify unread count is updated
	unreadCount, err = env.NotificationService.CountUnread(env.Ctx, agent.ID)
	require.NoError(t, err)
	assert.Equal(t, 4, unreadCount)
}

func TestNotifyOnReply_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent for the post owner
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	// Create a test user and agent for the reply creator
	replyCreatorUserID, _ := env.CreateTestUser()
	replyCreatorAgent := env.CreateTestAgent(replyCreatorUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create a test reply
	reply := &models.Reply{
		ID:         uuid.New(),
		AgentID:    replyCreatorAgent.ID,
		ParentID:   post.ID,
		ParentType: "post",
		Content:    "Test reply",
		CreatedAt:  time.Now(),
	}
	err = env.ReplyRepository.Create(env.Ctx, reply)
	require.NoError(t, err)

	// Notify on reply
	err = env.NotificationService.NotifyOnReply(env.Ctx, reply, post)
	require.NoError(t, err)

	// Verify notification was created
	unreadCount, err := env.NotificationService.CountUnread(env.Ctx, postOwnerAgent.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, unreadCount)

	// Get the notification and verify its properties
	notifications, _, err := env.NotificationService.GetNotificationsByAgentID(env.Ctx, postOwnerAgent.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	notification := notifications[0]
	assert.Equal(t, postOwnerAgent.ID, notification.AgentID)
	assert.Equal(t, string(services.NotificationTypeReply), notification.Type)
	assert.Equal(t, "New reply to your post", notification.Content)
	assert.Equal(t, "post", notification.TargetType)
	assert.Equal(t, reply.ID, notification.TargetID)
}

func TestNotifyOnVote_Integration(t *testing.T) {
	// Create a test environment with a real database
	env := NewTestNotificationEnv(t)
	defer env.Cleanup()

	// Create a test user and agent for the post owner
	postOwnerUserID, _ := env.CreateTestUser()
	postOwnerAgent := env.CreateTestAgent(postOwnerUserID)

	// Create a test user and agent for the voter
	voterUserID, _ := env.CreateTestUser()
	voterAgent := env.CreateTestAgent(voterUserID)

	// Create a test board
	board := &models.Board{
		ID:          uuid.New(),
		AgentID:     postOwnerAgent.ID,
		Title:       "Test Board",
		Description: "Test Board Description",
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := env.BoardRepository.Create(env.Ctx, board)
	require.NoError(t, err)

	// Create a test post
	post := &models.Post{
		ID:        uuid.New(),
		BoardID:   board.ID,
		AgentID:   postOwnerAgent.ID,
		Content:   "Test content",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = env.PostRepository.Create(env.Ctx, post)
	require.NoError(t, err)

	// Create a test upvote
	upvote := &models.Vote{
		ID:         uuid.New(),
		AgentID:    voterAgent.ID,
		TargetID:   post.ID,
		TargetType: "post",
		Value:      1,
		CreatedAt:  time.Now(),
	}
	err = env.VoteRepository.Create(env.Ctx, upvote)
	require.NoError(t, err)

	// Notify on vote
	err = env.NotificationService.NotifyOnVote(env.Ctx, upvote, postOwnerAgent.ID)
	require.NoError(t, err)

	// Verify notification was created
	unreadCount, err := env.NotificationService.CountUnread(env.Ctx, postOwnerAgent.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, unreadCount)

	// Get the notification and verify its properties
	notifications, _, err := env.NotificationService.GetNotificationsByAgentID(env.Ctx, postOwnerAgent.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 1)

	notification := notifications[0]
	assert.Equal(t, postOwnerAgent.ID, notification.AgentID)
	assert.Equal(t, string(services.NotificationTypeVote), notification.Type)
	assert.Equal(t, "Someone upvoted your post", notification.Content)
	assert.Equal(t, "post", notification.TargetType)
	assert.Equal(t, upvote.ID, notification.TargetID)

	// Create a test downvote on a reply
	reply := &models.Reply{
		ID:         uuid.New(),
		AgentID:    postOwnerAgent.ID,
		ParentID:   post.ID,
		ParentType: "post",
		Content:    "Test reply",
		CreatedAt:  time.Now(),
	}
	err = env.ReplyRepository.Create(env.Ctx, reply)
	require.NoError(t, err)

	downvote := &models.Vote{
		ID:         uuid.New(),
		AgentID:    voterAgent.ID,
		TargetID:   reply.ID,
		TargetType: "reply",
		Value:      -1,
		CreatedAt:  time.Now(),
	}
	err = env.VoteRepository.Create(env.Ctx, downvote)
	require.NoError(t, err)

	// Notify on downvote
	err = env.NotificationService.NotifyOnVote(env.Ctx, downvote, postOwnerAgent.ID)
	require.NoError(t, err)

	// Verify another notification was created
	unreadCount, err = env.NotificationService.CountUnread(env.Ctx, postOwnerAgent.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, unreadCount)

	// Get the notifications and verify the new one's properties
	notifications, _, err = env.NotificationService.GetNotificationsByAgentID(env.Ctx, postOwnerAgent.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, notifications, 2)

	// Find the downvote notification (should be the newest one)
	var downvoteNotification *models.Notification
	for _, n := range notifications {
		if n.TargetID == downvote.ID {
			downvoteNotification = n
			break
		}
	}

	require.NotNil(t, downvoteNotification)
	assert.Equal(t, postOwnerAgent.ID, downvoteNotification.AgentID)
	assert.Equal(t, string(services.NotificationTypeVote), downvoteNotification.Type)
	assert.Equal(t, "Someone downvoted your reply", downvoteNotification.Content)
	assert.Equal(t, "reply", downvoteNotification.TargetType)
	assert.Equal(t, downvote.ID, downvoteNotification.TargetID)
}
