package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/service"
	"txpool-viz/internal/storage"
	"txpool-viz/pkg"

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

	storage := storage.NewStorage(srvc.Redis, srvc.Logger)
	queue := fmt.Sprintf("stream:%s", endpoint.Name)

	// Launch queue monitor
	go monitorQueueSize(ctx, srvc.Redis, srvc.Logger, queue, interval)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			txString, err := srvc.Redis.LPop(ctx, queue).Result()
			if err == redis.Nil {
				time.Sleep(interval)
				continue
			} else if err != nil {
				time.Sleep(interval)
				srvc.Logger.Error(fmt.Sprintf("Error reading queued txs: %s", err), pkg.Fields{"queue": queue})
				continue
			}

			tx := strings.Split(txString, ":")
			if len(tx) < 2 {
				srvc.Logger.Warn(fmt.Sprintf("Invalid transaction format: %s", txString))
				continue
			}

			processTransactions(ctx, tx[1], endpoint, srvc, storage)

			srvc.Logger.Info(fmt.Sprintf("Processed. Client: %s, TxHash: %s", tx[0], tx[1]))
		}
	}
}

func processTransactions(ctx context.Context, txHash string, endpoint *config.Endpoint, srvc *service.Service, storage *storage.Storage) {
	// Pull the TX receipts
	tx, _, err := endpoint.Client.TransactionByHash(ctx, common.HexToHash(txHash))

	// @ndeto @TODO If not pending, queue it up to keep polling
	// if !isPending {}

	if err != nil {
		srvc.Logger.Error(fmt.Sprintf("Error getting TX details. Err: %s", err))
	}

	time := time.Now().Unix()

	err = storage.UpdateTransaction(ctx, tx, endpoint.Name, time)

	if err != nil {
		srvc.Logger.Error("Error Updating TX", pkg.Fields{"txHash": txHash, "tx": tx})
	}
}

func monitorQueueSize(ctx context.Context, redis *redis.Client, logger pkg.Logger, queue string, interval time.Duration) {
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
