package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"txpool-viz/config"
	"txpool-viz/internal/controller"
	"txpool-viz/internal/service"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown signal handler
	go handleShutdown(cancel)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize services
	srvc, err := service.NewService(cfg, ctx)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Create and start the controller
	ctrl := controller.NewController(cfg, srvc)
	if err := ctrl.Serve(ctx); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")
	cancel()
}
