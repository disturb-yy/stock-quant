package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/health"
	"github.com/disturb-yy/stock-quant/internal/market"
	"github.com/disturb-yy/stock-quant/internal/stock"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/disturb-yy/stock-quant/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	apiV1Prefix = "/api/v1"
)

func newRouter(applicationLogger *slog.Logger, statusReaders ...demo.StatusReader) *gin.Engine {
	return newRouterWithMarket(applicationLogger, nil, nil, statusReaders...)
}

func newRouterWithOverview(applicationLogger *slog.Logger, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, statusReaders ...demo.StatusReader) *gin.Engine {
	return newRouterWithMarket(applicationLogger, overviewReader, nil, statusReaders...)
}

func newRouterWithMarket(applicationLogger *slog.Logger, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, statusReaders ...demo.StatusReader) *gin.Engine {
	return newRouterWithMarketAndSignals(applicationLogger, overviewReader, sectorReader, nil, statusReaders...)
}

func newRouterWithMarketAndSignals(applicationLogger *slog.Logger, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, statusReaders ...demo.StatusReader) *gin.Engine {
	return newRouterWithMarketSignalsAndRankings(applicationLogger, overviewReader, sectorReader, signalReader, nil, statusReaders...)
}

func newRouterWithMarketSignalsAndRankings(applicationLogger *slog.Logger, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, rankingReader interface {
	Rank(context.Context, market.RankingRequest) (market.MarketRankings, error)
}, statusReaders ...demo.StatusReader) *gin.Engine {
	return newRouterWithMarketSignalsAndRankingsAndStocks(applicationLogger, overviewReader, sectorReader, signalReader, rankingReader, nil, statusReaders...)
}

func newRouterWithMarketSignalsAndRankingsAndStocks(applicationLogger *slog.Logger, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, rankingReader interface {
	Rank(context.Context, market.RankingRequest) (market.MarketRankings, error)
}, stockOverviewReader interface {
	Overview(context.Context, string) (stock.StockOverview, error)
}, statusReaders ...demo.StatusReader) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(logger.GinMiddleware(applicationLogger), gin.CustomRecovery(apiV1RecoveryHandler))

	apiV1 := router.Group(apiV1Prefix)
	includeDevelopment := len(statusReaders) > 0 && statusReaders[0] != nil
	apiV1.GET("/openapi.json", func(context *gin.Context) {
		openAPIHandler(context, includeDevelopment)
	})
	health.RegisterRoutes(apiV1)
	market.RegisterRoutes(apiV1, overviewReader)
	market.RegisterSectorRoutes(apiV1, sectorReader)
	market.RegisterSignalRoutes(apiV1, signalReader)
	market.RegisterRankingRoutes(apiV1, rankingReader)
	stock.RegisterOverviewRoutes(apiV1, stockOverviewReader)
	if len(statusReaders) > 0 {
		demo.RegisterRoutes(apiV1, statusReaders[0])
	}

	router.NoRoute(apiV1NoRouteHandler)
	router.NoMethod(apiV1NoMethodHandler)
	return router
}

func newHTTPServer(applicationLogger *slog.Logger, address string, statusReaders ...demo.StatusReader) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: newRouter(applicationLogger, statusReaders...),
	}
}

func newHTTPServerWithOverview(applicationLogger *slog.Logger, address string, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, statusReaders ...demo.StatusReader) *http.Server {
	return newHTTPServerWithMarket(applicationLogger, address, overviewReader, nil, statusReaders...)
}

func newHTTPServerWithMarket(applicationLogger *slog.Logger, address string, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, statusReaders ...demo.StatusReader) *http.Server {
	return newHTTPServerWithMarketAndSignals(applicationLogger, address, overviewReader, sectorReader, nil, statusReaders...)
}

func newHTTPServerWithMarketAndSignals(applicationLogger *slog.Logger, address string, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, statusReaders ...demo.StatusReader) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: newRouterWithMarketAndSignals(applicationLogger, overviewReader, sectorReader, signalReader, statusReaders...),
	}
}

func newHTTPServerWithMarketSignalsAndRankings(applicationLogger *slog.Logger, address string, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, rankingReader interface {
	Rank(context.Context, market.RankingRequest) (market.MarketRankings, error)
}, statusReaders ...demo.StatusReader) *http.Server {
	return newHTTPServerWithMarketSignalsAndRankingsAndStocks(applicationLogger, address, overviewReader, sectorReader, signalReader, rankingReader, nil, statusReaders...)
}

func newHTTPServerWithMarketSignalsAndRankingsAndStocks(applicationLogger *slog.Logger, address string, overviewReader interface {
	Overview(context.Context) (market.MarketOverview, error)
}, sectorReader interface {
	Sectors(context.Context) (market.MarketSectors, error)
}, signalReader interface {
	Scan(context.Context, market.SignalRequest) (market.MarketSignals, error)
}, rankingReader interface {
	Rank(context.Context, market.RankingRequest) (market.MarketRankings, error)
}, stockOverviewReader interface {
	Overview(context.Context, string) (stock.StockOverview, error)
}, statusReaders ...demo.StatusReader) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: newRouterWithMarketSignalsAndRankingsAndStocks(applicationLogger, overviewReader, sectorReader, signalReader, rankingReader, stockOverviewReader, statusReaders...),
	}
}

func openAPIHandler(context *gin.Context, includeDevelopment bool) {
	context.JSON(http.StatusOK, api.OpenAPIDocument(includeDevelopment))
}

func apiV1RecoveryHandler(context *gin.Context, _ any) {
	if isAPIPath(context.Request.URL.Path) {
		writeAPIError(context, http.StatusInternalServerError, api.CodeInternal, "服务内部错误", nil)
		return
	}

	context.AbortWithStatus(http.StatusInternalServerError)
}

func validatePaginationQuery() gin.HandlerFunc {
	// 仅由实际的列表路由显式挂载，避免影响 health 和 OpenAPI 等非分页接口。
	return func(context *gin.Context) {
		if _, err := api.ParsePagination(context.Request.URL.Query()); err != nil {
			var paginationError *api.PaginationValidationError
			if errors.As(err, &paginationError) {
				writeAPIError(context, http.StatusBadRequest, api.CodeInvalidPagination, "分页参数无效", paginationError.Details())
				return
			}

			writeAPIError(context, http.StatusBadRequest, api.CodeValidation, "请求参数校验失败", nil)
			return
		}

		context.Next()
	}
}

func apiV1NoRouteHandler(context *gin.Context) {
	if isAPIPath(context.Request.URL.Path) {
		writeAPIError(context, http.StatusNotFound, api.CodeNotFound, "请求的资源不存在", nil)
		return
	}

	context.Status(http.StatusNotFound)
}

func apiV1NoMethodHandler(context *gin.Context) {
	if isAPIPath(context.Request.URL.Path) {
		writeAPIError(context, http.StatusMethodNotAllowed, api.CodeMethodNotAllowed, "请求方法不被允许", nil)
		return
	}

	context.Status(http.StatusMethodNotAllowed)
}

func writeAPIError(context *gin.Context, status int, code api.ErrorCode, message string, details map[string]any) {
	context.AbortWithStatusJSON(status, api.NewErrorResponse(code, message, details))
}

func isAPIPath(path string) bool {
	return path == apiV1Prefix || strings.HasPrefix(path, apiV1Prefix+"/")
}
