package service

import (
	"context"
	"txpool-viz/internal/logger"
	"txpool-viz/utils"

	"github.com/redis/go-redis/v9"
)

type TransactionServiceImpl struct {
	redis  *redis.Client
	logger logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(ctx context.Context, r *redis.Client, l logger.Logger) *TransactionServiceImpl {
	return &TransactionServiceImpl{
		redis:  r,
		logger: l,
	}
}

// GetLatestTransactions handles the request to get the latest transactions
func (ts *TransactionServiceImpl) GetLatestTransactions(ctx context.Context, start int64, stop int64) ([]string, error) {
	if stop == 0 {
		stop = -1
	}

	results, err := ts.redis.ZRangeWithScores(ctx, utils.RedisUniversalKey(), start, stop).Result()
	if err != nil {
		ts.logger.Error("Redis ZRangeWithScores failed", err)
		return nil, err
	}

	var hashes []string
	for _, z := range results {
		hashes = append(hashes, z.Member.(string))
	}

	return hashes, nil
}
