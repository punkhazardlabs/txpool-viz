package inclusion_list

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/r3labs/sse/v2"
	"github.com/redis/go-redis/v9"
)

// InclusionListService encapsulates the logger and Redis client.
type InclusionListService struct {
	logger logger.Logger
	redis  *redis.Client
}

// NewInclusionListService constructs a new InclusionListService instance.
func NewInclusionListService(l logger.Logger, r *redis.Client) *InclusionListService {
	return &InclusionListService{
		logger: l,
		redis:  r,
	}
}

// StreamInclusionList connects to the Beacon SSE stream and processes inclusion list events.
func (s *InclusionListService) StreamInclusionList(ctx context.Context, beaconSseUrl string, websocket string, ethclient *ethclient.Client) {
	sseURL := fmt.Sprintf("%s/eth/v1/events?topics=block&topics=inclusion_list", beaconSseUrl)
	s.logger.Info("Attempting connection to Beacon SSE endpoint", logger.Fields{
		"url": sseURL,
	})

	client := dialSSEConnection(sseURL)

	events := make(chan *sse.Event)
	errs := make(chan error, 1)

	go func() {
		err := client.SubscribeRaw(func(msg *sse.Event) {
			if len(msg.Data) == 0 {
				s.logger.Warn("Received empty SSE event data")
				return
			}

			select {
			case <-ctx.Done():
				return
			case events <- msg:
			}
		})
		errs <- err
	}()

	s.logger.Info("Successfully subscribed to SSE stream")
	s.logger.Info("Subscribing to new blocks")

	go s.VerifyInclusionLists(ctx, websocket, ethclient)

	for {
		select {
		case <-ctx.Done():
			client.Unsubscribe(events)
			return

		case event := <-events:
			if event == nil {
				continue
			}
			
			if err := s.handleInclusionListMessage(ctx, event.Data); err != nil {
				s.logger.Error("Failed to handle inclusion list message", err)
			}

		case err := <-errs:
			if err != nil {
				s.logger.Error("SSE subscription error", err)
			}
			return
		}
	}
}

// VerifyInclusionLists processes incoming blocks against inclusion lists
func (s *InclusionListService) VerifyInclusionLists(ctx context.Context, websocketUrl string, ethClient *ethclient.Client) {
	l := s.logger
	conn, err := s.dialWebSocket(ctx, websocketUrl)
	if err != nil {
		l.Error("Error dialing websocket. Err: %s", err.Error())
	}
	defer conn.Close(websocket.StatusNormalClosure, "stream shutdown")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := conn.Read(ctx)
			if err != nil {
				if websocket.CloseStatus(err) == websocket.StatusNormalClosure || errors.Is(err, context.Canceled) {
					return
				}
				l.Error(fmt.Sprintf("Error reading stream %s", err.Error()))
				return
			}

			var event model.BlockSubscriptionEvent
			if err := json.Unmarshal(msg, &event); err != nil {
				l.Error("JSON parse error: %s", err.Error())
				continue
			}

			blockNumberStr := event.Params.Result.Number
			blockNumber, err := strconv.ParseInt(blockNumberStr, 0, 64)
			if err != nil {
				log.Fatalf("Failed to parse block number %s: %v", blockNumberStr, err)
			}
			l.Info("Block number:", blockNumber)

			block, err := ethClient.BlockByNumber(ctx, big.NewInt(blockNumber))
			if err != nil {
				l.Error("Failed to fetch block %s: %s", blockNumber, err.Error())
				continue
			}

			// Fetch inclusion list from Redis
			key := "txpool:inclusion:txns"
			blockField := blockNumber

			inclusionListCSV, err := s.redis.HGet(ctx, key, strconv.FormatInt(blockField, 10)).Result()
			if err == redis.Nil {
				l.Info("No inclusion list for block %s", blockField)
				continue
			} else if err != nil {
				l.Error("Redis HGet error: %s", err.Error())
				continue
			}

			expectedTxs := strings.Split(inclusionListCSV, ",")
			actualTxs := make(map[string]struct{})
			for _, tx := range block.Transactions() {
				actualTxs[tx.Hash().Hex()] = struct{}{}
			}

			// Check for missing transactions
			for _, expectedHash := range expectedTxs {
				if _, exists := actualTxs[expectedHash]; !exists {
					l.Error("Tx %s missing from block %s", expectedHash, blockField)
				} else {
					l.Info(("tx found in block!"))
				}
			}
		}
	}
}

