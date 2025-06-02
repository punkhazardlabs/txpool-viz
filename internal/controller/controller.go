package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		l.Info("Serving txpool-viz at http://localhost:" + os.Getenv("PORT"))
		if err := c.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Error("server failed to start", logger.Fields{"error": err.Error()})
		}
	}()

	// Start transaction streams and respective processors
	wg.Add(1)
	go func() {
		defer wg.Done()
		transactions.Stream(ctx, c.Config, c.Services, &wg)
	}()

	if c.Config.FocilEnabled == "true" {
		// Start inclusion list SSE listener if url is configured
		wg.Add(1)
		go func() {
			defer wg.Done()
			inclusionListsService := inclusion_list.NewInclusionListService(l, c.Services.Redis)
			inclusionListsService.Stream(ctx, c.Config.Endpoints, c.Config.BeaconUrls)
		}()
	}

	// Wait for shutdown signal
	<-ctx.Done()

	l.Info("Shutdown signal received, shutting down services...")

	// Cleanly shut down HTTP servers
	_ = c.httpServer.Shutdown(context.Background())

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

	srvc, err := service.NewService(c.Config)
	if err != nil {
		return fmt.Errorf("failed to set up services: %w", err)
	}
	c.Services = srvc
	return nil
}

func (c *Controller) configureRouter(ctx context.Context, r *redis.Client, l logger.Logger) {
	//Initialize handler with needed services
	txService := service.NewTransactionService(ctx, r, l, c.Config.Endpoints)
	ilService := service.NewInclusionListService(r, l)
	handler := handler.NewHandler(txService, ilService)

	c.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"}, // Restrict to required methods
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register all routes
	route.RegisterRoutes(c.router, handler)

	// Create a mux to serve both API and static frontend
	mux := http.NewServeMux()

	// API routes — mounted on /api/
	mux.Handle("/api/", http.StripPrefix("/api", c.router))

	// Frontend static files — mounted at /
	fs := http.FileServer(http.Dir("./frontend/dist"))
	mux.Handle("/", fs)

	// Configure server
	c.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
