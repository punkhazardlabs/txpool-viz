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
	"txpool-viz/internal/broker"
	"txpool-viz/internal/service"
	"txpool-viz/internal/transactions"
	"txpool-viz/pkg"

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

func New() *Controller {
	return &Controller{
		router:   gin.Default(),
		shutdown: make(chan struct{}),
	}
}

func (c *Controller) Serve() error {
	ctx, cancel := context.WithCancel(context.Background())

	// Graceful shutdown signal handler
	go func() {
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan
			log.Println("Shutting down...")
			cancel()
	}()

	// Initialize services and configurations
	if err := c.initialize(); err != nil {
			return err
	}

	c.configureRouter()

	go func() {
			if err := c.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Fatalf("Server failed: %v", err)
			}
	}()

	// Start polling transactions
	go transactions.PollTransactions(ctx, c.Config, c.Services)

	// Start processing transactions
	go broker.ProcessTransactions(ctx, c.Config, c.Services)

	<-ctx.Done()

	return nil
}

func (c *Controller) initialize() error {
	if cfg, err := config.Load(); err != nil {
		return err
	} else {
		c.Config = cfg
	}

	if srvc, err := c.setupServices(); err != nil {
		return err
	} else {
		c.Services = srvc
	}

	return nil
}

func (c *Controller) setupServices() (*service.Service, error) {
	// Initialize redis client
	redisUrl := os.Getenv("REDIS_URL")

	if redisUrl == "" {
		return nil, fmt.Errorf("Error reading REDIS_URL from environment")
	}

	redisOptions, err := redis.ParseURL(redisUrl)

	if err != nil {
		return nil, fmt.Errorf("Error parsing REDIS_URL: %v", err)
	}

	redisClient := redis.NewClient(redisOptions)

	// Initialize Postgres connection
	conn := os.Getenv("POSTGRES_URL")

	if conn == "" {
		return nil, fmt.Errorf("Error reading POSTGRES_URL: %v", err)
	}

	// Initialize Logger
	logger := pkg.NewLogger(nil)

	return &service.Service{
		Redis:  redisClient,
		DB:     "",
		Logger: logger,
	}, nil
}

func (c *Controller) configureRouter() {
	c.httpServer = &http.Server{
		Addr:         "localhost:8080",
		Handler:      c.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
