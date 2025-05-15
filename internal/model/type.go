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

type BlockSubscriptionEvent struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  struct {
			Subscription string  `json:"subscription"`
			Result       NewHead `json:"result"`
	} `json:"params"`
}

type NewHead struct {
	ParentHash             string `json:"parentHash"`
	Sha3Uncles             string `json:"sha3Uncles"`
	Miner                  string `json:"miner"`
	StateRoot              string `json:"stateRoot"`
	TransactionsRoot       string `json:"transactionsRoot"`
	ReceiptsRoot           string `json:"receiptsRoot"`
	LogsBloom              string `json:"logsBloom"`
	Difficulty             string `json:"difficulty"`
	Number                 string `json:"number"`
	GasLimit               string `json:"gasLimit"`
	GasUsed                string `json:"gasUsed"`
	Timestamp              string `json:"timestamp"`
	ExtraData              string `json:"extraData"`
	MixHash                string `json:"mixHash"`
	Nonce                  string `json:"nonce"`
	Hash                   string `json:"hash"`
	BaseFeePerGas          string `json:"baseFeePerGas"`
	WithdrawalsRoot        string `json:"withdrawalsRoot"`
	BlobGasUsed            string `json:"blobGasUsed"`
	ExcessBlobGas          string `json:"excessBlobGas"`
	ParentBeaconBlockRoot  string `json:"parentBeaconBlockRoot"`
	RequestsHash           string `json:"requestsHash"`
}