// handleInclusionListMessage processes a single inclusion list message.
func (s *InclusionListService) handleInclusionListMessage(ctx context.Context, jsonData []byte) error {
	msg, err := parseInclusionListMessage(jsonData)
	if err != nil {
		s.logger.Error("Failed to parse inclusion list message", logger.Fields{
			"error": err,
			"data":  string(jsonData),
		})
		return err
	}

	transactions := msg.Data.Message.Transactions
	slot := msg.Data.Message.Slot
	txCount := len(transactions)

	if txCount == 0 && slot == "" {
		return nil
	}

	updated, err := s.updateInclusionScore(ctx, slot, txCount)
	if err != nil {
		s.logger.Error("Failed to update inclusion count", logger.Fields{
			"error": err,
			"slot":  slot,
		})
		return err
	}

	if updated {
		if err := s.storeInclusionTransactions(ctx, slot, transactions); err != nil {
			s.logger.Error("Failed to store transaction list", logger.Fields{
				"error": err,
				"slot":  slot,
			})
			return err
		}

		s.logger.Info("IL", "txs", string(jsonData))

		s.logger.Info("Updated inclusion list", logger.Fields{
			"slot":         slot,
			"new_tx_count": txCount,
		})
	}

	return nil
}

// storeInclusionTransactions saves the transactions for a given slot to a Redis HSET.
func (s *InclusionListService) storeInclusionTransactions(ctx context.Context, slot string, transactions []string) error {
	data, err := json.Marshal(transactions)
	if err != nil {
		return err
	}
	return s.redis.HSet(ctx, utils.RedisInclusionListTxnsKey(), slot, data).Err()
}

// updateInclusionScore updates the transaction count score in the Redis ZSET, only if the new count is greater.
func (s *InclusionListService) updateInclusionScore(ctx context.Context, slot string, txCount int) (bool, error) {
	z := redis.Z{
		Score:  float64(txCount),
		Member: slot,
	}

	updateCount, err := s.redis.ZAddArgs(ctx, utils.RedisInclusionScoreKey(), redis.ZAddArgs{
		GT:      true,
		Members: []redis.Z{z},
	}).Result()

	if err != nil {
		return false, err
	}

	return updateCount > 0, nil
}

// parseInclusionListMessage unmarshals the JSON inclusion list message into a MempoolMessage struct.
func parseInclusionListMessage(jsonData []byte) (model.MempoolMessage, error) {
	var msg model.MempoolMessage

	if err := json.Unmarshal(jsonData, &msg); err != nil {
		return model.MempoolMessage{}, fmt.Errorf("error unmarshaling JSON: %w", err)
	}
	return msg, nil
}

// dialSSEConnection establishes an SSE client connection.
func dialSSEConnection(sseURL string) *sse.Client {
	return sse.NewClient(sseURL)
}

func (s *InclusionListService) dialWebSocket(ctx context.Context, endpoint string) (*websocket.Conn, error) {
	l := s.logger
	conn, resp, err := websocket.Dial(ctx, endpoint, nil)

	if err != nil {
		return nil, fmt.Errorf("error connecting to websocket: %s", err)
	}

	l.Debug(fmt.Sprintf("Endpoint: inclusion list websocket connected with response %s", resp.Status))

	payload := &model.RPCRequest{
		Method:  "eth_subscribe",
		Params:  []string{"newHeads"},
		Id:      1,
		Jsonrpc: "2.0",
	}

	requestData, _ := json.Marshal(payload)

	if err = conn.Write(ctx, websocket.MessageText, requestData); err != nil {
		return nil, fmt.Errorf("error sending conection request: %s", err)
	}

	_, msg, err := conn.Read(ctx)

	if err != nil {
		return nil, fmt.Errorf("Failed to read response from inclusion list websocket")
	}

	var response model.SubscriptionResponse
	if err = json.Unmarshal(msg, &response); err != nil {
		l.Error(fmt.Sprintf("Response Error: %s", err))
	}

	l.Debug(fmt.Sprintf("Subscription ID for inclusion list: %s", msg))
	l.Info("Inlusion list websocket connected. Listening for blocks")

	return conn, nil
}
