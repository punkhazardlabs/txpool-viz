package storage

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

// Storage handles all Redis operations for transactions
type ClientStorage struct {
	rdb    *redis.Client
	logger logger.Logger
	client string

	// Redis key references
	MetaKey   string
	TxKey     string
	StreamKey string
}

// NewStorage creates a new storage instance
func NewClientStorage(client string, rdb *redis.Client, l logger.Logger) *ClientStorage {
	return &ClientStorage{
		rdb:       rdb,
		logger:    l,
		client:    client,
		MetaKey:   utils.RedisClientMetaKey(client),
		StreamKey: utils.RedisStreamKey(client),
	}
}

// StoreTransaction stores a transaction with its metadata in the specified per-client queue
func (s *ClientStorage) StoreTransaction(ctx context.Context, txHash string, localDetectionTime int64) error {
	// 1. Store metadata and tx data separately for efficient filtering in per client hash txpool:geth:meta { txHash: StoredTx: {Tx, TxMetadata}}
	txMetaData := &model.StoredTransaction{
		Hash: txHash,
		Metadata: model.TransactionMetadata{
			Status:       model.StatusReceived,
			TimeReceived: localDetectionTime,
		},
	}

	// Serialize metadata to JSON
	txJsonMetaData, err := json.Marshal(txMetaData)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// 3. Store metadata entry in Redis
	err = s.rdb.HSet(ctx, s.MetaKey, txHash, txJsonMetaData).Err()
	if err != nil {
		return fmt.Errorf("error creating metadata entry txHash:%s, error: %s", txHash, err.Error())
	}

	return nil
}

func (s *ClientStorage) updateStoredTx(ctx context.Context, txHash string, updateFn func(*model.StoredTransaction) error) error {
	val, err := s.rdb.HGet(ctx, s.MetaKey, txHash).Result()
	if err == redis.Nil {
		s.logger.Debug("Transaction metadata not found", "txHash", txHash)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get transaction metadata from %s: %w", s.MetaKey, err)
	}

	var storedTx model.StoredTransaction
	if err := json.Unmarshal([]byte(val), &storedTx); err != nil {
		return fmt.Errorf("failed to unmarshal transaction metadata: %w", err)
	}

	if err := updateFn(&storedTx); err != nil {
		return err
	}

	updated, err := json.Marshal(storedTx)
	if err != nil {
		return fmt.Errorf("error marshaling updated metadata: %w", err)
	}

	if err := s.rdb.HSet(ctx, s.MetaKey, txHash, updated).Err(); err != nil {
		return fmt.Errorf("error saving updated metadata to Redis: %w", err)
	}

	s.addToIndexes(ctx, &storedTx)

	return nil
}

func (s *ClientStorage) UpdateMinedTransaction(ctx context.Context, txHash string, tx *types.Transaction, blockTimestamp int64, receiptStatus uint64, blockNumber *big.Int, blockHash *common.Hash) error {
	return s.updateStoredTx(ctx, txHash, func(storedTx *model.StoredTransaction) error {
		storedTx.Metadata.Status = model.StatusMined
		storedTx.Metadata.TimeMined = &blockTimestamp

		mineStatus := model.MinedTxStatus(receiptStatus)
		storedTx.Metadata.MineStatus = mineStatus.String()

		if storedTx.Metadata.TimePending == nil {
			storedTx.Metadata.TimePending = &blockTimestamp
		}

		if tx != nil {
			sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			if err != nil {
				return fmt.Errorf("failed to derive sender: %w", err)
			}
			storedTx.Tx = structureTx(tx, sender)
		}

		if blockNumber != nil {
			storedTx.Metadata.BlockNumber = blockNumber.Uint64()
		}
		if blockHash != nil {
			storedTx.Metadata.BlockHash = blockHash.Hex()
		}

		return nil
	})
}

func (s *ClientStorage) UpdatePendingTransaction(ctx context.Context, txHash string, tx *types.Transaction, timestamp int64) error {
	return s.updateStoredTx(ctx, txHash, func(storedTx *model.StoredTransaction) error {
		storedTx.Metadata.Status = model.StatusPending

		localDetectionTime := timestamp
		storedTx.Metadata.TimePending = &localDetectionTime

		if tx != nil {
			sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			if err != nil {
				return fmt.Errorf("failed to derive sender: %w", err)
			}
			storedTx.Tx = structureTx(tx, sender)
		}
		return nil
	})
}

func (s *ClientStorage) UpdateDroppedTransaction(ctx context.Context, txHash string, timestamp int64) error {
	return s.updateStoredTx(ctx, txHash, func(storedTx *model.StoredTransaction) error {
		storedTx.Metadata.Status = model.StatusDropped
		storedTx.Metadata.TimeDropped = timestamp
		return nil
	})
}

func (s *ClientStorage) UpdateQueuedTransaction(ctx context.Context, txHash string, tx *types.Transaction, timestamp int64) error {
	return s.updateStoredTx(ctx, txHash, func(storedTx *model.StoredTransaction) error {
		storedTx.Metadata.Status = model.StatusQueued

		if tx != nil {
			sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			if err != nil {
				return fmt.Errorf("failed to derive sender: %w", err)
			}
			storedTx.Tx = structureTx(tx, sender)
		}

		return nil
	})
}

// extracts the core fields from types.Transaction into model.Tx
func structureTx(tx *types.Transaction, sender common.Address) model.Tx {
	isContractCreation := tx.To() == nil

	txData := model.Tx{
		ChainID:             tx.ChainId().String(),
		From:                sender.Hex(),
		Nonce:               tx.Nonce(),
		Value:               tx.Value().String(),
		Gas:                 tx.Gas(),
		GasPrice:            tx.GasPrice(),
		MaxFeePerGas:        tx.GasFeeCap().String(),
		MaxPriorityFee:      tx.GasTipCap().String(),
		MaxFeePerBlobGas:    "",
		Data:                hex.EncodeToString(tx.Data()),
		Type:                tx.Type(),
		IsContractCreation:  isContractCreation,
	}

	if tx.To() != nil {
		txData.To = tx.To().Hex()
	}

	return txData
}


