package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	prefixTx        = "tx:"     // For storing transaction data
	prefixIndex     = "idx:"    // For storing indexes
	prefixGroup     = "group:"  // For storing grouped transactions
	prefixStats     = "stats:"  // For storing statistics
	prefixFilter    = "filter:" // For storing filter results
	prefixMetadata  = "meta:"   // For storing transaction metadata
	prefixAddresses = "addr:"   // For storing address-related data
)

// Storage handles all Redis operations for transactions
type Storage struct {
	rdb *redis.Client
}

// NewStorage creates a new storage instance
func NewStorage(rdb *redis.Client) *Storage {
	return &Storage{rdb: rdb}
}

// StoreTransaction stores a transaction with its metadata
func (s *Storage) StoreTransaction(ctx context.Context, tx *StoredTransaction) error {
	// Store the full transaction data
	txKey := fmt.Sprintf("%s%s:%d", prefixTx, tx.Metadata.From, tx.Metadata.Nonce)
	txData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("error marshaling transaction: %w", err)
	}

	// Store metadata separately for efficient filtering
	metaKey := fmt.Sprintf("%s%s:%d", prefixMetadata, tx.Metadata.From, tx.Metadata.Nonce)
	metaData, err := json.Marshal(tx.Metadata)
	if err != nil {
		return fmt.Errorf("error marshaling metadata: %w", err)
	}

	// Store in Redis
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, txKey, txData, 24*time.Hour)
	pipe.Set(ctx, metaKey, metaData, 24*time.Hour)

	// Add to indexes
	s.addToIndexes(ctx, pipe, tx)

	_, err = pipe.Exec(ctx)
	return err
}

// addToIndexes adds the transaction to various indexes for efficient filtering
func (s *Storage) addToIndexes(ctx context.Context, pipe redis.Pipeliner, tx *StoredTransaction) {
	// Index by gas price
	if tx.Metadata.GasPrice != nil {
		pipe.ZAdd(ctx, fmt.Sprintf("%sgas_price", prefixIndex), redis.Z{
			Score:  float64(tx.Metadata.GasPrice.Int64()),
			Member: fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce),
		})
	}

	// Index by nonce
	pipe.ZAdd(ctx, fmt.Sprintf("%snonce", prefixIndex), redis.Z{
		Score:  float64(tx.Metadata.Nonce),
		Member: fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce),
	})

	// Index by type
	pipe.SAdd(ctx, fmt.Sprintf("%stype:%d", prefixIndex, tx.Metadata.Type),
		fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce))

	// Index by address
	pipe.SAdd(ctx, fmt.Sprintf("%sfrom:%s", prefixIndex, tx.Metadata.From),
		fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce))
	if tx.Metadata.To != "" {
		pipe.SAdd(ctx, fmt.Sprintf("%sto:%s", prefixIndex, tx.Metadata.To),
			fmt.Sprintf("%s:%d", tx.Metadata.From, tx.Metadata.Nonce))
	}
}

// FilterTransactions retrieves transactions based on filter criteria
func (s *Storage) FilterTransactions(ctx context.Context, criteria FilterCriteria) ([]StoredTransaction, error) {
	// Start with all transactions
	baseKey := fmt.Sprintf("%s*", prefixTx)
	iter := s.rdb.Scan(ctx, 0, baseKey, 0).Iterator()

	var results []StoredTransaction
	for iter.Next(ctx) {
		key := iter.Val()
		if !strings.HasPrefix(key, prefixTx) {
			continue
		}

		// Get transaction data
		txData, err := s.rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var tx StoredTransaction
		if err := json.Unmarshal(txData, &tx); err != nil {
			continue
		}

		// Apply filters
		if s.matchesFilter(tx, criteria) {
			results = append(results, tx)
		}
	}

	return results, iter.Err()
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

// GroupTransactions groups transactions based on grouping criteria
func (s *Storage) GroupTransactions(ctx context.Context, criteria GroupingCriteria) (*GroupedTransactions, error) {
	// Get all transactions
	txs, err := s.FilterTransactions(ctx, FilterCriteria{})
	if err != nil {
		return nil, err
	}

	groups := make(map[string][]StoredTransaction)
	var totalTxs int64

	for _, tx := range txs {
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
