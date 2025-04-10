package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"txpool-viz/config"
	// "txpool-viz/internal/broker"
	"txpool-viz/internal/controller/handler"
	route "txpool-viz/internal/controller/routes"
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

func NewController(cfg *config.Config, srvc *service.Service) *Controller {
	return &Controller{
		Config:   cfg,
		Services: srvc,
		router:   gin.Default(),
		shutdown: make(chan struct{}),
	}
}

func (c *Controller) Serve(ctx context.Context) error {
	// Initialize services and configurations
	if err := c.initialize(ctx); err != nil {
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

func (c *Controller) initialize(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	c.Config = cfg

	srvc, err := service.NewService(cfg, ctx)
	if err != nil {
		return fmt.Errorf("failed to set up services: %w", err)
	}
	c.Services = srvc
	return nil
}

func (c *Controller) configureRouter(ctx context.Context, r *redis.Client, l pkg.Logger) {
	handler := handler.NewHandler(ctx, r, l)

	route.RegisterRoutes(c.router, &handler)

	c.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%s", os.Getenv("PORT")),
		Handler:      c.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}
