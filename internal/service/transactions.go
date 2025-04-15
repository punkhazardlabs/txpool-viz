package service

import (
	"context"
	"txpool-viz/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// const (
// 	pending = "pending"
// 	queued  = "queued"
// )

type transactionServiceImpl struct {
	redis  *redis.Client
	logger logger.Logger
}

// NewTransactionService creates a new transaction service
func NewTransactionService(ctx context.Context, r *redis.Client, l logger.Logger) *transactionServiceImpl {
	return &transactionServiceImpl{
		redis:  r,
		logger: l,
	}
}

// GetLatestTransactions handles the request to get the latest transactions
func (ts *transactionServiceImpl) GetLatestTransactions(ctx *gin.Context) {
	// This needs to supply a list of recent transactions

	// It will pull that from a queue
	ts.logger.Info("REST API Exposed")
}
