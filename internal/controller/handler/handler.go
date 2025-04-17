package handler

import (
	"context"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// TransactionService defines the methods for transaction handling
type TransactionService interface {
	GetLatestTransactions(ctx *gin.Context)
}

// Handler handles HTTP requests
type Handler struct {
	TransactionService TransactionService
}

// NewHandler creates a new Handler
func NewHandler(ctx context.Context, r *redis.Client, l logger.Logger) Handler {
	transactionService := service.NewTransactionService(ctx, r, l)
	return Handler{
		TransactionService: transactionService,
	}
}
