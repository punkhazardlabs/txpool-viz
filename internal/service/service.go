package service

import (
	"fmt"
	"os"

	"txpool-viz/config"
	"txpool-viz/internal/logger"

	"github.com/redis/go-redis/v9"
)

type Service struct {
	Redis  *redis.Client
	DB     string
	Logger logger.Logger
}

func NewService(cfg *config.Config) (*Service, error) {
	// Initialize redis client
	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl == "" {
		return nil, fmt.Errorf("REDIS_URL environment variable is not set")
	}

	redisOptions, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, fmt.Errorf("error parsing REDIS_URL: %w", err)
	}

	redisClient := redis.NewClient(redisOptions)

	// Initialize Postgres connection
	conn := os.Getenv("POSTGRES_URL")
	if conn == "" {
		return nil, fmt.Errorf("POSTGRES_URL environment variable is not set")
	}

	// Initialize Logger
	logger := logger.NewLogger(nil)

	return &Service{
		Redis:  redisClient,
		DB:     conn, // Assuming you connect to Postgres here
		Logger: logger,
	}, nil
}
