package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

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
		TxKey:     utils.RedisClientTxKey(client),
		StreamKey: utils.RedisStreamKey(client),
	}
}

// StoreTransaction stores a transaction with its metadata in the specified queue
func (s *ClientStorage) StoreTransaction(ctx context.Context, txHash string) error {
	// Store entry in Redis queue hash. We have no TX details at this point
	if err := s.rdb.HSet(ctx, s.TxKey, txHash, "").Err(); err != nil {
		return fmt.Errorf("error creating transaction entry in queue %s: %w", s.TxKey, err)
	}

	// Store metadata separately for efficient filtering in per client hash txpool:geth:meta { txHash: Metadata }
	txMetaData := &model.StoredTransaction{
		Hash: txHash,
		Metadata: model.TransactionMetadata{
			Status: model.StatusReceived,
		},
	}

	txJsonMetaData, err := json.Marshal(txMetaData)

	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// Store metadata entry
	err = s.rdb.HSet(ctx, s.MetaKey, txHash, txJsonMetaData).Err()

	if err != nil {
		return fmt.Errorf("error creating metadata entry txHash:%s, error: %s", txHash, err.Error())
	}

	return nil
}

// Update StoredTransaction with Receipt Details
func (s *ClientStorage) UpdateTransaction(
	ctx context.Context,
	tx *types.Transaction,
	clientTxQueue string,
	status model.TransactionStatus,
	timestamp int64,
) error {
	txHash := tx.Hash().Hex()

	// Attempt to get transaction metadata
	val, err := s.rdb.HGet(ctx, s.MetaKey, txHash).Result()
	if err == redis.Nil {
		// Retry logic: only requeue up to 5 times
		if err := s.shouldRequeueTx(ctx, tx); err != nil {
			fmt.Println("Requeue")
			return err
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get transaction metadata from %s: %w", s.MetaKey, err)
	}

	// Unmarshal metadata
	var meta map[string]any
	if err := json.Unmarshal([]byte(val), &meta); err != nil {
		return fmt.Errorf("failed to unmarshal transaction metadata: %w", err)
	}

	// Recover sender
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		return fmt.Errorf("failed to recover transaction sender: %w", err)
	}

	// Delegate update based on transaction status
	switch status {
	case model.StatusQueued:
		err = s.updateTransactionQueued(ctx, tx, txHash, meta, sender, timestamp)
	case model.StatusPending:
		err = s.updateTransactionPending(ctx, tx, txHash, meta, sender, timestamp)
	case model.StatusDropped:
		err = s.updateTransactionDropped(ctx, txHash, meta, timestamp)
	case model.StatusMined:
		err = s.updateTransactionMined(ctx, txHash, meta, timestamp)
	default:
		err = fmt.Errorf("unknown transaction status: %v", status)
	}

	return err
}

// Requeue logic with retry limit
func (s *ClientStorage) shouldRequeueTx(ctx context.Context, tx *types.Transaction) error {
	const maxRetries = 5

	txHash := tx.Hash().Hex()
	retryKey := fmt.Sprintf("retry:%s", txHash)

	// Get current retry count
	retryStr, err := s.rdb.Get(ctx, retryKey).Result()
	retries := 0
	if err == nil {
		retries, _ = strconv.Atoi(retryStr)
	}

	if retries < maxRetries {
		// Increment and requeue
		if err := s.rdb.Set(ctx, retryKey, retries+1, 15*time.Minute).Err(); err != nil {
			return fmt.Errorf("failed to set retry count: %w", err)
		}
		if err := s.rdb.RPush(ctx, s.StreamKey, fmt.Sprintf("%s:%s", s.client, txHash)).Err(); err != nil {
			return fmt.Errorf("failed to requeue transaction: %w", err)
		}
	} else {
		log.Printf("Dropping tx %s after %d retries", txHash, retries)
	}

	return nil
}

func (s *ClientStorage) updateTransactionPending(ctx context.Context, tx *types.Transaction, txKey string, meta map[string]any, sender common.Address, time int64) error {
	meta["status"] = model.StatusPending
	meta["type"] = utils.GetTransactionType(tx)
	meta["nonce"] = tx.Nonce()
	meta["from"] = sender.Hex()
	if tx.To() != nil {
		meta["to"] = tx.To().Hex()
	}
	meta["time_pending"] = time
	meta["time_received"] = tx.Time()

	// Convert meta to JSON string
	metaJSON, err := json.Marshal(meta)


	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// Unmarshal into StoredTransaction
	var storedTx model.StoredTransaction
	if err := json.Unmarshal(metaJSON, &storedTx); err != nil {
		return fmt.Errorf("error unmarshaling metadata: %w", err)
	}

	// Index
	s.addToIndexes(ctx, &storedTx)

	return s.saveMetadata(ctx, txKey, meta)
}

