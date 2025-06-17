package utils

import (
	"fmt"
	"txpool-viz/internal/model"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// getTransactionType determines the type of transaction
func GetTransactionType(tx *types.Transaction) model.TransactionType {
	switch {
	case tx.Type() == types.LegacyTxType:
		return model.LegacyTxType
	case tx.Type() == types.AccessListTxType:
		return model.AccessListTxType
	case tx.Type() == types.DynamicFeeTxType:
		return model.DynamicFeeTxType
	case tx.Type() == types.BlobTxType:
		return model.BlobTxType
	case tx.Type() == types.SetCodeTxType:
		return model.SetCodeTxType
	default:
		return model.LegacyTxType
	}
}

// Given a transaction and the sender address
// It returns the transactions unique id -> txhash:nonce
func GetTxKey(tx *types.Transaction, addr common.Address) string {
	txKey := fmt.Sprintf("%s:%d", addr, tx.Nonce())
	return txKey
}

