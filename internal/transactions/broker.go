package transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"txpool-viz/internal/config"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/internal/service"
	"txpool-viz/internal/storage"
	"txpool-viz/utils"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

func Stream(ctx context.Context, cfg *config.Config, srvc *service.Service, wg *sync.WaitGroup) {
	ProcessTransactions(ctx, cfg, srvc)

	for _, endpoint := range cfg.Endpoints {
		wg.Add(1)
		go func(endpoint config.Endpoint) {
			defer wg.Done()
			streamEndpoint(ctx, endpoint, srvc.Logger, srvc.Redis)
		}(endpoint)
	}
}

func streamEndpoint(ctx context.Context, endpoint config.Endpoint, l logger.Logger, r *redis.Client) {
	// Make the websocket connection
	conn, err := dialWebSocket(ctx, endpoint, l)

	if err != nil {
		l.Error(err.Error(), logger.Fields{
			"endpoint": endpoint.Name,
			"url":      endpoint.Websocket,
		})
	}

	// Defer websocket close
	defer conn.Close(websocket.StatusNormalClosure, "stream shutdown")

	// retrieve the universal redis sorted set key
	redisUniversalSortedSetKey := utils.RedisUniversalKey()

	// Create new per-client redis storage instance
	storage := storage.NewClientStorage(endpoint.Name, r, l)

	// Start streaming mempool txHashes
	for {
		select {
		case <-ctx.Done():
			l.Info("Shutting down streamEndpoint", logger.Fields{"endpoint": endpoint.Name})
			return
		default:
			time := time.Now().Unix()
			_, msg, err := conn.Read(ctx)

			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure || errors.Is(err, context.Canceled) {
					return
				}
				l.Error(fmt.Sprintf("Error reading stream %s", err.Error()), logger.Fields{"endpoint": endpoint.Name})
				return
			}

			var event model.SubscriptionResponse
			if err := json.Unmarshal(msg, &event); err != nil {
				l.Error("JSON parse error")
				continue
			}

			txHash := event.Params.TxHash

			r.ZAddNX(ctx, redisUniversalSortedSetKey, redis.Z{
				Score:  float64(time),
				Member: txHash,
			})

			if err := storage.StoreTransaction(ctx, txHash, time); err != nil {
				l.Error("Error storing tx to cache", logger.Fields{"txHash": txHash})
			}

			streamKey := utils.RedisStreamKey(endpoint.Name)
			r.RPush(ctx, streamKey, fmt.Sprintf("%s:%s", endpoint.Name, event.Params.TxHash))
		}
	}

}

func dialWebSocket(ctx context.Context, endpoint config.Endpoint, l logger.Logger) (*websocket.Conn, error) {
	conn, resp, err := websocket.Dial(ctx, endpoint.Websocket, nil)

	if err != nil {
		return nil, fmt.Errorf("error connecting to websocket: %s", err)
	}

	l.Debug(fmt.Sprintf("Endpoint: %s Websocket connected with repsonse %s", endpoint.Name, resp.Status))

	payload := &model.RPCRequest{
		Method:  "eth_subscribe",
		Params:  []string{"newPendingTransactions"},
		Id:      1,
		Jsonrpc: "2.0",
	}

	requestData, _ := json.Marshal(payload)

	if err = conn.Write(ctx, websocket.MessageText, requestData); err != nil {
		return nil, fmt.Errorf("error sending conection request: %s", err)
	}

	_, msg, err := conn.Read(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to read response from socket: %s", endpoint.Name)
	}

	var response model.SubscriptionResponse
	if err = json.Unmarshal(msg, &response); err != nil {
		l.Error(fmt.Sprintf("Response Error: %s", err))
	}

	l.Debug(fmt.Sprintf("Subscription ID for endpoint: %s", msg), logger.Fields{"endpoint": endpoint.Name})
	l.Info("Websocket connected", logger.Fields{"endpoint": endpoint.Name})

	return conn, nil
}
