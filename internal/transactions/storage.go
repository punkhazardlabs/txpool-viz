package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"txpool-viz/pkg"

	"github.com/redis/go-redis/v9"
)

// Storage handles all Redis operations for transactions
type Storage struct {
	rdb    *redis.Client
	logger pkg.Logger
}

// NewStorage creates a new storage instance
func NewStorage(rdb *redis.Client, l pkg.Logger) *Storage {
	return &Storage{
		rdb:    rdb,
		logger: l,
	}
}

// StoreTransaction stores a transaction with its metadata in the specified queue
func (s *Storage) StoreTransaction(ctx context.Context, tx *StoredTransaction, queue string) error {
	// Create the transaction key
	txKey := fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce)

	// Store the full transaction data in the queue hash
	txData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("error marshaling transaction: %w", err)
	}

	// Store in Redis queue hash
	if err := s.rdb.HSet(ctx, queue, txKey, txData).Err(); err != nil {
		return fmt.Errorf("error storing transaction in queue %s: %w", queue, err)
	}

	// Store metadata separately for efficient filtering
	metaKey := fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce)
	metaData, err := json.Marshal(tx.Metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// Add to indexes
	s.addToIndexes(ctx, tx, queue)

	// Store metadata
	s.rdb.HSet(ctx, fmt.Sprintf("meta:%s", queue), metaKey, metaData)

	return nil
}

// addToIndexes adds the transaction to various indexes for efficient filtering
func (s *Storage) addToIndexes(ctx context.Context, tx *StoredTransaction, queue string) {
	pipe := s.rdb.Pipeline()
	txKey := fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce)

	// Index by gas price
	if tx.Metadata.GasPrice != nil {
		pipe.ZAdd(ctx, fmt.Sprintf("%s:gas", queue), redis.Z{
			Score:  float64(tx.Metadata.GasPrice.Int64()),
			Member: txKey,
		})
	}

	// Index by nonce
	pipe.ZAdd(ctx, fmt.Sprintf("%s:nonce", queue), redis.Z{
		Score:  float64(tx.Metadata.Nonce),
		Member: txKey,
	})

	// Index by type
	pipe.ZAdd(ctx, fmt.Sprintf("%s:type", queue), redis.Z{
		Score:  float64(tx.Metadata.Type),
		Member: txKey,
	})

	_, err := pipe.Exec(ctx)

	if err != nil {
		s.logger.Error(fmt.Sprintf("Error adding transaction %s", err))
	}
}

// FilterTransactions retrieves transactions based on filter criteria from a specific queue
func (s *Storage) FilterTransactions(ctx context.Context, queue string, criteria FilterCriteria) ([]StoredTransaction, error) {
	// Get all transactions from the queue
	txMap, err := s.rdb.HGetAll(ctx, queue).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting transactions from queue %s: %w", queue, err)
	}

	var results []StoredTransaction
	for _, txData := range txMap {
		var tx StoredTransaction
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
func (s *Storage) matchesFilter(tx StoredTransaction, criteria FilterCriteria) bool {
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
func (s *Storage) GroupTransactions(ctx context.Context, queue string, criteria GroupingCriteria) (*GroupedTransactions, error) {
	// Get all transactions from the queue
	txMap, err := s.rdb.HGetAll(ctx, queue).Result()
	if err != nil {
		return nil, fmt.Errorf("error getting transactions from queue %s: %w", queue, err)
	}

	groups := make(map[string][]StoredTransaction)
	var totalTxs int64

	for _, txData := range txMap {
		var tx StoredTransaction
		if err := json.Unmarshal([]byte(txData), &tx); err != nil {
			continue
		}

		groupKey := s.getGroupKey(tx, criteria)
		groups[groupKey] = append(groups[groupKey], tx)
		totalTxs++
	}

	return &GroupedTransactions{
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
func (s *Storage) getGroupKey(tx StoredTransaction, criteria GroupingCriteria) string {
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
