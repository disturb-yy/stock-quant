package main

import (
	datahttp "github.com/disturb-yy/stock-quant/internal/data/interface/http"
	"github.com/disturb-yy/stock-quant/internal/health"
	"github.com/gin-gonic/gin"
)

const apiV1Prefix = "/api/v1"

func newRouter(services ...datahttp.Service) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group(apiV1Prefix)
	health.RegisterRoutes(api)
	if len(services) > 0 && services[0] != nil {
		datahttp.RegisterRoutes(api, services[0])
		if queryService, ok := services[0].(datahttp.StockQueryService); ok {
			datahttp.RegisterStockQueryRoutes(api, queryService)
		}
	}
	return router
}
