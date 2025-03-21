package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"txpool-viz/config"
	"txpool-viz/internal/broker"
	"txpool-viz/internal/transactions"
)

func main() {
	cfg, err := config.Load()

	ctx, cancel := context.WithCancel(context.Background())

	// Handle OS signals for graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	
	// Start polling transactions
	transactions.PollTransactions(ctx, cfg)

	// Start processing transactions
	broker.ProcessTransactions(ctx, cfg)

	<-ctx.Done()
	log.Println("Shutdown complete")
}
