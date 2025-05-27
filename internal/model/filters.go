package model

import (
	"math/big"
)

// FilterCriteria represents the filtering options
type FilterCriteria struct {
	GasPriceRange struct {
		Min *big.Int `json:"min,omitempty"`
		Max *big.Int `json:"max,omitempty"`
	} `json:"gas_price_range"`
	NonceRange struct {
		Min uint64 `json:"min,omitempty"`
		Max uint64 `json:"max,omitempty"`
	} `json:"nonce_range"`
	AddressPatterns struct {
		From []string `json:"from,omitempty"`
		To   []string `json:"to,omitempty"`
	} `json:"address_patterns"`
	Types []TransactionType `json:"types,omitempty"`
}

// GroupingCriteria represents the grouping options
type GroupingCriteria struct {
	GroupByGasPrice   bool `json:"group_by_gas_price"`
	GroupByNonceRange bool `json:"group_by_nonce_range"`
	GroupByAddress    bool `json:"group_by_address"`
	GroupByType       bool `json:"group_by_type"`
	GasPriceRanges    []struct {
		Min *big.Int `json:"min"`
		Max *big.Int `json:"max"`
	} `json:"gas_price_ranges,omitempty"`
	NonceRanges []struct {
		Min uint64 `json:"min"`
		Max uint64 `json:"max"`
	} `json:"nonce_ranges,omitempty"`
}

// StoredTransaction represents a transaction with its metadata
type StoredTransaction struct {
	Hash     string              `json:"hash"`
	Tx       Tx                  `json:"tx"`
	Metadata TransactionMetadata `json:"metadata"`
}

// GroupedTransactions represents the result of grouping transactions
type GroupedTransactions struct {
	Groups map[string][]StoredTransaction `json:"groups"`
	Stats  struct {
		TotalTransactions int64 `json:"total_transactions"`
		GroupCount        int   `json:"group_count"`
	} `json:"stats"`
}

type CountArgs struct {
	TxCount int64 `json:"tx_count" binding:"required"`
}
