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

func (ts *TransactionServiceImpl) GetLatestNTransactions(ctx context.Context, n int64) ([]string, error) {
	start := -n
	stop := int64(-1)

	results, err := ts.redis.ZRangeWithScores(ctx, utils.RedisUniversalKey(), start, stop).Result()
	if err != nil {
		ts.logger.Error("Redis ZRangeWithScores failed", err)
		return nil, err
	}

	var hashes []string
	for _, z := range results {
		hashes = append(hashes, z.Member.(string))
	}

	ts.logger.Info("GetLatestNTransactions called")
	return hashes, nil
}
