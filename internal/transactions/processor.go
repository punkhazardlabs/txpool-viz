package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/internal/service"
	"txpool-viz/internal/storage"
	"txpool-viz/utils"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
)

func ProcessTransactions(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	// Initialize a queue for each client
	interval := cfg.Polling["interval"]
	for _, endpoint := range cfg.Endpoints {
		go processEndpointQueue(ctx, &endpoint, srvc, interval)
	}
}

func processEndpointQueue(ctx context.Context, endpoint *config.Endpoint, srvc *service.Service, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	storage := storage.NewClientStorage(endpoint.Name, srvc.Redis, srvc.Logger)
	queue := utils.RedisStreamKey(endpoint.Name)

	// Launch queue monitor
	go monitorQueueSize(ctx, srvc.Redis, srvc.Logger, queue, interval)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			//Capture timestamp at the top level
			currentTime := time.Now().Unix()

			txString, err := srvc.Redis.LPop(ctx, queue).Result()
			if err == redis.Nil {
				time.Sleep(interval)
				continue
			} else if err != nil {
				time.Sleep(interval)
				srvc.Logger.Error(fmt.Sprintf("Error reading queued txs: %s", err), logger.Fields{"queue": queue})
				continue
			}

			tx := strings.Split(txString, ":")
			if len(tx) < 2 {
				srvc.Logger.Warn(fmt.Sprintf("Invalid transaction format: %s", txString))
				continue
			}

			processTransaction(ctx, tx[1], endpoint, srvc, storage, currentTime)

			srvc.Logger.Info(fmt.Sprintf("Processed. Client: %s, TxHash: %s", tx[0], tx[1]))
		}
	}
}

func processTransaction(ctx context.Context, txHash string, endpoint *config.Endpoint, srvc *service.Service, storage *storage.ClientStorage, time int64) {
	// Pull the TX receipts
	tx, isPending, err := endpoint.Client.TransactionByHash(ctx, common.HexToHash(txHash))

	// Handle transaction not found
	if err == ethereum.NotFound {
		if err := storage.UpdateTransaction(ctx, tx, endpoint.Name, model.StatusDropped, time); err != nil {
			srvc.Logger.Error("Error updating dropped transaction", logger.Fields{
				"txHash": txHash,
				"error":  err.Error(),
			})
		}
		return
	}

	// Handle other fetch errors
	if err != nil {
		srvc.Logger.Error("Error fetching transaction", logger.Fields{
			"txHash": txHash,
			"error":  err.Error(),
		})
		return
	}

	// Handle queued transactions
	if !isPending {
		if err := storage.UpdateTransaction(ctx, tx, endpoint.Name, model.StatusQueued, time); err != nil {
			srvc.Logger.Error("Error updating queued transaction", logger.Fields{
				"txHash": txHash,
				"error":  err.Error(),
			})
			return
		}
		// Requeue for further processing
		if err := srvc.Redis.RPush(ctx, fmt.Sprintf("stream:%s", endpoint.Name),
			fmt.Sprintf("%s:%s", endpoint.Name, txHash)).Err(); err != nil {
			srvc.Logger.Error("Error requeueing transaction", logger.Fields{
				"txHash": txHash,
				"error":  err.Error(),
			})
		}
		return
	}

	// Handle pending transactions
	if err := storage.UpdateTransaction(ctx, tx, endpoint.Name, model.StatusPending, time); err != nil {
		srvc.Logger.Error("Error updating pending transaction", logger.Fields{
			"txHash": txHash,
			"error":  err.Error(),
		})
		return
	}
}

func monitorQueueSize(ctx context.Context, redis *redis.Client, logger logger.Logger, queue string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := redis.LLen(ctx, queue).Result()
			if err != nil {
				logger.Error(fmt.Sprintf("Error getting queue length: %s", err))
				continue
			}
			logger.Info(fmt.Sprintf("Current queue size: %d", count))
		case <-ctx.Done():
			return
		}
	}
}
