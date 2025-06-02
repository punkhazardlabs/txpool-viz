package inclusion_list

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"txpool-viz/internal/config"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
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
func (ils *InclusionListService) Stream(ctx context.Context, endpoints []config.Endpoint, beaconEndpoints []config.BeaconEndpoint) {

	for _, beaconEndpoint := range beaconEndpoints {
		go ils.streamBeaconUrl(ctx, beaconEndpoint)
	}

	for _, endpoint := range endpoints {
		go ils.processClientInclusionList(ctx, endpoint)
	}

	<-ctx.Done()
}

func (ils *InclusionListService) streamBeaconUrl(ctx context.Context, endpoint config.BeaconEndpoint) {
	sseURL := fmt.Sprintf("%s/eth/v1/events?topics=block&topics=inclusion_list", endpoint.BeaconUrl)
	ils.logger.Info("Attempting connection to Beacon SSE endpoint", logger.Fields{
		"url": sseURL,
	})

	client := dialSSEConnection(sseURL)

	events := make(chan *sse.Event)
	errs := make(chan error, 1)

	go func() {
		err := client.SubscribeRaw(func(msg *sse.Event) {
			if len(msg.Data) == 0 {
				ils.logger.Warn("Received empty SSE event data")
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

	ils.logger.Info("Successfully subscribed to SSE stream")

	for {
		select {
		case <-ctx.Done():
			client.Unsubscribe(events)
			return

		case event := <-events:
			if event == nil {
				continue
			}

			if err := ils.handleInclusionListMessage(ctx, event.Data); err != nil {
				ils.logger.Error("Failed to handle inclusion list message", err)
			}

		case err := <-errs:
			if err != nil {
				ils.logger.Error("SSE subscription error", err)
			}
			return
		}
	}
}

// handleInclusionListMessage processes a single inclusion list message.
func (ils *InclusionListService) handleInclusionListMessage(ctx context.Context, jsonData []byte) error {
	msg, err := parseInclusionListMessage(jsonData)
	if err != nil {
		ils.logger.Error("Failed to parse inclusion list message", logger.Fields{
			"error": err,
			"data":  string(jsonData),
		})
		return err
	}

	transactions := make([]*types.Transaction, 0, len(msg.Data.Message.Transactions))

	for _, txDataHex := range msg.Data.Message.Transactions {
		txData, err := hexutil.Decode(txDataHex)
		if err != nil {
			ils.logger.Error("Hex decode failed", "err", err)
			continue
		}

		tx := new(types.Transaction)
		err = tx.UnmarshalBinary(txData)
		if err != nil {
			ils.logger.Error("UnmarshalBinary failed", "err", err)
			continue
		}

		transactions = append(transactions, tx)
	}

	slot := msg.Data.Message.Slot
	txCount := len(transactions)

	if txCount == 0 && slot == "" {
		return nil
	}

	updated, err := ils.updateInclusionScore(ctx, slot, txCount)
	if err != nil {
		ils.logger.Error("Failed to update inclusion count", logger.Fields{
			"error": err,
			"slot":  slot,
		})
		return err
	}

	if updated {
		if err := ils.storeInclusionTransactions(ctx, slot, transactions); err != nil {
			ils.logger.Error("Failed to store transaction list", logger.Fields{
				"error": err,
				"slot":  slot,
			})
			return err
		}

		ils.logger.Info("Updated inclusion list", logger.Fields{
			"slot":         slot,
			"new_tx_count": txCount,
		})
	}

	return nil
}

// storeInclusionTransactions saves the transactions for a given slot to a Redis HSET.
func (ils *InclusionListService) storeInclusionTransactions(ctx context.Context, slot string, transactions []*types.Transaction) error {
	data, err := json.Marshal(transactions)
	if err != nil {
		return err
	}
	return ils.redis.HSet(ctx, utils.RedisInclusionListTxnsKey(), slot, data).Err()
}

// updateInclusionScore updates the transaction count score in the Redis ZSET, only if the new count is greater.
func (ils *InclusionListService) updateInclusionScore(ctx context.Context, slot string, txCount int) (bool, error) {
	z := redis.Z{
		Score:  float64(txCount),
		Member: slot,
	}

	updateCount, err := ils.redis.ZAddArgs(ctx, utils.RedisInclusionScoreKey(), redis.ZAddArgs{
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

func (ils *InclusionListService) processClientInclusionList(ctx context.Context, endpoint config.Endpoint) {
	// Dial WebSocket endpoint directly
	client, err := ethclient.DialContext(ctx, endpoint.Websocket)
	if err != nil {
		ils.logger.Error("Failed to connect to WebSocket endpoint", "err", err.Error())
		return
	}

	// Subscribe to new block headers
	headers := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(ctx, headers)
	if err != nil {
		ils.logger.Error("Error subscribing to newHeads", "err", err.Error())
		return
	}

	defer sub.Unsubscribe()

	ils.logger.Info("Subscribed to new block headers")

	var wg sync.WaitGroup
	defer wg.Wait()

	// Process each new block as it arrives
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case err := <-sub.Err():
			ils.logger.Error("Subscription error", "err", err)
			return
		case header := <-headers:
			ils.logger.Info("New Block", "block_number", header.Number.String())
			wg.Add(1)
			go ils.processBlock(ctx, client, header.Number)
		}
	}
}

func (ils *InclusionListService) processBlock(ctx context.Context, ethClient *ethclient.Client, blockNumber *big.Int) {
	if ctx.Err() != nil {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
		// Fetch full block
		block, err := ethClient.BlockByNumber(ctx, blockNumber)
		if err != nil {
			ils.logger.Error("Failed to fetch block", "blockNumber", blockNumber, "err", err)
			return
		}

		// Get tx hashes in this block
		blockTxHashes := make(map[common.Hash]bool)
		for _, tx := range block.Transactions() {
			blockTxHashes[tx.Hash()] = true
		}

		// Retrieve inclusion list txs from storage
		slotKey := utils.RedisInclusionListTxnsKey()
		ilTxData, err := ils.redis.HGet(ctx, slotKey, blockNumber.String()).Result()
		if err != nil {
			ils.logger.Warn("Failed to get inclusion list", "slotKey", slotKey, "err", err.Error(), "blocknumber", blockNumber)
			return
		}

		// Decode inclusion list txs (as array)
		var ilTxs []*types.Transaction
		if err := json.Unmarshal([]byte(ilTxData), &ilTxs); err != nil {
			ils.logger.Error("Failed to decode IL txs array", "err", err)
			return
		}

		// Extract hashes
		var ilTxHashes []common.Hash
		for _, tx := range ilTxs {
			if tx != nil {
				ilTxHashes = append(ilTxHashes, tx.Hash())
			}
		}

		// Compare IL tx hashes with block tx hashes
		var included, missing []common.Hash
		for _, hash := range ilTxHashes {
			if blockTxHashes[hash] {
				included = append(included, hash)
			} else {
				missing = append(missing, hash)
			}
		}

		inclusionReportKey := utils.RedisInclusionListReportKey()
		slot := blockNumber.String()

		report := model.InclusionReport{
			Included: included,
			Missing:  missing,
			Summary: model.InclusionSummary{
				Total:    len(ilTxHashes),
				Included: len(included),
				Missing:  len(missing),
			},
		}

		reportJSON, err := json.Marshal(report)
		if err != nil {
			ils.logger.Error("Failed to marshal inclusion report", "err", err)
			return
		}

		// Store in hash under slot field
		if err := ils.redis.HSet(ctx, inclusionReportKey, slot, reportJSON).Err(); err != nil {
			ils.logger.Error("Failed to store inclusion report in hash", "err", err)
		}
	}
}
