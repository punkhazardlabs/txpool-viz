package main

import (
	"fmt"
	"log"
	
	"txpool-viz/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	fmt.Println(cfg)
}