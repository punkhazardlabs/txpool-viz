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
	"txpool-viz/internal/model"
	"txpool-viz/internal/service"
	"txpool-viz/internal/storage"
	"txpool-viz/pkg"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
)

type Result struct {
	Pending map[string]map[string]*types.Transaction `json:"pending"`
	Queued  map[string]map[string]*types.Transaction `json:"queued"`
}

func Stream(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	// Start listening to txhashes
	ProcessTransactions(ctx, cfg, srvc)

	// Start the streaming txhashes
	for _, endpoint := range cfg.Endpoints {
		go streamEndpoint(ctx, endpoint, srvc.Logger, srvc.Redis)
	}
}

func streamEndpoint(ctx context.Context, endpoint config.Endpoint, l pkg.Logger, r *redis.Client) {
	conn, resp, err := websocket.Dial(ctx, endpoint.Websocket, nil)

	if err != nil {
		l.Error(fmt.Sprintf("Error connecting to websocket. Error: %s", err), pkg.Fields{"endpoint": endpoint.Name})
		return
	}

	l.Debug(fmt.Sprintf("Endpoint: %s Websocket connected with repsonse %s", endpoint.Name, resp.Status))

	defer conn.Close(websocket.StatusNormalClosure, "Closing connection")

	payload := &model.RPCRequest{
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

	var response model.SubscriptionResponse
	if err = json.Unmarshal(msg, &response); err != nil {
		l.Error(fmt.Sprintf("Response Error: %s", err))
	}

	l.Debug(fmt.Sprintf("Subscription ID for endpoint: %s", msg), pkg.Fields{"endpoint": endpoint.Name})
	l.Info("Websocket connected", pkg.Fields{"endpoint": endpoint.Name})

	for {
		time := time.Now().Unix()

		_, msg, err := conn.Read(context.Background())

		if err != nil {
			l.Error("Error reading stream", pkg.Fields{"endpoint": endpoint.Name})
		}

		var event model.SubscriptionResponse

		err = json.Unmarshal(msg, &event)
		if err != nil {
			l.Error("JSON parse error")
			continue
		}

		txHash := event.Params.TxHash

		// Add to universal queue if it does not exist
		r.ZAddNX(ctx, "universal", redis.Z{
			Score:  float64(time),
			Member: txHash,
		})

		// Create the tx Stored Transaction entry and add to the client queue
		storedTx := &model.StoredTransaction{
			Hash: txHash,
			Metadata: model.TransactionMetadata{
				Status:       model.StatusReceived,
			},
		}

		storage := storage.NewStorage(r, l)
		err = storage.StoreTransaction(ctx, storedTx, endpoint.Name)

		if err != nil {
			l.Error("Error storing tx to cache", pkg.Fields{
				"txHash": txHash,
			})
		} 

		// Queue the TX
		r.RPush(context.Background(), fmt.Sprintf("stream:%s", endpoint.Name), fmt.Sprintf("%s:%s", endpoint.Name, event.Params.TxHash))
	}
}

// PollTransactions polls transactions from endpoints at regular intervals
func PollTransactions(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	storage := storage.NewStorage(srvc.Redis, srvc.Logger)

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

func getTransactions(ctx context.Context, endpoint config.Endpoint, storage *storage.Storage, l pkg.Logger) {
	l.Info("Polling transactions", pkg.Fields{"endpoint": endpoint.Name})

	payload := &model.RPCRequest{
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

	var rpcResponse model.RPCResponse
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
func processTransactionBatch(ctx context.Context, storage *storage.Storage, listName string, transactions map[string]map[string]*types.Transaction) {
	for address, txs := range transactions {
		for nonce, tx := range txs {
			// Create metadata for the transaction
			metadata := model.TransactionMetadata{
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
				metadata.Type = model.BlobTx
				// Note: We can't access MaxFeePerBlobGas directly as it's not exposed in the interface
			case types.DynamicFeeTxType:
				metadata.Type = model.EIP1559Tx
				// Note: We can't access MaxFeePerGas and MaxPriorityFee directly as they're not exposed in the interface
			default:
				metadata.Type = model.LegacyTx
				metadata.GasPrice = tx.GasPrice()
			}

			// Create stored transaction
			storedTx := &model.StoredTransaction{
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
