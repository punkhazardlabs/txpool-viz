package model

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

type MinedTxStatus uint64

const (
	Failed MinedTxStatus = iota
	Success
)

func (ms MinedTxStatus) String() string {
	switch {
	case ms == Failed:
		return "failed"
	case ms == Success:
		return "success"
	default:
		return "unknown"
	}
}

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
type TransactionStatus string

const (
	StatusReceived TransactionStatus = "received"
	StatusPending  TransactionStatus = "pending"
	StatusQueued   TransactionStatus = "queued"
	StatusMined    TransactionStatus = "mined"
	StatusDropped  TransactionStatus = "dropped"
)

// TransactionMetadata contains additional metadata for filtering and grouping
type TransactionMetadata struct {
	Status       TransactionStatus `json:"status"`                  // Custom status enum
	IsContract   bool              `json:"is_contract"`             // Whether destination is contract
	TimeReceived int64             `json:"time_received,omitempty"` // When seen in mempool
	TimePending  *int64            `json:"time_pending,omitempty"`
	TimeQueued   int64             `json:"time_queued"`
	TimeMined    *int64            `json:"time_mined"`
	TimeDropped  int64             `json:"time_dropped"`
	BlockNumber  uint64            `json:"block_number,omitempty"`
	BlockHash    string            `json:"block_hash,omitempty"`
	MineStatus   string            `json:"mine_status,omitempty"`
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

type RPCRequest struct {
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	Id      int      `json:"id"`
	Jsonrpc string   `json:"jsonrpc"`
}

type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  Result `json:"result"`
}

type SubscriptionResponse struct {
	Jsonrpc string             `json:"jsonrpc"`
	Method  string             `json:"method"`
	Params  SubscriptionParams `json:"params"`
}

type SubscriptionParams struct {
	Subscription string `json:"subscription"`
	TxHash       string `json:"result"`
}

type Result struct {
	Pending map[string]map[string]*types.Transaction `json:"pending"`
	Queued  map[string]map[string]*types.Transaction `json:"queued"`
}

type SSEMessage struct {
	Slot                       string   `json:"slot"`
	ValidatorIndex             string   `json:"validator_index"`
	InclusionListCommitteeRoot string   `json:"inclusion_list_committee_root"`
	Transactions               []string `json:"transactions"` // Array of transaction hashes
}

type Data struct {
	Message   SSEMessage `json:"message"`
	Signature string     `json:"signature"`
}

type MempoolMessage struct {
	Version string `json:"version"`
	Data    Data   `json:"data"`
}
