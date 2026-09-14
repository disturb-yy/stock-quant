package market

import (
	"context"
	"net/http"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	overviewPath = "/markets/overview"
	sectorsPath  = "/markets/sectors"
)

// RegisterRoutes 注册市场概览 HTTP 路由。
func RegisterRoutes(router *gin.RouterGroup, reader interface {
	Overview(context.Context) (MarketOverview, error)
}) {
	if reader == nil {
		return
	}
	router.GET(overviewPath, func(context *gin.Context) {
		overview, err := reader.Overview(context.Request.Context())
		if err != nil {
			context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(
				api.CodeDependencyUnavailable,
				"市场概览数据暂不可用",
				nil,
			))
			return
		}
		context.JSON(http.StatusOK, overview)
	})
}

// RegisterSectorRoutes 注册行业表现 HTTP 路由。
func RegisterSectorRoutes(router *gin.RouterGroup, reader interface {
	Sectors(context.Context) (MarketSectors, error)
}) {
	if reader == nil {
		return
	}
	router.GET(sectorsPath, func(context *gin.Context) {
		sectors, err := reader.Sectors(context.Request.Context())
		if err != nil {
			context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(
				api.CodeDependencyUnavailable,
				"行业数据暂不可用",
				nil,
			))
			return
		}
		context.JSON(http.StatusOK, sectors)
	})
}
