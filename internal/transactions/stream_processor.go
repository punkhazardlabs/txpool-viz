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

const (
	notIndexedError = "transaction indexing is in progress"
)

func ProcessTransactions(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	// Initialize a queue for each client
	interval := cfg.Polling.Interval
	for _, endpoint := range cfg.Endpoints {
		go processEndpointQueue(ctx, &endpoint, srvc, interval)
	}
}

func processEndpointQueue(ctx context.Context, endpoint *config.Endpoint, srvc *service.Service, intervalString string) {
	interval, err := time.ParseDuration(intervalString)

	if err != nil {
		srvc.Logger.Error("Error parsing endpoint interval", "error", err.Error())
	}

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

		case <-ticker.C:
			currentTime := time.Now().Unix()

			txString, err := srvc.Redis.LPop(ctx, queue).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				srvc.Logger.Error(fmt.Sprintf("Error reading queued txs: %s", err), logger.Fields{"queue": queue})
				continue
			}

			tx := strings.Split(txString, ":")
			if len(tx) < 2 {
				srvc.Logger.Warn(fmt.Sprintf("Invalid transaction format: %s", txString))
				continue
			}

			go processTransaction(ctx, tx[1], endpoint, srvc, storage, currentTime)
		}
	}
}

func processTransaction(
	ctx context.Context,
	txHash string,
	endpoint *config.Endpoint,
	srvc *service.Service,
	storage *storage.ClientStorage,
	timestamp int64,
) {
	streamKey := utils.RedisStreamKey(endpoint.Name)
	l := srvc.Logger

	// Check for transaction receipt — if exists, it's mined
	receipt, err := endpoint.Client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err == nil {
		l.Debug("Transaction is mined", logger.Fields{
			"txHash":      txHash,
			"blockNumber": receipt.BlockNumber,
			"status":      model.MinedTxStatus(receipt.Status).String(),
			"endpoint":    endpoint.Name,
		})

		// fetch tx details
		tx, _, txErr := endpoint.Client.TransactionByHash(ctx, receipt.TxHash)
		if txErr != nil {
			l.Error("Error fetching mined transaction details", logger.Fields{"txHash": txHash, "error": txErr.Error()})
			return
		}

		block, err := endpoint.Client.BlockByNumber(ctx, receipt.BlockNumber)
		if err != nil {
			l.Error("Error fetching block details", logger.Fields{"txHash": txHash, "error": err.Error()})
			return
		}

		blocktimestamp := block.Time()

		if err := storage.UpdateTransaction(ctx, txHash, tx, model.StatusMined, int64(blocktimestamp), &receipt.Status); err != nil {
			l.Error("Error updating mined transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
		}
		return
	} else if err.Error() == notIndexedError {
		l.Debug("Transaction receipt not indexed yet", logger.Fields{
			"txHash": txHash,
			"endpoint": endpoint.Name,
		})
		// Requeue
		if err := srvc.Redis.RPush(ctx, streamKey, fmt.Sprintf("%s:%s", endpoint.Name, txHash)).Err(); err != nil {
			l.Error("Error requeuing transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
		}

		return
	}

	// No receipt — check if it's still in mempool
	tx, isPending, err := endpoint.Client.TransactionByHash(ctx, common.HexToHash(txHash))
	if err == ethereum.NotFound {
		// Not in mempool — it's dropped
		l.Debug("Transaction dropped", logger.Fields{"txHash": txHash})
		if err := storage.UpdateTransaction(ctx, txHash, nil, model.StatusDropped, timestamp, nil); err != nil {
			l.Error("Error updating dropped transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
		}
		return
	}

	if err != nil {
		l.Error("Error fetching transaction from mempool", logger.Fields{"txHash": txHash, "error": err.Error()})
		return
	}

	// If in mempool and pending
	if isPending {
		l.Debug("Transaction is pending", logger.Fields{"txHash": txHash, "endpoint": endpoint.Name})
		if err := storage.UpdateTransaction(ctx, txHash, tx, model.StatusPending, timestamp, nil); err != nil {
			l.Error("Error updating pending transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
		}
	} else {
		// It's queued — waiting for future block (nonce/gas)
		l.Debug("Transaction is queued", logger.Fields{"txHash": txHash})
		if err := storage.UpdateTransaction(ctx, txHash, tx, model.StatusQueued, timestamp, nil); err != nil {
			l.Error("Error updating queued transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
		}
	}

	// Requeue for future check
	if err := srvc.Redis.RPush(ctx, streamKey, fmt.Sprintf("%s:%s", endpoint.Name, txHash)).Err(); err != nil {
		l.Error("Error requeuing transaction", logger.Fields{"txHash": txHash, "error": err.Error()})
	}
}

func monitorQueueSize(ctx context.Context, redis *redis.Client, l logger.Logger, queue string, interval time.Duration) {
	ticker := time.NewTicker(time.Duration(interval) * 5) // Adjust the interval as needed later. Maybe add to config.yaml
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := redis.LLen(ctx, queue).Result()
			if err != nil {
				l.Warn(fmt.Sprintf("Error getting queue length: %s", err.Error()))
				continue
			}
			l.Info("Queue size checked", logger.Fields{"queue": queue, "size": count})
		case <-ctx.Done():
			return
		}
	}
}
