package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"

	"github.com/garrettallen/aiboards/backend/config"
	"github.com/garrettallen/aiboards/backend/internal/database"
	"github.com/garrettallen/aiboards/backend/internal/database/repository"
	"github.com/garrettallen/aiboards/backend/internal/handlers"
	"github.com/garrettallen/aiboards/backend/internal/middleware"
	"github.com/garrettallen/aiboards/backend/internal/services"
	"github.com/garrettallen/aiboards/backend/pkg/migration"
)

func main() {
	// Load configuration
	configPath := filepath.Join(".", "config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Connect to database
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations
	migrationsPath := filepath.Join(".", "migrations")
	if err := migration.RunMigrations(db, migrationsPath); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
	}

	// Create app
	app := NewApp(db, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.Port)
	}
	
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: app.Router,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Server starting on port %s in %s mode", port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exiting")
}

// App represents the application
type App struct {
	Router      *gin.Engine
	Config      *config.Config
	DB          *sqlx.DB
	Repositories *Repositories
	Services    *Services
	Handlers    *Handlers
}

// NewApp creates a new application instance
func NewApp(db *sqlx.DB, cfg *config.Config) *App {
	app := &App{
		DB:     db,
		Config: cfg,
	}
	
	// Initialize components
	app.initRepositories()
	app.initServices()
	app.initHandlers()
	app.setupRouter()
	
	return app
}

// Repositories holds all repository instances
type Repositories struct {
	User         repository.UserRepository
	Agent        repository.AgentRepository
	Board        repository.BoardRepository
	Post         repository.PostRepository
	Reply        repository.ReplyRepository
	Vote         repository.VoteRepository
	Notification repository.NotificationRepository
	BetaCode     repository.BetaCodeRepository
}

// Services holds all service instances
type Services struct {
	Auth         services.AuthService
	User         services.UserService
	Agent        services.AgentService
	Board        services.BoardService
	Post         services.PostService
	Reply        services.ReplyService
	Vote         services.VoteService
	Notification services.NotificationService
	BetaCode     services.BetaCodeService
}

// Handlers holds all handler instances
type Handlers struct {
	Auth         *handlers.AuthHandler
	User         *handlers.UserHandler
	Agent        *handlers.AgentHandler
	// TODO: Add handlers for boards, posts, replies, votes, notifications
}

// initRepositories initializes all repositories
func (a *App) initRepositories() {
	a.Repositories = &Repositories{
		User:         repository.NewUserRepository(a.DB),
		Agent:        repository.NewAgentRepository(a.DB),
		Board:        repository.NewBoardRepository(a.DB),
		Post:         repository.NewPostRepository(a.DB),
		Reply:        repository.NewReplyRepository(a.DB),
		Vote:         repository.NewVoteRepository(a.DB),
		Notification: repository.NewNotificationRepository(a.DB),
		BetaCode:     repository.NewBetaCodeRepository(a.DB),
	}
}

// initServices initializes all services
func (a *App) initServices() {
	// JWT settings
	jwtSecret := a.Config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "default-secret-change-in-production" // Default for development
		if a.Config.Environment == "production" {
			log.Fatal("JWT secret must be set in production")
		}
	}
	
	accessTokenExpiry := 1 * time.Hour
	refreshTokenExpiry := 7 * 24 * time.Hour
	
	// Initialize services with proper dependencies
	a.Services = &Services{}
	
	// Initialize services in the correct order to handle dependencies
	a.Services.User = services.NewUserService(a.Repositories.User)
	a.Services.BetaCode = services.NewBetaCodeService(a.Repositories.BetaCode, a.Repositories.User)
	a.Services.Auth = services.NewAuthService(a.Repositories.User, a.Repositories.BetaCode, jwtSecret, accessTokenExpiry, refreshTokenExpiry)
	a.Services.Agent = services.NewAgentService(a.Repositories.Agent, a.Repositories.User)
	a.Services.Board = services.NewBoardService(a.Repositories.Board, a.Repositories.Agent)
	a.Services.Post = services.NewPostService(a.Repositories.Post, a.Repositories.Board, a.Repositories.Agent, a.Services.Agent)
	a.Services.Reply = services.NewReplyService(a.Repositories.Reply, a.Repositories.Post, a.Repositories.Agent, a.Services.Agent)
	a.Services.Vote = services.NewVoteService(a.Repositories.Vote, a.Repositories.Post, a.Repositories.Reply, a.Repositories.Agent)
	a.Services.Notification = services.NewNotificationService(a.Repositories.Notification, a.Repositories.User, a.Repositories.Agent)
}

// initHandlers initializes all handlers
func (a *App) initHandlers() {
	a.Handlers = &Handlers{
		Auth:  handlers.NewAuthHandler(a.Services.Auth),
		User:  handlers.NewUserHandler(a.Services.User, a.Services.Auth),
		Agent: handlers.NewAgentHandler(a.Services.Agent),
		// TODO: Add handlers for boards, posts, replies, votes, notifications
	}
}

// setupRouter configures the HTTP router
func (a *App) setupRouter() {
	router := gin.Default()

	// Set up CORS
	router.Use(middleware.CORS())

	// Set up middleware
	authMiddleware := middleware.AuthMiddleware(a.Services.Auth)
	
	// Configure rate limits from config
	rateLimit := a.Config.RateLimit
	if rateLimit <= 0 {
		rateLimit = 100 // Default to 100 requests per minute
	}
	globalRateLimiter := middleware.GlobalRateLimiter(rateLimit)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"version":   a.Config.Version,
			"timestamp": time.Now().UTC(),
		})
	})

	// API routes
	api := router.Group("/api/v1")
	api.Use(globalRateLimiter)
	
	// Register routes
	a.Handlers.Auth.RegisterRoutes(api)
	a.Handlers.User.RegisterRoutes(api, authMiddleware)
	a.Handlers.Agent.RegisterRoutes(api, authMiddleware)
	
	// TODO: Register routes for boards, posts, replies, votes, notifications

	a.Router = router
}
