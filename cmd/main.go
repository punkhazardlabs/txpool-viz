package main

import (
	"log"
	
	"txpool-viz/config"
	"txpool-viz/internal/transactions"
)

func main() {
	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Start polling transactions
	transactions.PollTransactions(cfg)
}