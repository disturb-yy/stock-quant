package stock

import (
	"context"
	"errors"
	"net/http"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const overviewPath = "/stocks/:symbol"

// RegisterOverviewRoutes 注册股票详情概览 HTTP 路由。
func RegisterOverviewRoutes(router *gin.RouterGroup, reader interface {
	Overview(context.Context, string) (StockOverview, error)
}) {
	if reader == nil {
		return
	}
	router.GET(overviewPath, func(context *gin.Context) {
		overview, err := reader.Overview(context.Request.Context(), context.Param("symbol"))
		if err != nil {
			writeOverviewError(context, err)
			return
		}
		context.JSON(http.StatusOK, overview)
	})
}

func writeOverviewError(context *gin.Context, err error) {
	var validationError *SymbolValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "股票代码参数无效", validationError.Details()))
		return
	}
	if errors.Is(err, ErrInstrumentNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票详情数据暂不可用", nil))
}
