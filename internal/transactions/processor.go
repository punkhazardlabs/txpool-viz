package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/service"
	"txpool-viz/pkg"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

	storage := NewStorage(srvc.Redis, srvc.Logger)
	queue := fmt.Sprintf("stream:%s", endpoint.Name)

	go func() {
		for {
			select {
			case <-ticker.C:
				count, err := srvc.Redis.LLen(ctx, queue).Result()
				if err != nil {
					srvc.Logger.Error(fmt.Sprintf("Error getting queue length: %s", err))
					continue
				}
				srvc.Logger.Info(fmt.Sprintf("Current queue size: %d", count))
			case <-ctx.Done():
				return
			}
		}
	}()

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

func processTransactions(ctx context.Context, txHash string, endpoint *config.Endpoint, srvc *service.Service, storage *Storage) {
	// Pull the TX receipts
	tx, _, err := endpoint.Client.TransactionByHash(ctx, common.HexToHash(txHash))

	if err != nil {
		srvc.Logger.Error(fmt.Sprintf("Error getting TX details. Err: %s", err))
	}

	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

	if err != nil {
		srvc.Logger.Error("Invalid Signature: %s", err)
	}

	// Create new stored transaction
	storedTx := &StoredTransaction{
		Hash: tx.Hash().String(),
		Metadata: TransactionMetadata{
			Status:       StatusQueued,
			TimeReceived: time.Now().Unix(),
			Type:         getTransactionType(tx),
			Nonce:        tx.Nonce(),
			From:         sender.Hex(),
			To:           tx.To().Hex(),
			Timestamp:    time.Now().Unix(),
		},
	}

	// Set gas price based on transaction type
	switch tx.Type() {
	case types.DynamicFeeTxType:
		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
	case types.BlobTxType:
		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
		storedTx.Metadata.MaxFeePerBlobGas = tx.BlobGasFeeCap()
	default:
		storedTx.Metadata.GasPrice = tx.GasPrice()
	}

	// Store to Redis
	err = storage.StoreTransaction(ctx, storedTx, endpoint.Name)

	if err == redis.Nil {
	} else if err != nil {
		srvc.Logger.Error("Error storing tx to cache", pkg.Fields{"txHash": tx.Hash(), "error": err.Error()})
	}
}

// getTransactionType determines the type of transaction
func getTransactionType(tx *types.Transaction) TransactionType {
	switch {
	case tx.Type() == types.BlobTxType:
		return BlobTx
	case tx.Type() == types.DynamicFeeTxType:
		return EIP1559Tx
	case tx.Type() == types.AccessListTxType:
		return EIP2930Tx
	default:
		return LegacyTx
	}
}
