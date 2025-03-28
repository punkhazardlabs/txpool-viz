package transactions

import (
	"math/big"
)

// TransactionType represents the type of Ethereum transaction
type TransactionType int

const (
	LegacyTx TransactionType = iota
	EIP1559Tx
	BlobTx
	EIP2930Tx
	EIP7702Tx
)

// TransactionStatus represents the current state of a transaction
type TransactionStatus int

const (
	StatusQueued TransactionStatus = iota
	StatusPending
	StatusMined
	StatusDropped
)

// TransactionMetadata contains additional metadata for filtering and grouping
type TransactionMetadata struct {
	Type             TransactionType   `json:"type"`
	GasPrice         *big.Int          `json:"gas_price,omitempty"`
	MaxFeePerGas     *big.Int          `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFee   *big.Int          `json:"max_priority_fee,omitempty"`
	MaxFeePerBlobGas *big.Int          `json:"max_fee_per_blob_gas,omitempty"`
	Nonce            uint64            `json:"nonce"`
	From             string            `json:"from"`
	To               string            `json:"to"`
	IsContract       bool              `json:"is_contract"`
	Timestamp        int64             `json:"timestamp"`
	Status           TransactionStatus `json:"status"`
	TimeReceived     int64             `json:"time_received"` // When first seen in mempool
	TimePending      int64             `json:"time_pending"`  // When moved to pending
	TimeMined        int64             `json:"time_mined"`    // When mined
	TimeDropped      int64             `json:"time_dropped"`  // When dropped
	BlockNumber      uint64            `json:"block_number"`  // If mined
	BlockHash        string            `json:"block_hash"`    // If mined
}

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
