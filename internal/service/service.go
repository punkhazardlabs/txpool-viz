package service

import (
	"context"
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

	// Wipe redis keys for a fresh instance
	redisClient.FlushAll(context.Background())

	// Initialize Postgres connection
	conn := os.Getenv("POSTGRES_URL")
	if conn == "" {
		return nil, fmt.Errorf("POSTGRES_URL environment variable is not set")
	}

	devEnvironment := os.Getenv("ENV") != "prod"
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info" // Default log level if not set
	}

	loggerConfig := &logger.LoggerConfig{
		Development: devEnvironment,                // Use development mode if not prod,
		Level:       logger.LogLevel(cfg.LogLevel), // Set log level from config
	}

	// Initialize Logger
	logger := logger.NewLogger(loggerConfig)

	return &Service{
		Redis:  redisClient,
		DB:     conn,
		Logger: logger,
	}, nil
}
