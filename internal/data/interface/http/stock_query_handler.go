package http

import (
	"context"
	"net/http"

	"github.com/disturb-yy/stock-quant/internal/data/application"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
	"github.com/gin-gonic/gin"
)

type StockQueryService interface {
	QueryStockData(context.Context, application.StockQueryInput) (domain.StockQueryResult, error)
}

var _ StockQueryService = (*application.Service)(nil)

type StockQueryHandler struct {
	service StockQueryService
}

func RegisterStockQueryRoutes(router *gin.RouterGroup, service StockQueryService) {
	if router == nil || service == nil {
		return
	}
	handler := &StockQueryHandler{service: service}
	router.GET("/stocks/:symbol/data", handler.get)
}

type stockDataResponse struct {
	Symbol       string                    `json:"symbol"`
	BasicInfo    *stockBasicInfoResponse   `json:"basic_info"`
	DailyBars    []stockDailyBarResponse   `json:"daily_bars"`
	Availability stockAvailabilityResponse `json:"availability"`
	Source       *sourceResponse           `json:"source"`
	UpdatedAt    *string                   `json:"updated_at"`
	DataAsOf     *string                   `json:"data_as_of"`
}

type stockBasicInfoResponse struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Market string `json:"market"`
	Status string `json:"status"`
}

type stockDailyBarResponse struct {
	TradeDate string  `json:"trade_date"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
}

type stockAvailabilityResponse struct {
	BasicInfo string `json:"basic_info"`
	DailyBars string `json:"daily_bars"`
}

func (handler *StockQueryHandler) get(ctx *gin.Context) {
	input := application.StockQueryInput{
		Symbol:    ctx.Param("symbol"),
		StartDate: optionalQuery(ctx, "start_date"),
		EndDate:   optionalQuery(ctx, "end_date"),
	}
	result, err := handler.service.QueryStockData(ctx.Request.Context(), input)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toStockDataResponse(result))
}

func optionalQuery(ctx *gin.Context, name string) *string {
	value, exists := ctx.GetQuery(name)
	if !exists {
		return nil
	}
	return &value
}

func toStockDataResponse(result domain.StockQueryResult) stockDataResponse {
	response := stockDataResponse{
		Symbol:    result.Symbol,
		DailyBars: make([]stockDailyBarResponse, 0, len(result.DailyBars)),
		Availability: stockAvailabilityResponse{
			BasicInfo: result.Availability.BasicInfo,
			DailyBars: result.Availability.DailyBars,
		},
		Source:    sourceResponseFor(result.Source),
		UpdatedAt: timeString(result.UpdatedAt),
		DataAsOf:  dateString(result.DataAsOf),
	}
	if result.BasicInfo != nil {
		response.BasicInfo = &stockBasicInfoResponse{
			Symbol: result.Symbol,
			Name:   result.BasicInfo.Name,
			Market: result.BasicInfo.Market,
			Status: result.BasicInfo.Status,
		}
	}
	for _, bar := range result.DailyBars {
		response.DailyBars = append(response.DailyBars, stockDailyBarResponse{
			TradeDate: bar.TradeDate.UTC().Format(domain.DateLayout), Open: bar.Open, High: bar.High,
			Low: bar.Low, Close: bar.Close, Volume: bar.Volume,
		})
	}
	return response
}

func sourceResponseFor(source *domain.SourceProvenance) *sourceResponse {
	if source == nil {
		return nil
	}
	return &sourceResponse{Provider: source.Provider, Mode: source.Mode}
}
