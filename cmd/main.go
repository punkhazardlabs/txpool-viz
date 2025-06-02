package main

import (
	"log"

	"txpool-viz/internal/config"
	"txpool-viz/internal/controller"
	"txpool-viz/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize services
	srvc, err := service.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Create and start the controller
	ctrl := controller.NewController(cfg, srvc)
	if err := ctrl.Serve(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
