package routes

import (
	"net/http"
	"txpool-viz/internal/controller/handler"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the API routes for the application
func RegisterRoutes(router *gin.Engine, handler *handler.Handler) {
	api := router.Group("/")

	api.GET("/ping", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "pong")
	})

	api.GET("/transactions", handler.GetLatestTransactions)
	api.GET("/transaction/:txHash", handler.GetTransactionDetails)
}
