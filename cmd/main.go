package main

import (
	"log"

	"txpool-viz/internal/controller"
)

func main() {
	if err := controller.New().Serve(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}