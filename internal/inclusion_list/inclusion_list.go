package inclusion_list

import (
	"context"
	"encoding/json"
	"fmt"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

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
func (s *InclusionListService) StreamInclusionList(ctx context.Context, beaconSseUrl string) {
	sseURL := fmt.Sprintf("%s/eth/v1/events?topics=block&topics=inclusion_list", beaconSseUrl)
	s.logger.Info("Attempting connection to Beacon SSE endpoint", logger.Fields{
		"url": sseURL,
	})

	client := dialSSEConnection(sseURL)

	err := client.SubscribeRaw(func(msg *sse.Event) {
		if len(msg.Data) == 0 {
			s.logger.Warn("Received empty SSE event data")
			return
		}

		if err := s.handleInclusionListMessage(ctx, msg.Data); err != nil {
			s.logger.Error("Failed to handle inclusion list message", err)
		}
	})

	if err != nil {
		s.logger.Error("Failed to subscribe to SSE stream", err)
		return
	}

	s.logger.Info("Successfully subscribed to SSE stream")
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

		s.logger.Info("Updated inclusion list in Redis", logger.Fields{
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
