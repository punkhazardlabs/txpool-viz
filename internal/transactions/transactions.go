package transactions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/service"
	"txpool-viz/pkg"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

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
	Subscrition string `json:"subscription"`
	TxHash      string `json:"result"`
}

type Result struct {
	Pending map[string]map[string]*types.Transaction `json:"pending"`
	Queued  map[string]map[string]*types.Transaction `json:"queued"`
}

func Stream(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	// Start the streams
	for _, endpoint := range cfg.Endpoints {
		go streamEndpoint(ctx, endpoint, srvc.Logger, *srvc.Redis)
	}
}

func streamEndpoint(ctx context.Context, endpoint config.Endpoint, l pkg.Logger, r redis.Client) {
	conn, resp, err := websocket.Dial(ctx, endpoint.Websocket, nil)

	if err != nil {
		l.Error(fmt.Sprintf("Error connecting to websocket. Error: %s", err), pkg.Fields{"endpoint": endpoint.Name})
		return
	}

	l.Info(fmt.Sprintf("Endpoint: %s Websocket connected with repsonse %s", endpoint.Name, resp.Status))

	defer conn.Close(websocket.StatusNormalClosure, "Closing connection")

	payload := &RPCRequest{
		Method:  "eth_subscribe",
		Params:  []string{"newPendingTransactions"},
		Id:      1,
		Jsonrpc: "2.0",
	}

	requestData, _ := json.Marshal(payload)

	if err = conn.Write(ctx, websocket.MessageText, requestData); err != nil {
		l.Error(fmt.Sprintf("Error sending conection request: %s", err))
	}

	_, msg, err := conn.Read(ctx)

	if err != nil {
		l.Info(fmt.Sprintf("Failed to read response from socket: %s", endpoint.Name))
	}

	var response SubscriptionResponse
	if err = json.Unmarshal(msg, &response); err != nil {
		l.Error(fmt.Sprintf("Response Error: %s", err))
	}

	l.Info(fmt.Sprintf("Subscription ID for endpoint: %s", msg), pkg.Fields{"endpoint": endpoint.Name})

	for {
		_, msg, err := conn.Read(context.Background())

		if err != nil {
			l.Error("Error reading stream", pkg.Fields{"endpoint": endpoint.Name})
		}

		var event SubscriptionResponse

		err = json.Unmarshal(msg, &event)
		if err != nil {
			l.Error("JSON parse error")
			continue
		}

		l.Info(fmt.Sprintf("Subscription: %s, TxHash: %s", event.Params.Subscrition, event.Params.TxHash))

		// Queue the TX
		r.RPush(context.Background(), "queuedTxHash", fmt.Sprintf("%s:%s", endpoint.Name, event.Params.TxHash))
	}
}

// PollTransactions polls transactions from endpoints at regular intervals
func PollTransactions(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	storage := NewStorage(srvc.Redis, srvc.Logger)

	for _, endpoint := range cfg.Endpoints {
		go func(endpoint config.Endpoint) {
			ticker := time.NewTicker(cfg.Polling["interval"])
			defer ticker.Stop()

			srvc.Logger.Info("Polling started for:", endpoint.Name)

			for {
				select {
				case <-ctx.Done():
					srvc.Logger.Info("Shutting down PollTransactions for", endpoint.Name)
					return
				case <-ticker.C:
					pollCtx, cancel := context.WithTimeout(ctx, cfg.Polling["timeout"])
					getTransactions(pollCtx, endpoint, storage, srvc.Logger)
					cancel()
				}
			}
		}(endpoint)
	}
}

func getTransactions(ctx context.Context, endpoint config.Endpoint, storage *Storage, l pkg.Logger) {
	l.Info("Polling transactions", pkg.Fields{"endpoint": endpoint.Name})

	payload := &RPCRequest{
		Method:  "txpool_content",
		Params:  []string{},
		Id:      1,
		Jsonrpc: "2.0",
	}

	requestData, err := json.Marshal(payload)
	if err != nil {
		l.Error("Error marshalling request data", pkg.Fields{"error": err})
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.RPCUrl, bytes.NewBuffer(requestData))
	if err != nil {
		l.Error("Error creating request", pkg.Fields{"error": err})
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			l.Error("Request cancelled", pkg.Fields{"endpoint": endpoint.Name, "error": ctx.Err()})
			return
		}
		l.Error("Error sending request", pkg.Fields{"error": err})
		return
	}

	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("Error reading response", pkg.Fields{"error": err})
		return
	}

	var rpcResponse RPCResponse
	if err = json.Unmarshal(responseData, &rpcResponse); err != nil {
		l.Error("Error unmarshalling response data", pkg.Fields{"error": err})
		return
	}

	processTransactionBatch(ctx, storage, "pending", rpcResponse.Result.Pending)
	processTransactionBatch(ctx, storage, "queued", rpcResponse.Result.Queued)

	l.Info(fmt.Sprintf("Processed %d pending txs, %d queued txs",
		len(rpcResponse.Result.Pending), len(rpcResponse.Result.Queued)),
		pkg.Fields{"endpoint": endpoint.Name})
}

