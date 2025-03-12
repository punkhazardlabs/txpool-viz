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
)

type RPCRequest struct {
	Method  string
	Params  []string
	Id      int
	Jsonrpc string
}

func PollTransactions(cfg *config.Config) {
	for _, endpoint := range cfg.Endpoints {
		go func(endpoint config.Endpoint) {
			ticker := time.NewTicker((cfg.Polling["interval"]))
			defer ticker.Stop()

			for range ticker.C {
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Polling["timeout"])
				go func() {
					defer cancel()
					getTransactions(ctx, endpoint)
				}()
				select {
				case <-ctx.Done():
					fmt.Printf("Transaction polling for %s timed out\n", endpoint.Name)
				case <-time.After(cfg.Polling["timeout"]):
				}
			}
		}(endpoint)
	}
	select {}
}

func getTransactions(ctx context.Context, endpoint config.Endpoint) {
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

	fmt.Println(string(responseData))
}
