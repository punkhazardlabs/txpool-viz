package handler

import (
	"net/http"
	"txpool-viz/internal/model"
	"txpool-viz/internal/service"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests
type Handler struct {
	TxService *service.TransactionServiceImpl
}

func (h *Handler) GetLatestTransactions(c *gin.Context) {
	var txCount model.CountArgs
	if err := c.ShouldBindJSON(&txCount); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid range parameters"})
		return
	}

	ctx := c.Request.Context()
	txs, err := h.TxService.GetLatestNTransactions(ctx, txCount.TxCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, txs)
}

func NewHandler(service *service.TransactionServiceImpl) *Handler {
	return &Handler{
		TxService: service,
	}
}
