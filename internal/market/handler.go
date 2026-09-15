package market

import (
	"context"
	"errors"
	"net/http"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	overviewPath = "/markets/overview"
	sectorsPath  = "/markets/sectors"
	signalsPath  = "/markets/signals"
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

// RegisterSignalRoutes 注册市场信号扫描 HTTP 路由。
func RegisterSignalRoutes(router *gin.RouterGroup, scanner interface {
	Scan(context.Context, SignalRequest) (MarketSignals, error)
}) {
	if scanner == nil {
		return
	}
	router.GET(signalsPath, func(context *gin.Context) {
		request, err := signalRequestFromQuery(context)
		if err != nil {
			var validationError *SignalValidationError
			if errors.As(err, &validationError) {
				context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "信号参数无效", validationError.Details()))
				return
			}
			context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "请求参数校验失败", nil))
			return
		}

		result, err := scanner.Scan(context.Request.Context(), request)
		if err != nil {
			writeSignalError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

func signalRequestFromQuery(context *gin.Context) (SignalRequest, error) {
	values := context.Request.URL.Query()
	typeValues, ok := values["type"]
	if !ok || len(typeValues) != 1 {
		return SignalRequest{}, &SignalValidationError{Fields: map[string]string{"type": "必须提供且只能提供一次"}}
	}
	paramsValues, ok := values["params"]
	if !ok {
		return SignalRequest{Type: typeValues[0]}, nil
	}
	if len(paramsValues) != 1 {
		return SignalRequest{}, &SignalValidationError{Fields: map[string]string{"params": "只能提供一次"}}
	}
	return SignalRequest{Type: typeValues[0], Params: paramsValues[0]}, nil
}

func writeSignalError(context *gin.Context, err error) {
	var validationError *SignalValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "信号参数无效", validationError.Details()))
		return
	}
	var historyError *SignalHistoryError
	if errors.As(err, &historyError) {
		context.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponse(api.CodeInsufficientHistory, "历史行情不足", historyError.Details()))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "市场信号数据暂不可用", nil))
}
