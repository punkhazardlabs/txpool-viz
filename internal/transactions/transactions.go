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
	"txpool-viz/pkg"

	"github.com/ethereum/go-ethereum/core/types"
)

type RPCRequest struct {
	Method  string
	Params  []string
	Id      int
	Jsonrpc string
}

type RPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  Result `json:"result"`
}

type Result struct {
	Pending map[string]map[string]*types.Transaction `json:"pending"`
	Queued  map[string]map[string]*types.Transaction `json:"queued"`
}

func PollTransactions(ctx context.Context, cfg *config.Config) {
	storage := NewStorage(cfg.RedisClient)

	for _, endpoint := range cfg.UserCfg.Endpoints {
		go func(endpoint config.Endpoint) {
			ticker := time.NewTicker(cfg.UserCfg.Polling["interval"])
			defer ticker.Stop()

			cfg.Logger.Info("Polling started for:", endpoint.Name)

			for {
				select {
				case <-ctx.Done():
					cfg.Logger.Info("Shutting down PollTransactions for", endpoint.Name)
					return
				case <-ticker.C:
					pollCtx, cancel := context.WithTimeout(ctx, cfg.UserCfg.Polling["timeout"])
					getTransactions(pollCtx, endpoint, storage, cfg.Logger)
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

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.Url, bytes.NewBuffer(requestData))
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
				To:         tx.To().String(),
				IsContract: false, // This would need to be determined by checking the contract code
				Timestamp:  time.Now().Unix(),
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
				Tx:       tx,
				Metadata: metadata,
			}

			// Store the transaction
			if err := storage.StoreTransaction(ctx, storedTx); err != nil {
				fmt.Printf("Error storing TX (address: %s, nonce: %d): %v\n", address, nonce, err)
			}
		}
	}
}
