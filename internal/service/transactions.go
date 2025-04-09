package service

import (
	"context"
	"txpool-viz/pkg"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// const (
// 	pending = "pending"
// 	queued  = "queued"
// )

type transactionServiceImpl struct {
	redis  *redis.Client
	logger pkg.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(ctx context.Context, r *redis.Client, l pkg.Logger) *transactionServiceImpl {
	return &transactionServiceImpl{
		redis:  r,
		logger: l,
	}
}

// GetLatestTransactions handles the request to get the latest transactions
func (ts *transactionServiceImpl) GetLatestTransactions(ctx *gin.Context) {
	// pendingTxs := ts.redis.HGetAll(ctx, pending)
	// queuedTxs := ts.redis.HGetAll(ctx, queued)

	ts.logger.Info("REST API Exposed")
}
