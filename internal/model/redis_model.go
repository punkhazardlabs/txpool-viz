package model

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
	StatusReceived TransactionStatus = iota
	StatusPending
	StatusQueued
	StatusMined
	StatusDropped
)

// TransactionMetadata contains additional metadata for filtering and grouping
type TransactionMetadata struct {
	Status       TransactionStatus `json:"status"`        // Custom status enum
	IsContract   bool              `json:"is_contract"`   // Whether destination is contract
	TimeReceived int64             `json:"time_received"` // When seen in mempool
	TimePending  int64             `json:"time_pending"`
	TimeQueued   int64             `json:"time_queued"`
	TimeMined    int64             `json:"time_mined"`
	TimeDropped  int64             `json:"time_dropped"`
	BlockNumber  uint64            `json:"block_number,omitempty"`
	BlockHash    string            `json:"block_hash,omitempty"`
}

type Tx struct {
	ChainID          string   `json:"chain_id"`
	From             string   `json:"from"`
	To               string   `json:"to,omitempty"`
	Nonce            uint64   `json:"nonce"`
	Value            string   `json:"value"`
	Gas              uint64   `json:"gas"`
	GasPrice         *big.Int `json:"gas_price,omitempty"`
	MaxFeePerGas     string   `json:"max_fee_per_gas,omitempty"`
	MaxPriorityFee   string   `json:"max_priority_fee,omitempty"`
	MaxFeePerBlobGas string   `json:"max_fee_per_blob_gas,omitempty"`
	Data             string   `json:"data,omitempty"`
	Type             uint8    `json:"type"`
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
