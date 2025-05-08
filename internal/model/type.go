package model

import "github.com/ethereum/go-ethereum/core/types"

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
	Slot                          string   `json:"slot"`
	ValidatorIndex                string   `json:"validator_index"`
	InclusionListCommitteeRoot    string   `json:"inclusion_list_committee_root"`
	Transactions                  []string `json:"transactions"` // Array of transaction hashes
}

type Data struct {
	Message   SSEMessage `json:"message"`
	Signature string  `json:"signature"`
}

type MempoolMessage struct {
	Version string `json:"version"`
	Data    Data   `json:"data"`
}