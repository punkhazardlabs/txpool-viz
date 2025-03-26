package routes

import (
	"net/http"
	"txpool-viz/internal/controller/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the routes for the application
func RegisterRoutes(router *gin.Engine, handler *handler.Handler) {
	router.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	router.GET("/transactions", handler.TransactionService.GetLatestTransactions)
}
