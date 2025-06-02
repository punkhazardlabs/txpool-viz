package handler

import (
	"net/http"
	"strconv"
	"txpool-viz/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	TxService            *service.TransactionServiceImpl
	InclusionListService *service.InclusionListService
}

const DefaultTxCount = 100000

func NewHandler(txService *service.TransactionServiceImpl, ilService *service.InclusionListService) *Handler {
	return &Handler{
		TxService:            txService,
		InclusionListService: ilService,
	}
}

func (h *Handler) GetLatestTransactions(c *gin.Context) {
	txCountStr := c.DefaultQuery("tx_count", strconv.Itoa(DefaultTxCount))
	txCount, err := strconv.Atoi(txCountStr)
	if err != nil || txCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tx_count parameter"})
		return
	}

	ctx := c.Request.Context()
	txs, err := h.TxService.GetLatestNTransactions(ctx, int64(txCount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, txs)
}

func (h *Handler) GetTransactionDetails(c *gin.Context) {
	txHash := c.Param("txHash")
	if txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing transaction hash"})
		return
	}

	ctx := c.Request.Context()
	details, err := h.TxService.GetTxDetails(ctx, txHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, details)
}

func (h *Handler) GetInclusionLists(c *gin.Context) {
	ctx := c.Request.Context()

	inclusionReports, err := h.InclusionListService.GetInclusionLists(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, inclusionReports)
}
