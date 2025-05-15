package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/controller/handler"
	route "txpool-viz/internal/controller/routes"
	"txpool-viz/internal/inclusion_list"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/service"
	"txpool-viz/internal/transactions"

	"github.com/gin-contrib/cors"
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
	l := c.Services.Logger
	defer cancel()

	var wg sync.WaitGroup

	// Graceful shutdown signal handler
	go c.handleShutdown(cancel)

	// Initialize services and configurations
	if err := c.initialize(); err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	c.configureRouter(ctx, c.Services.Redis, l)

	// Start backend HTTP API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("API server failed", logger.Fields{"error": err.Error()})
		}
	}()

	// Start transaction streams and respective processors
	wg.Add(1)
	go func() {
		defer wg.Done()
		transactions.Stream(ctx, c.Config, c.Services, &wg)
	}()

	// Start inclusion list SSE listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		inclusionListService := inclusion_list.NewInclusionListService(l, c.Services.Redis)
		inclusionListService.StreamInclusionList(ctx, c.Config.BeaconSSEUrl, c.Config.Endpoints[0].Websocket, c.Config.Endpoints[0].Client)
	}()

	// Start frontend static file server
	frontendServer := &http.Server{
		Addr:    ":8080",
		Handler: http.FileServer(http.Dir("./frontend/dist")),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		l.Info("Serving frontend at http://localhost:8080")
		if err := frontendServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("Frontend server failed", logger.Fields{"error": err.Error()})
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	l.Info("Shutdown signal received, shutting down services...")

	// Cleanly shut down HTTP servers
	_ = c.httpServer.Shutdown(context.Background())
	_ = frontendServer.Shutdown(context.Background())

	l.Info("Waiting for background routines to finish...")
	wg.Wait()

	l.Info("All services shut down cleanly")
	return nil
}

func (c *Controller) handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	cancel() // Cancel the context to unblock Serve()
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

	// Wipe redis keys for a fresh instance
	redisClient.FlushAll(context.Background())

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
	txService := service.NewTransactionService(ctx, r, l, c.Config.Endpoints)
	handler := handler.NewHandler(txService)

	allowedOrigins := "http://localhost:8080" // front-end port

	c.router.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(allowedOrigins, ","),
		AllowMethods:     []string{"GET", "POST"}, // Restrict to required methods
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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
