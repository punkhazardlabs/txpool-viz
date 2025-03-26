package broker

import (
	"context"
	"fmt"
	"time"

	"txpool-viz/config"
	"txpool-viz/internal/service"

	"github.com/redis/go-redis/v9"
)

func ProcessTransactions(ctx context.Context, cfg *config.Config, srvc *service.Service) {
	queues := []string{"pending", "queued"}

	for _, queue := range queues {
		go processQueue(ctx, cfg, queue, srvc)
	}
}

func processQueue(ctx context.Context, cfg *config.Config, queue string, srvc  *service.Service) {
	ticker := time.NewTicker(cfg.Polling["interval"])
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			srvc.Logger.Info("Shutting down ProcessTransactions...")
			return
		case <-ticker.C:
			res, err := srvc.Redis.HGetAll(ctx, queue).Result()

			if err == redis.Nil {
				srvc.Logger.Info("No messages in queue:", queue)
				continue
			} else if err != nil {
				srvc.Logger.Error("Error reading from queue:", queue, "error:", err)
				continue
			}

			srvc.Logger.Info(fmt.Sprintf("Received %d messages from %s", len(res), queue))
		}
	}
}