func (s *ClientStorage) updateTransactionMined(ctx context.Context, txKey string, meta map[string]any, time int64) error {
	meta["status"] = model.StatusMined
	meta["time_mined"] = time

	return s.saveMetadata(ctx, txKey, meta)
}

func (s *ClientStorage) updateTransactionDropped(ctx context.Context, txKey string, meta map[string]any, time int64) error {
	meta["status"] = model.StatusDropped
	meta["time_failed"] = time

	return s.saveMetadata(ctx, txKey, meta)
}

func (s *ClientStorage) updateTransactionQueued(ctx context.Context, tx *types.Transaction, txKey string, meta map[string]any, sender common.Address, time int64) error {
	meta["status"] = model.StatusQueued
	meta["type"] = utils.GetTransactionType(tx)
	meta["nonce"] = tx.Nonce()
	meta["from"] = sender.Hex()
	if tx.To() != nil {
		meta["to"] = tx.To().Hex()
	}
	meta["time_queued"] = time
	meta["time_received"] = tx.Time()

	// Convert meta to JSON string
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// Unmarshal into StoredTransaction
	var storedTx model.StoredTransaction
	if err := json.Unmarshal(metaJSON, &storedTx); err != nil {
		return fmt.Errorf("error unmarshaling metadata: %w", err)
	}

	// Index
	s.addToIndexes(ctx, &storedTx)

	return s.saveMetadata(ctx, txKey, meta)
}

func (s *ClientStorage) saveMetadata(ctx context.Context, txKey string, meta map[string]any) error {
	updated, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	err = s.rdb.HSet(ctx, s.MetaKey, txKey, updated).Err()
	if err == redis.Nil {
	} else if err != nil {
		return fmt.Errorf("error saving metadata: %w", err)
	}

	return nil
}

// addToIndexes adds the transaction to various indexes for efficient filtering
func (s *ClientStorage) addToIndexes(ctx context.Context, tx *model.StoredTransaction) {
	pipe := s.rdb.Pipeline()
	txKey := tx.Hash
	client := s.client

	// Index by gas price
	if tx.Metadata.GasPrice != nil {
		pipe.ZAdd(ctx, utils.RedisGasIndexKey(client), redis.Z{
			Score:  float64(tx.Metadata.GasPrice.Int64()),
			Member: txKey,
		})
	}

	// Index by nonce
	pipe.ZAdd(ctx, utils.RedisNonceIndexKey(client), redis.Z{
		Score:  float64(tx.Metadata.Nonce),
		Member: txKey,
	})

	// Index by type
	pipe.ZAdd(ctx, utils.RedisTypeIndexKey(client), redis.Z{
		Score:  float64(tx.Metadata.Type),
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
		if tx.Metadata.GasPrice != nil && tx.Metadata.GasPrice.Cmp(criteria.GasPriceRange.Min) < 0 {
			return false
		}
	}
	if criteria.GasPriceRange.Max != nil {
		if tx.Metadata.GasPrice != nil && tx.Metadata.GasPrice.Cmp(criteria.GasPriceRange.Max) > 0 {
			return false
		}
	}

	// Check nonce range
	if criteria.NonceRange.Min > 0 && tx.Metadata.Nonce < criteria.NonceRange.Min {
		return false
	}
	if criteria.NonceRange.Max > 0 && tx.Metadata.Nonce > criteria.NonceRange.Max {
		return false
	}

	// Check address patterns
	if len(criteria.AddressPatterns.From) > 0 {
		matched := false
		for _, pattern := range criteria.AddressPatterns.From {
			if matched, _ = regexp.MatchString(pattern, tx.Metadata.From); matched {
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
			if matched, _ = regexp.MatchString(pattern, tx.Metadata.To); matched {
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check transaction types
	if len(criteria.Types) > 0 {
		matched := false
		for _, t := range criteria.Types {
			if tx.Metadata.Type == t {
				matched = true
				break
			}
		}
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
		if tx.Metadata.GasPrice != nil {
			parts = append(parts, fmt.Sprintf("gas:%d", tx.Metadata.GasPrice.Int64()))
		}
	}

	if criteria.GroupByNonceRange {
		parts = append(parts, fmt.Sprintf("nonce:%d", tx.Metadata.Nonce))
	}

	if criteria.GroupByAddress {
		parts = append(parts, fmt.Sprintf("from:%s", tx.Metadata.From))
		if tx.Metadata.To != "" {
			parts = append(parts, fmt.Sprintf("to:%s", tx.Metadata.To))
		}
	}

	if criteria.GroupByType {
		parts = append(parts, fmt.Sprintf("type:%d", tx.Metadata.Type))
	}

	if len(parts) == 0 {
		return "default"
	}

	return strings.Join(parts, "|")
}
