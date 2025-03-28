package transactions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// TransactionStreamer handles streaming and tracking transaction status
type TransactionStreamer struct {
	client    *ethclient.Client
	rpcClient *rpc.Client
	txStore   map[string]*StoredTransaction
	mu        sync.RWMutex
	stopChan  chan struct{}
}

// NewTransactionStreamer creates a new transaction streamer
func NewTransactionStreamer(ethClient *ethclient.Client, rpcClient *rpc.Client) *TransactionStreamer {
	return &TransactionStreamer{
		client:    ethClient,
		rpcClient: rpcClient,
		txStore:   make(map[string]*StoredTransaction),
		stopChan:  make(chan struct{}),
	}
}

// Start begins streaming transactions and tracking their status
func (ts *TransactionStreamer) Start(ctx context.Context) error {
	// Start mempool polling goroutine
	go ts.pollMempool(ctx)

	// Start status tracking goroutine
	go ts.trackTransactionStatus(ctx)

	return nil
}

// Stop stops the transaction streamer
func (ts *TransactionStreamer) Stop() {
	close(ts.stopChan)
}

// pollMempool polls the mempool for new transactions
func (ts *TransactionStreamer) pollMempool(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Get mempool content
			var result map[string]map[string]map[string]*types.Transaction
			err := ts.rpcClient.Call(&result, "txpool_content")
			if err != nil {
				fmt.Printf("Error getting mempool content: %v\n", err)
				continue
			}

			// Process queued transactions
			for _, account := range result["queued"] {
				for _, tx := range account {
					ts.processNewTransaction(ctx, tx)
				}
			}

			// Process pending transactions
			for _, account := range result["pending"] {
				for _, tx := range account {
					ts.processNewTransaction(ctx, tx)
				}
			}
		case <-ctx.Done():
			return
		case <-ts.stopChan:
			return
		}
	}
}

// processNewTransaction processes a new transaction from the mempool
func (ts *TransactionStreamer) processNewTransaction(ctx context.Context, tx *types.Transaction) {
	// Check if we already have this transaction
	ts.mu.RLock()
	_, exists := ts.txStore[tx.Hash().Hex()]
	ts.mu.RUnlock()

	if exists {
		return
	}

	// Get full transaction details
	_, _, err := ts.client.TransactionByHash(ctx, tx.Hash())
	if err != nil {
		fmt.Printf("Error getting transaction details: %v\n", err)
		return
	}

	// Get sender address
	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err != nil {
		fmt.Printf("Error getting sender address: %v\n", err)
		return
	}

	// Create new stored transaction
	storedTx := &StoredTransaction{
		Hash: tx.Hash().Hex(),
		Metadata: TransactionMetadata{
			Status:       StatusQueued,
			TimeReceived: time.Now().Unix(),
			Type:         getTransactionType(tx),
			Nonce:        tx.Nonce(),
			From:         sender.Hex(),
			To:           tx.To().Hex(),
			Timestamp:    time.Now().Unix(),
		},
	}

	// Set gas price based on transaction type
	switch tx.Type() {
	case types.DynamicFeeTxType:
		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
	case types.BlobTxType:
		storedTx.Metadata.MaxFeePerGas = tx.GasFeeCap()
		storedTx.Metadata.MaxPriorityFee = tx.GasTipCap()
		storedTx.Metadata.MaxFeePerBlobGas = tx.BlobGasFeeCap()
	default:
		storedTx.Metadata.GasPrice = tx.GasPrice()
	}

	// Store transaction
	ts.mu.Lock()
	ts.txStore[storedTx.Hash] = storedTx
	ts.mu.Unlock()
}

// trackTransactionStatus tracks the status of transactions
func (ts *TransactionStreamer) trackTransactionStatus(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ts.mu.RLock()
			for _, tx := range ts.txStore {
				// Skip if already mined or dropped
				if tx.Metadata.Status == StatusMined || tx.Metadata.Status == StatusDropped {
					continue
				}

				// Check if transaction is pending
				if tx.Metadata.Status == StatusQueued {
					var result map[string]map[string]map[string]*types.Transaction
					err := ts.rpcClient.Call(&result, "txpool_content")
					if err == nil {
						// Check if transaction is in pending pool
						for _, account := range result["pending"] {
							if _, exists := account[tx.Hash]; exists {
								ts.mu.RUnlock()
								ts.mu.Lock()
								tx.Metadata.Status = StatusPending
								tx.Metadata.TimePending = time.Now().Unix()
								ts.mu.Unlock()
								ts.mu.RLock()
								break
							}
						}
					}
				}

				// Check if transaction is mined
				if tx.Metadata.Status == StatusPending {
					hash := common.HexToHash(tx.Hash)
					receipt, err := ts.client.TransactionReceipt(ctx, hash)
					if err == nil && receipt != nil {
						ts.mu.RUnlock()
						ts.mu.Lock()
						tx.Metadata.Status = StatusMined
						tx.Metadata.TimeMined = time.Now().Unix()
						tx.Metadata.BlockNumber = receipt.BlockNumber.Uint64()
						tx.Metadata.BlockHash = receipt.BlockHash.Hex()
						ts.mu.Unlock()
						ts.mu.RLock()
					}
				}
			}
			ts.mu.RUnlock()
		case <-ctx.Done():
			return
		case <-ts.stopChan:
			return
		}
	}
}

// GetTransaction returns a stored transaction by hash
func (ts *TransactionStreamer) GetTransaction(hash string) (*StoredTransaction, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tx, exists := ts.txStore[hash]
	return tx, exists
}

// GetAllTransactions returns all stored transactions
func (ts *TransactionStreamer) GetAllTransactions() []*StoredTransaction {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	txs := make([]*StoredTransaction, 0, len(ts.txStore))
	for _, tx := range ts.txStore {
		txs = append(txs, tx)
	}
	return txs
}

// getTransactionType determines the type of transaction
func getTransactionType(tx *types.Transaction) TransactionType {
	switch {
	case tx.Type() == types.BlobTxType:
		return BlobTx
	case tx.Type() == types.DynamicFeeTxType:
		return EIP1559Tx
	case tx.Type() == types.AccessListTxType:
		return EIP2930Tx
	default:
		return LegacyTx
	}
}
