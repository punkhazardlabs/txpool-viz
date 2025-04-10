package service

import (
	"context"
	"fmt"
	"os"

	"txpool-viz/config"
	"txpool-viz/pkg"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	Redis  *redis.Client
	DB     *pgxpool.Pool
	Logger pkg.Logger
}

func NewService(cfg *config.Config, ctx context.Context) (*Service, error) {
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
	connectionString := os.Getenv("POSTGRES_URL")
	if connectionString == "" {
		return nil, fmt.Errorf("POSTGRES_URL environment variable is not set")
	}

	conn, err := pgxpool.New(ctx, connectionString)

	if err != nil {
		return nil, fmt.Errorf("Error connecting to database. Err: %s", err)
	}

	// Initialize Logger
	logger := pkg.NewLogger(nil)

	return &Service{
		Redis:  redisClient,
		DB:     conn, // Assuming you connect to Postgres here
		Logger: logger,
	}, nil
}
