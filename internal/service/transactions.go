package service

import (
	"context"
	"encoding/json"
	"txpool-viz/config"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"github.com/redis/go-redis/v9"
)

type TransactionServiceImpl struct {
	redis     *redis.Client
	logger    logger.Logger
	endpoints []config.Endpoint
}

// NewTransactionService creates a new transaction service
func NewTransactionService(ctx context.Context, r *redis.Client, l logger.Logger, cfgEndpoints []config.Endpoint) *TransactionServiceImpl {
	return &TransactionServiceImpl{
		redis:     r,
		logger:    l,
		endpoints: cfgEndpoints,
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

	return hashes, nil
}

func (ts *TransactionServiceImpl) GetTxDetails(ctx context.Context, txHash string) (map[string]model.StoredTransaction, error) {
	res := make(map[string]model.StoredTransaction)
	// for each endpoint, retrieve the specific record
	for _, endpoint := range ts.endpoints {
		metaKey := utils.RedisClientMetaKey(endpoint.Name)

		// retrieve the record
		val, err := ts.redis.HGet(ctx, metaKey, txHash).Result()
		if err != nil {
			if err == redis.Nil {
				ts.logger.Debug("No record for %s in %s\n", txHash, metaKey)
				continue
			}
			ts.logger.Error("Redis error: %v\n", err)
			continue
		}

		// Unmarshal
		var storedTx model.StoredTransaction
		err = json.Unmarshal([]byte(val), &storedTx)
		if err != nil {
			ts.logger.Error("Failed to unmarshal transaction: %v\n", err)
			return nil, err
		}

		res[endpoint.Name] = storedTx
	}

	return res, nil
}
