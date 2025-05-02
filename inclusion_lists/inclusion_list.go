package inclusion_list

import (
	"encoding/json"
	"fmt"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"

	"github.com/r3labs/sse/v2"
)

func StreamInclusionList(beaconSseUrl string, l logger.Logger) {
	// Construct the SSE URL
	sseURL := fmt.Sprintf("%s/eth/v1/events?topics=block&topics=inclusion_list", beaconSseUrl)
	l.Info("Attempting connection to Beacon SSE endpoint", logger.Fields{
		"url": sseURL,
	})

	// Create a new SSE client
	client := dialSSEConnection(sseURL)

	// Subscribe to the SSE stream
	err := client.SubscribeRaw(func(msg *sse.Event) {
		// Ensure data is present
		if len(msg.Data) == 0 {
			l.Warn("received empty SSE event data")
			return
		}

		// Handle the incoming message
		if err := handleInclusionListMessage(msg.Data, l); err != nil {
			l.Error("Failed to handle inclusion list message:", err)
		}
	})
	if err != nil {
		l.Error("Failed to subscribe to SSE stream", err)
		return
	}

	l.Info("Successfully subscribed to SSE stream")
}

func dialSSEConnection(sseURL string) *sse.Client {
	client := sse.NewClient(sseURL)
	return client
}

func handleInclusionListMessage(jsonData []byte, l logger.Logger) error {
	var mempoolMessage model.MempoolMessage

	// Attempt unmarshal
	if err := json.Unmarshal(jsonData, &mempoolMessage); err != nil {
		l.Error("Error unmarshaling JSON", logger.Fields{
			"error": err,
			"data":  string(jsonData),
		})
		return fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	transactions := mempoolMessage.Data.Message.Transactions

	// Log transaction count
	if len(transactions) == 0 {
		l.Warn("Inclusion list message contains no transactions", logger.Fields{
			"slot": mempoolMessage.Data.Message.Slot,
		})
	} else {
		l.Info("Inclusion List Transactions", logger.Fields{
			"slot":        mempoolMessage.Data.Message.Slot,
			"Inclusion list tx count": len(transactions),
		})
	}

	return nil
}
