package service

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"txpool-viz/internal/logger"
	"txpool-viz/internal/model"
	"txpool-viz/utils"

	"github.com/redis/go-redis/v9"
)

type InclusionListService struct {
	redis  *redis.Client
	logger logger.Logger
	enabled bool
}

func NewInclusionListService(r *redis.Client, l logger.Logger, focilEnabled bool) *InclusionListService {
	return &InclusionListService{
		redis:  r,
		logger: l,
		enabled: focilEnabled,
	}
}

func (il *InclusionListService) GetInclusionLists(ctx context.Context) ([]model.InclusionListWithSlot, error) {
	inclusionReportKey := utils.RedisInclusionListReportKey()

	results, err := il.redis.HGetAll(ctx, inclusionReportKey).Result()
	if err != nil {
		return nil, err
	}

	var sortedReports []model.InclusionListWithSlot

	for slotStr, reportJSON := range results {
		slot, err := strconv.Atoi(slotStr)
		if err != nil {
			il.logger.Error("Invalid slot key", slotStr)
			continue
		}

		var report model.InclusionReport
		if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
			il.logger.Error("Invalid entry", err.Error())
			continue
		}

		sortedReports = append(sortedReports, model.InclusionListWithSlot{
			Slot:   slot,
			Report: report,
		})
	}

	sort.Slice(sortedReports, func(i, j int) bool {
		return sortedReports[i].Slot > sortedReports[j].Slot
	})

	return sortedReports, nil
}

// IsFocilEnabled checks if the Focil feature is enabled
func (il *InclusionListService) IsFocilEnabled() bool {
	return il.enabled
}