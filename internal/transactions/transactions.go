package transactions

import (
	"context"
	"fmt"

	cfg "txpool-viz/config"
)

func PollTransactions(ctx context.Context, cfg cfg.Config) {
	for _, endpoint := range cfg.Endpoints {
		fmt.Println(endpoint)
	}
}
