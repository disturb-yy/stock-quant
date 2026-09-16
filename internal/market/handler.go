package market

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	overviewPath = "/markets/overview"
	sectorsPath  = "/markets/sectors"
	signalsPath  = "/markets/signals"
	rankingsPath = "/markets/rankings"
	barsPath     = "/stocks/:symbol/bars"
)

// RegisterBarsRoutes 注册股票研究型日线 HTTP 路由。
func RegisterBarsRoutes(router *gin.RouterGroup, query interface {
	Bars(context.Context, BarsRequest) (StockBars, error)
}) {
	if query == nil {
		return
	}
	router.GET(barsPath, func(context *gin.Context) {
		request, err := barsRequestFromQuery(context)
		if err != nil {
			writeBarsError(context, err)
			return
		}
		request.Symbol = context.Param("symbol")
		request, err = normalizeBarsRequest(request)
		if err != nil {
			writeBarsError(context, err)
			return
		}
		result, err := query.Bars(context.Request.Context(), request)
		if err != nil {
			writeBarsError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

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

// RegisterRankingRoutes 注册股票排行 HTTP 路由。
func RegisterRankingRoutes(router *gin.RouterGroup, query interface {
	Rank(context.Context, RankingRequest) (MarketRankings, error)
}) {
	if query == nil {
		return
	}
	router.GET(rankingsPath, func(context *gin.Context) {
		request, err := rankingRequestFromQuery(context)
		if err != nil {
			writeRankingRequestError(context, err)
			return
		}
		result, err := query.Rank(context.Request.Context(), request)
		if err != nil {
			writeRankingError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

func rankingRequestFromQuery(context *gin.Context) (RankingRequest, error) {
	values := context.Request.URL.Query()
	metricValues, ok := values["metric"]
	if !ok || len(metricValues) != 1 || strings.TrimSpace(metricValues[0]) == "" {
		return RankingRequest{}, &RankingValidationError{Fields: map[string]string{"metric": "必须提供且只能提供一次"}}
	}
	pagination, err := api.ParsePagination(values)
	if err != nil {
		return RankingRequest{}, err
	}
	return RankingRequest{Metric: metricValues[0], Page: pagination.Page, PageSize: pagination.PageSize}, nil
}

func writeRankingRequestError(context *gin.Context, err error) {
	var paginationError *api.PaginationValidationError
	if errors.As(err, &paginationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeInvalidPagination, "分页参数无效", paginationError.Details()))
		return
	}
	var validationError *RankingValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "排行参数无效", validationError.Details()))
		return
	}
	context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "请求参数校验失败", nil))
}

func writeRankingError(context *gin.Context, err error) {
	var validationError *RankingValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "排行参数无效", validationError.Details()))
		return
	}
	var historyError *RankingHistoryError
	if errors.As(err, &historyError) {
		context.AbortWithStatusJSON(http.StatusUnprocessableEntity, api.NewErrorResponse(api.CodeInsufficientHistory, "历史行情不足", historyError.Details()))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票排行数据暂不可用", nil))
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

func barsRequestFromQuery(context *gin.Context) (BarsRequest, error) {
	values := context.Request.URL.Query()
	request := BarsRequest{}
	for _, field := range []struct {
		name   string
		target *string
	}{
		{name: "timeframe", target: &request.Timeframe},
		{name: "adjust", target: &request.Adjust},
		{name: "range", target: &request.Range},
		{name: "from", target: &request.From},
		{name: "to", target: &request.To},
		{name: "benchmark", target: &request.Benchmark},
	} {
		queryValues, ok := values[field.name]
		if !ok {
			continue
		}
		if len(queryValues) != 1 || strings.TrimSpace(queryValues[0]) == "" {
			return BarsRequest{}, newBarsValidationError(field.name, "必须提供且只能提供一次")
		}
		*field.target = queryValues[0]
	}
	return request, nil
}

func writeBarsError(context *gin.Context, err error) {
	var validationError *BarsValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "股票行情参数无效", validationError.Details()))
		return
	}
	if errors.Is(err, ErrBarsInstrumentNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票行情数据暂不可用", nil))
}

func newBarsValidationError(field, message string) *BarsValidationError {
	return &BarsValidationError{Fields: map[string]string{field: message}}
}
