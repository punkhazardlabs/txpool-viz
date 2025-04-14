package pkg

import (
	"txpool-viz/internal/model"

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