// addToIndexes adds the transaction to various indexes for efficient filtering
func (s *ClientStorage) addToIndexes(ctx context.Context, tx *model.StoredTransaction) {
	pipe := s.rdb.Pipeline()
	txKey := tx.Hash
	client := s.client

	// Index by gas price
	if tx.Tx.GasPrice != nil {
		pipe.ZAdd(ctx, utils.RedisGasIndexKey(client), redis.Z{
			Score:  float64(tx.Tx.GasPrice.Int64()),
			Member: txKey,
		})
	}

	// Index by nonce
	pipe.ZAdd(ctx, utils.RedisNonceIndexKey(client), redis.Z{
		Score:  float64(tx.Tx.Nonce),
		Member: txKey,
	})

	// Index by type
	pipe.ZAdd(ctx, utils.RedisTypeIndexKey(client), redis.Z{
		Score:  float64(tx.Tx.Type),
		Member: txKey,
	})

	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Error(fmt.Sprintf("Error adding transaction to indexes: %s", err))
	}
}

// FilterTransactions retrieves transactions based on filter criteria from a specific queue
func (s *ClientStorage) FilterTransactions(ctx context.Context, queue string, criteria model.FilterCriteria) ([]model.StoredTransaction, error) {
	// Get all transactions from the queue
	txMap, err := s.rdb.HGetAll(ctx, queue).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting transactions from queue %s: %w", queue, err)
	}

	var results []model.StoredTransaction
	for _, txData := range txMap {
		var tx model.StoredTransaction
		if err := json.Unmarshal([]byte(txData), &tx); err != nil {
			continue
		}

		// Apply filters
		if s.matchesFilter(tx, criteria) {
			results = append(results, tx)
		}
	}

	return results, nil
}

// matchesFilter checks if a transaction matches the filter criteria
func (s *ClientStorage) matchesFilter(tx model.StoredTransaction, criteria model.FilterCriteria) bool {
	// Check gas price range
	if criteria.GasPriceRange.Min != nil {
		if tx.Tx.GasPrice != nil && tx.Tx.GasPrice.Cmp(criteria.GasPriceRange.Min) < 0 {
			return false
		}
	}
	if criteria.GasPriceRange.Max != nil {
		if tx.Tx.GasPrice != nil && tx.Tx.GasPrice.Cmp(criteria.GasPriceRange.Max) > 0 {
			return false
		}
	}

	// Check nonce range
	if criteria.NonceRange.Min > 0 && tx.Tx.Nonce < criteria.NonceRange.Min {
		return false
	}
	if criteria.NonceRange.Max > 0 && tx.Tx.Nonce > criteria.NonceRange.Max {
		return false
	}

	// Check address patterns
	if len(criteria.AddressPatterns.From) > 0 {
		matched := false
		for _, pattern := range criteria.AddressPatterns.From {
			if matched, _ = regexp.MatchString(pattern, tx.Tx.From); matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(criteria.AddressPatterns.To) > 0 {
		matched := false
		for _, pattern := range criteria.AddressPatterns.To {
			if matched, _ = regexp.MatchString(pattern, tx.Tx.To); matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check transaction types
	if len(criteria.Types) > 0 {
		matched := slices.Contains(criteria.Types, model.TransactionType(tx.Tx.Type))
		if !matched {
			return false
		}
	}

	return true
}

// GroupTransactions groups transactions based on grouping criteria from a specific queue
func (s *ClientStorage) GroupTransactions(ctx context.Context, queue string, criteria model.GroupingCriteria) (*model.GroupedTransactions, error) {
	// Get all transactions from the queue
	txMap, err := s.rdb.HGetAll(ctx, queue).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting transactions from queue %s: %w", queue, err)
	}

	groups := make(map[string][]model.StoredTransaction)
	var totalTxs int64

	for _, txData := range txMap {
		var tx model.StoredTransaction
		if err := json.Unmarshal([]byte(txData), &tx); err != nil {
			continue
		}

		groupKey := s.getGroupKey(tx, criteria)
		groups[groupKey] = append(groups[groupKey], tx)
		totalTxs++
	}

	return &model.GroupedTransactions{
		Groups: groups,
		Stats: struct {
			TotalTransactions int64 `json:"total_transactions"`
			GroupCount        int   `json:"group_count"`
		}{
			TotalTransactions: totalTxs,
			GroupCount:        len(groups),
		},
	}, nil
}

// getGroupKey generates a key for grouping transactions
func (s *ClientStorage) getGroupKey(tx model.StoredTransaction, criteria model.GroupingCriteria) string {
	var parts []string

	if criteria.GroupByGasPrice {
		if tx.Tx.GasPrice != nil {
			parts = append(parts, fmt.Sprintf("gas:%d", tx.Tx.GasPrice.Int64()))
		}
	}

	if criteria.GroupByNonceRange {
		parts = append(parts, fmt.Sprintf("nonce:%d", tx.Tx.Nonce))
	}

	if criteria.GroupByAddress {
		parts = append(parts, fmt.Sprintf("from:%s", tx.Tx.From))
		if tx.Tx.To != "" {
			parts = append(parts, fmt.Sprintf("to:%s", tx.Tx.To))
		}
	}

	if criteria.GroupByType {
		parts = append(parts, fmt.Sprintf("type:%d", tx.Tx.Type))
	}

	if len(parts) == 0 {
		return "default"
	}

	return strings.Join(parts, "|")
}