// processTransactionBatch processes a batch of transactions and stores them
func processTransactionBatch(ctx context.Context, storage *Storage, listName string, transactions map[string]map[string]*types.Transaction) {
	for address, txs := range transactions {
		for nonce, tx := range txs {
			// Create metadata for the transaction
			metadata := TransactionMetadata{
				Nonce:      tx.Nonce(),
				From:       address,
				IsContract: false, // This would need to be determined by checking the contract code
				Timestamp:  time.Now().Unix(),
			}

			// Handle To address, which can be nil for contract creation
			if tx.To() != nil {
				metadata.To = tx.To().String()
			} else {
				metadata.To = "" // Empty string for contract creation
			}

			// Set transaction type and gas-related fields
			switch tx.Type() {
			case types.BlobTxType:
				metadata.Type = BlobTx
				// Note: We can't access MaxFeePerBlobGas directly as it's not exposed in the interface
			case types.DynamicFeeTxType:
				metadata.Type = EIP1559Tx
				// Note: We can't access MaxFeePerGas and MaxPriorityFee directly as they're not exposed in the interface
			default:
				metadata.Type = LegacyTx
				metadata.GasPrice = tx.GasPrice()
			}

			// Create stored transaction
			storedTx := &StoredTransaction{
				Hash:     tx.Hash().Hex(),
				Metadata: metadata,
			}

			// Store the transaction in the appropriate queue
			if err := storage.StoreTransaction(ctx, storedTx, listName); err != nil {
				fmt.Printf("Error storing TX (address: %s, nonce: %s): %v\n", address, nonce, err)
			}
		}
	}
}

// // processNewTransaction processes a new transaction from the mempool
// func (ts *TransactionStreamer) processNewTransaction(ctx context.Context, tx *types.Transaction) {
// 	// Check if we already have this transaction
// 	ts.mu.RLock()
// 	_, exists := ts.txStore[tx.Hash().Hex()]
// 	ts.mu.RUnlock()

// 	if exists {
// 		return
// 	}

// 	// Get full transaction details
// 	_, _, err := ts.client.TransactionByHash(ctx, tx.Hash())
// 	if err != nil {
// 		fmt.Printf("Error getting transaction details: %v\n", err)
// 		return
// 	}

// 	// Get sender address
// 	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
// 	if err != nil {
// 		fmt.Printf("Error getting sender address: %v\n", err)
// 		return
// 	}

// 	// Create new stored transaction
// 	storedTx := &StoredTransaction{
// 		Hash: tx.Hash().Hex(),
// 		Metadata: TransactionMetadata{
// 			Status:       StatusQueued,
// 			TimeReceived: time.Now().Unix(),
// 			Type:         getTransactionType(tx),
// 			Nonce:        tx.Nonce(),
// 			From:         sender.Hex(),
// 			To:           tx.To().Hex(),
// 			Timestamp:    time.Now().Unix(),
// 		},
// 	}

// 	// Set gas price based on transaction type
// 	switch tx.Type() {
// 	case types.DynamicFeeTxType:
// 		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
// 		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
// 	case types.BlobTxType:
// 		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
// 		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
// 		storedTx.Metadata.MaxFeePerBlobGas = tx.BlobGasFeeCap()
// 	default:
// 		storedTx.Metadata.GasPrice = tx.GasPrice()
// 	}

// 	// Store transaction
// 	ts.mu.Lock()
// 	ts.txStore[storedTx.Hash] = storedTx
// 	ts.mu.Unlock()
// }

// // trackTransactionStatus tracks the status of transactions
// func (ts *TransactionStreamer) trackTransactionStatus(ctx context.Context) {
// 	ticker := time.NewTicker(5 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			ts.mu.RLock()
// 			for _, tx := range ts.txStore {
// 				// Skip if already mined or dropped
// 				if tx.Metadata.Status == StatusMined || tx.Metadata.Status == StatusDropped {
// 					continue
// 				}

// 				// Check if transaction is pending
// 				if tx.Metadata.Status == StatusQueued {
// 					var result map[string]map[string]map[string]*types.Transaction
// 					err := ts.rpcClient.Call(&result, "txpool_content")
// 					if err == nil {
// 						// Check if transaction is in pending pool
// 						for _, account := range result["pending"] {
// 							if _, exists := account[tx.Hash]; exists {
// 								ts.mu.RUnlock()
// 								ts.mu.Lock()
// 								tx.Metadata.Status = StatusPending
// 								tx.Metadata.TimePending = time.Now().Unix()
// 								ts.mu.Unlock()
// 								ts.mu.RLock()
// 								break
// 							}
// 						}
// 					}
// 				}

// 				// Check if transaction is mined
// 				if tx.Metadata.Status == StatusPending {
// 					hash := common.HexToHash(tx.Hash)
// 					receipt, err := ts.client.TransactionReceipt(ctx, hash)
// 					if err == nil && receipt != nil {
// 						ts.mu.RUnlock()
// 						ts.mu.Lock()
// 						tx.Metadata.Status = StatusMined
// 						tx.Metadata.TimeMined = time.Now().Unix()
// 						tx.Metadata.BlockNumber = receipt.BlockNumber.Uint64()
// 						tx.Metadata.BlockHash = receipt.BlockHash.Hex()
// 						ts.mu.Unlock()
// 						ts.mu.RLock()
// 					}
// 				}
// 			}
// 			ts.mu.RUnlock()
// 		case <-ctx.Done():
// 			return
// 		case <-ts.stopChan:
// 			return
// 		}
// 	}
// }

// getTransactionType determines the type of transaction
func getTransactionType(tx *types.Transaction) TransactionType {
	switch {
	case tx.Type() == types.BlobTxType:
		return BlobTx
	case tx.Type() == types.DynamicFeeTxType:
		return EIP1559Tx
	case tx.Type() == types.AccessListTxType:
		return EIP2930Tx
	default:
		return LegacyTx
	}
}
