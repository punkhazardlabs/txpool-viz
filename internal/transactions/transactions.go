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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/redis/go-redis/v9"
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

func PollTransactions(cfg *config.Config) {
	for _, endpoint := range cfg.UserCfg.Endpoints {
		go func(endpoint config.Endpoint) {
			ticker := time.NewTicker(cfg.UserCfg.Polling["interval"])
			defer ticker.Stop()

			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.UserCfg.Polling["timeout"])

				done := make(chan struct{})

				go func() {
					getTransactions(ctx, endpoint, cfg.RedisClient)
					close(done)
				}()

				select {
				case <-done:
				case <-ctx.Done():
					fmt.Printf("Transaction polling for %s timed out\n", endpoint.Name)
				}
				cancel()
			}
		}(endpoint)
	}
	select {}
}

func getTransactions(ctx context.Context, endpoint config.Endpoint, rdb *redis.Client) {
	fmt.Println("Getting transactions from", endpoint.Name)
	payload := &RPCRequest{
		Method:  "txpool_content",
		Params:  []string{},
		Id:      1,
		Jsonrpc: "2.0",
	}

	requestData, err := json.Marshal(payload)

	if err != nil {
		fmt.Println("Error marshalling request data:", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.Url, bytes.NewBuffer(requestData))

	if err != nil {
		fmt.Println("Error marshalling request data:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		if ctx.Err() != nil {
			fmt.Printf("Request to %s cancelled: %v\n", endpoint.Name, ctx.Err())
			return
		}
		fmt.Println("Error sending request:", err)
		return
	}

	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	var rpcResponse RPCResponse
	if err = json.Unmarshal(responseData, &rpcResponse); err != nil {
		fmt.Println("Error unmarshalling response data:", err)
		return
	}

	processTransactionBatch(ctx, rdb, "pending", rpcResponse.Result.Pending)
	processTransactionBatch(ctx, rdb, "queued", rpcResponse.Result.Queued)
}

// storeTransaction processes a batch of transactions and stores them in Redis
func processTransactionBatch(ctx context.Context, rdb *redis.Client, listName string, transactions map[string]map[string]*types.Transaction) {
	for address, txs := range transactions {
		for nonce, tx := range txs {
			jsonTx, err := json.Marshal(tx)
			if err != nil {
				fmt.Printf("Error marshalling TX (address: %s, nonce: %s): %v\n", address, nonce, err)
				continue
			}

			redisKey := fmt.Sprintf("%s:%s", address, nonce)

			fmt.Println("Storing in Redis:", redisKey)

			// Store transaction in Redis hash
			if err := rdb.HSet(ctx, listName, redisKey, jsonTx).Err(); err != nil {
				fmt.Printf("Error pushing to Redis (list: %s, key: %s): %v\n", listName, redisKey, err)
			}
		}
	}
}
