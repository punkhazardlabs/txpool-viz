package service

import (
	"context"
	"encoding/json"
	"txpool-viz/internal/config"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"github.com/redis/go-redis/v9"
	"sync"
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

func (ts *TransactionServiceImpl) GetLatestNTransactions(ctx context.Context, n int64) ([]model.ApiTxResponse, error) {
	start := -n
	stop := int64(-1)

	results, err := ts.redis.ZRangeWithScores(ctx, utils.RedisUniversalKey(), start, stop).Result()
	if err != nil {
		ts.logger.Error("Redis ZRangeWithScores failed", "error", err)
		return nil, err
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	var transactions []model.ApiTxResponse
	for _, result := range results {
		wg.Add(1)
		go func(res redis.Z) {
			defer wg.Done()

			txHash := res.Member.(string)

			txDetails, err := ts.GetTxDetails(ctx, txHash)
			if err != nil {
				ts.logger.Error("Couldn't get tx details", "txHash", txHash, "error", err)
				return
			}

			mu.Lock()
			transactions = append(transactions, txDetails)
			mu.Unlock()
		}(result)
	}

	wg.Wait()
	return transactions, nil
}

func (ts *TransactionServiceImpl) GetLatestTxSummaries(ctx context.Context, n int64) ([]model.TxSummary, error) {
	txs, err := ts.GetLatestNTransactions(ctx, n)
	if err != nil {
		return nil, err
	}

	primary := ts.endpoints[0].Name
	metaKey := utils.RedisClientMetaKey(primary)

	var out []model.TxSummary
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, tx := range txs {
		wg.Add(1)
		go func(txHash string) {
			defer wg.Done()

			raw, err := ts.redis.HGet(ctx, metaKey, txHash).Result()
			if err != nil {
				return
			}
			var stx model.StoredTransaction
			if err := json.Unmarshal([]byte(raw), &stx); err != nil {
				return
			}
			tx := stx.Tx

			summary := model.TxSummary{
				Hash:    stx.Hash,
				From:    tx.From,
				GasUsed: float64(stx.Metadata.GasUsed),
				Nonce:   tx.Nonce,
				Type:    tx.String(),
			}

			mu.Lock()
			out = append(out, summary)
			mu.Unlock()
		}(tx.Hash)
	}

	wg.Wait()
	return out, nil
}

func (ts *TransactionServiceImpl) GetTxDetails(ctx context.Context, txHash string) (model.ApiTxResponse, error) {
	raw := make(map[string]model.StoredTransaction, len(ts.endpoints))

	for _, endpoint := range ts.endpoints {
		metaKey := utils.RedisClientMetaKey(endpoint.Name)

		// retrieve per client record
		val, err := ts.redis.HGet(context.Background(), metaKey, txHash).Result()
		if err != nil {
			if err == redis.Nil {
				ts.logger.Debug("No record for %s in %s\n", txHash, metaKey)
				continue
			}
			ts.logger.Error("Redis error", "error", err.Error())
			continue
		}

		// Unmarshal
		var storedTx model.StoredTransaction
		err = json.Unmarshal([]byte(val), &storedTx)
		if err != nil {
			ts.logger.Error("Failed to unmarshal transaction: %v\n", err)
			return model.ApiTxResponse{}, err
		}

		raw[endpoint.Name] = storedTx
	}

	// flatten each into maps
	txMaps := make(map[string]map[string]interface{}, len(raw))
	metaMaps := make(map[string]map[string]interface{}, len(raw))
	for client, stx := range raw {
		txMaps[client] = toMapTx(stx.Tx)
		metaMaps[client] = toMapMeta(stx.Metadata)
	}

	first := ts.endpoints[0].Name
	txRes := Compute(txMaps, first)
	metaRes := Compute(metaMaps, first)

	// assemble clients
	clients := make([]string, len(ts.endpoints))
	for i, ep := range ts.endpoints {
		clients[i] = ep.Name
	}

	resp := model.ApiTxResponse{
		Hash:    txHash,
		Clients: clients,
		Common: model.TxBlock{
			Tx:       txRes.Common,
			Metadata: metaRes.Common,
		},
		Diff: model.TxDiff{
			Tx:       txRes.Diff,
			Metadata: metaRes.Diff,
		},
	}

	return resp, nil
}
