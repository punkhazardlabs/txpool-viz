package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/controller/handler"
	route "txpool-viz/internal/controller/routes"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/service"
	"txpool-viz/internal/transactions"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Controller struct {
	Services   *service.Service
	Config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
	shutdown   chan struct{}
}

func NewController(cfg *config.Config, srvc *service.Service) *Controller {
	return &Controller{
		Config:   cfg,
		Services: srvc,
		router:   gin.Default(),
		shutdown: make(chan struct{}),
	}
}

func (c *Controller) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown signal handler
	go c.handleShutdown(cancel)

	// Initialize services and configurations
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	c.configureRouter(ctx, c.Services.Redis, c.Services.Logger)

	go func() {
		if err := c.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Call one method
	// Method Takes config and spins up a process for each endpoint
	go transactions.Stream(ctx, c.Config, c.Services)

	// Start polling transactions
	// go transactions.PollTransactions(ctx, c.Config, c.Services)

	// Start processing transactions
	// go broker.ProcessTransactions(ctx, c.Config, c.Services)

	<-ctx.Done()
	return nil
}

func (c *Controller) handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
	cancel()
}

func (c *Controller) initialize() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg

	srvc, err := c.setupServices()
	if err != nil {
		return fmt.Errorf("failed to set up services: %w", err)
	}
	c.Services = srvc
	return nil
}

func (c *Controller) setupServices() (*service.Service, error) {
	// Initialize redis client
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		return nil, errors.New("REDIS_URL environment variable is not set")
	}

	redisOptions, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing REDIS_URL: %w", err)
	}

	redisClient := redis.NewClient(redisOptions)

	// Initialize Postgres connection
	conn := os.Getenv("POSTGRES_URL")
	if conn == "" {
		return nil, errors.New("POSTGRES_URL environment variable is not set")
	}

	// Initialize Logger
	logger := logger.NewLogger(nil)

	return &service.Service{
		Redis:  redisClient,
		DB:     conn, // Assuming you connect to Postgres here
		Logger: logger,
	}, nil
}

func (c *Controller) configureRouter(ctx context.Context, r *redis.Client, l logger.Logger) {
	//Initialize handler with needed services
	txService := service.NewTransactionService(ctx, r, l)
	handler := handler.NewHandler(txService)

	// Register all routes
	route.RegisterRoutes(c.router, handler)

	// Configure server
	c.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler:      c.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
