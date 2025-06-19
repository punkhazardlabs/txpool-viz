package service

import (
	"fmt"
	"txpool-viz/internal/model"
)

type Result struct {
	Common map[string]interface{}            // fields identical across clients
	Diff   map[string]map[string]interface{} // fields that differ
}

// toMapTx flattens the Tx struct into a field→value map.
func toMapTx(t model.Tx) map[string]interface{} {
	m := map[string]interface{}{
		"chain_id":           t.ChainID,
		"from":               t.From,
		"to":                 t.To,
		"isContractCreation": t.IsContractCreation,
		"nonce":              t.Nonce,
		"value":              t.Value,
		"gas":                t.Gas,
		"type":               model.TransactionType(t.Type).String(),
		"data":               t.Data,
		"maxFeePerGas":       t.MaxFeePerGas,
		"maxPriorityFee":     t.MaxPriorityFee,
	}
	if t.GasPrice != nil {
		m["gasPrice"] = t.GasPrice.String()
	}
	if t.MaxFeePerBlobGas != "" {
		m["maxFeePerBlobGas"] = t.MaxFeePerBlobGas
	}
	return m
}

// toMapMeta flattens the TransactionMetadata into field→value.
func toMapMeta(md model.TransactionMetadata) map[string]interface{} {
	m := map[string]interface{}{
		"status":       string(md.Status),
		"mineStatus":   md.MineStatus,
		"gasUsed":      md.GasUsed,
		"blockNumber":  md.BlockNumber,
		"timeReceived": md.TimeReceived,
		"timeQueued":   md.TimeQueued,
		"blockHash":    md.BlockHash,
	}
	if md.TimePending != nil {
		m["timePending"] = *md.TimePending
	}
	if md.TimeMined != nil {
		m["timeMined"] = *md.TimeMined
	}
	m["timeDropped"] = md.TimeDropped
	return m
}

// Compute compares transaction data across multiple clients and returns a Result
func Compute(all map[string]map[string]interface{}, primaryClient string) Result {
	diff := diffMaps(all)

	base, ok := all[primaryClient]
	common := make(map[string]interface{}, len(base))
	if ok {
		for field, val := range base {
			if _, isDiff := diff[field]; !isDiff {
				common[field] = val
			}
		}
	}

	return Result{Common: common, Diff: diff}
}

// diffMaps returns only the fields whose values across clients differ.
func diffMaps(all map[string]map[string]interface{}) map[string]map[string]interface{} {
	// collect all field names
	fields := make(map[string]struct{})
	for _, m := range all {
		for k := range m {
			fields[k] = struct{}{}
		}
	}

	out := make(map[string]map[string]interface{}, len(fields))
	for field := range fields {
		// gather each client’s value, track unique
		vals := make(map[string]interface{}, len(all))
		uniq := make(map[string]struct{})
		for client, m := range all {
			v := m[field]
			vals[client] = v
			// check uniqueness
			uniq[fmt.Sprint(v)] = struct{}{}
		}
		if len(uniq) > 1 {
			out[field] = vals
		}
	}
	return out
}
