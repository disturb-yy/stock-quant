package demo

import (
	"net/http"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const demoStatusPath = "/dev/demo-status"

// RegisterRoutes 只向开发环境路由分组注册演示状态接口。
func RegisterRoutes(router *gin.RouterGroup, reader StatusReader) {
	if reader == nil {
		return
	}
	router.GET(demoStatusPath, func(context *gin.Context) {
		status, err := reader.DemoStatus(context.Request.Context())
		if err != nil {
			context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(
				api.CodeDependencyUnavailable,
				"演示数据存储不可用",
				nil,
			))
			return
		}
		context.JSON(http.StatusOK, status)
	})
}
