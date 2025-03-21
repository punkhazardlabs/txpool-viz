package broker

import (
	"context"
	"time"
	"fmt"

	"txpool-viz/config"

	"github.com/redis/go-redis/v9"
)

func ProcessTransactions(ctx context.Context, cfg *config.Config) {
	queues := []string{"pending", "queued"}

	for _, queue := range queues {
		go processQueue(ctx, cfg, queue)
	}
}

func processQueue(ctx context.Context, cfg *config.Config, queue string) {
	ticker := time.NewTicker(cfg.UserCfg.Polling["interval"])
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cfg.Logger.Info("Shutting down ProcessTransactions...")
			return
		case <-ticker.C:
			res, err := cfg.RedisClient.HGetAll(ctx, queue).Result()

			if err == redis.Nil {
				cfg.Logger.Info("No messages in queue:", queue)
				continue
			} else if err != nil {
				cfg.Logger.Error("Error reading from queue:", queue, "error:", err)
				continue
			}

			cfg.Logger.Info(fmt.Sprintf("Received %d messages from %s", len(res), queue))
		}
	}
}
