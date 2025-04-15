package pkg

import (
	"fmt"
	"txpool-viz/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// getTransactionType determines the type of transaction
func GetTransactionType(tx *types.Transaction) model.TransactionType {
	switch {
	case tx.Type() == types.BlobTxType:
		return model.BlobTx
	case tx.Type() == types.DynamicFeeTxType:
		return model.EIP1559Tx
	case tx.Type() == types.AccessListTxType:
		return model.EIP2930Tx
	default:
		return model.LegacyTx
	}
}

func GetTxKey(tx *types.Transaction, addr common.Address) string {
	txKey := fmt.Sprintf("%s:%d", addr, tx.Nonce())
	return txKey
}
