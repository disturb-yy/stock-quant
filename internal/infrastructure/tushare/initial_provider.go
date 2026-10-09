package tushare

import (
	"context"
	"errors"
	"fmt"
	"time"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/market/ports"
	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

const (
	initialStockBasicMaxRows = 6000
	stockBasicFields         = "ts_code,symbol,name,area,industry,cnspell,market,exchange,list_status,list_date,delist_date"
	tradeCalendarFields      = "exchange,cal_date,is_open,pretrade_date"
)

// InitialMarketDataProvider 查询初始股票身份和交易日历。
type InitialMarketDataProvider struct {
	client *Client
}

var _ ports.InitialMarketDataProvider = (*InitialMarketDataProvider)(nil)

// NewInitialMarketDataProvider 创建初始市场数据适配器。
func NewInitialMarketDataProvider(client *Client) *InitialMarketDataProvider {
	return &InitialMarketDataProvider{client: client}
}

// StockBasics 查询一个上市状态快照，并映射其中的每条记录。
func (provider *InitialMarketDataProvider) StockBasics(ctx context.Context, listStatus string) ([]domain.Stock, error) {
	if provider == nil || provider.client == nil {
		return nil, errors.New("fetch stock basics: Tushare client is required")
	}
	if listStatus != "L" && listStatus != "D" && listStatus != "P" {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("list status must be L, D, or P"))
	}
	rows, err := provider.client.Query(ctx, QueryRequest{
		APIName: "stock_basic", Params: map[string]any{"list_status": listStatus}, Fields: stockBasicFields,
		RequiredFields: []string{"ts_code", "symbol", "name", "area", "industry", "cnspell", "market", "exchange", "list_status", "list_date", "delist_date"},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch stock basics: %w", err)
	}
	if len(rows) > initialStockBasicMaxRows {
		return nil, apperror.New(apperror.CodeDataIncomplete, errors.New("stock_basic response exceeds the 6000-row limit"))
	}
	updatedAt := time.Now().UTC()
	stocks := make([]domain.Stock, 0, len(rows))
	for index, row := range rows {
		stock, mapErr := MapStockBasic(row, MappingMetadata{UpdatedAt: updatedAt})
		if mapErr != nil {
			return nil, fmt.Errorf("map stock_basic row %d: %w", index, mapErr)
		}
		if stock.ListStatus != listStatus {
			return nil, apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("stock_basic row %d has a different list status", index))
		}
		stocks = append(stocks, stock)
	}
	return stocks, nil
}

// TradeCalendars 按自然年切片查询指定范围的交易日历。
func (provider *InitialMarketDataProvider) TradeCalendars(ctx context.Context, exchange string, from, through types.TradingDate) ([]domain.TradeCalendar, error) {
	if provider == nil || provider.client == nil {
		return nil, errors.New("fetch trade calendars: Tushare client is required")
	}
	if exchange != "SSE" && exchange != "SZSE" {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("exchange must be SSE or SZSE"))
	}
	if !from.Valid() || !through.Valid() || from.String() > through.String() {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("calendar date range must be valid and ordered"))
	}
	calendars := make([]domain.TradeCalendar, 0)
	for year := yearOf(from); year <= yearOf(through); year++ {
		chunkFrom, chunkThrough, err := calendarYearRange(year, from, through)
		if err != nil {
			return nil, fmt.Errorf("build trade calendar range: %w", err)
		}
		chunk, err := provider.fetchCalendarChunk(ctx, exchange, chunkFrom, chunkThrough)
		if err != nil {
			return nil, err
		}
		calendars = append(calendars, chunk...)
	}
	return calendars, nil
}

func (provider *InitialMarketDataProvider) fetchCalendarChunk(ctx context.Context, exchange string, from, through types.TradingDate) ([]domain.TradeCalendar, error) {
	start, err := from.TushareString()
	if err != nil {
		return nil, fmt.Errorf("format trade calendar start date: %w", err)
	}
	end, err := through.TushareString()
	if err != nil {
		return nil, fmt.Errorf("format trade calendar end date: %w", err)
	}
	rows, err := provider.client.Query(ctx, QueryRequest{
		APIName: "trade_cal", Params: map[string]any{"exchange": exchange, "start_date": start, "end_date": end},
		Fields: tradeCalendarFields, RequiredFields: []string{"exchange", "cal_date", "is_open", "pretrade_date"},
	})
	if err != nil {
		return nil, fmt.Errorf("fetch %s trade calendar %s-%s: %w", exchange, from, through, err)
	}
	calendars := make([]domain.TradeCalendar, 0, len(rows))
	for index, row := range rows {
		calendar, mapErr := MapTradeCalendar(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map trade_cal row %d: %w", index, mapErr)
		}
		if calendar.Exchange != exchange {
			return nil, apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("trade_cal row %d has a different exchange", index))
		}
		calendars = append(calendars, calendar)
	}
	return calendars, nil
}

func yearOf(date types.TradingDate) int {
	value := date.String()
	return int(value[0]-'0')*1000 + int(value[1]-'0')*100 + int(value[2]-'0')*10 + int(value[3]-'0')
}

func calendarYearRange(year int, from, through types.TradingDate) (types.TradingDate, types.TradingDate, error) {
	start, err := types.ParseTradingDate(fmt.Sprintf("%04d-01-01", year))
	if err != nil {
		return types.TradingDate{}, types.TradingDate{}, err
	}
	end, err := types.ParseTradingDate(fmt.Sprintf("%04d-12-31", year))
	if err != nil {
		return types.TradingDate{}, types.TradingDate{}, err
	}
	if from.String() > start.String() {
		start = from
	}
	if through.String() < end.String() {
		end = through
	}
	return start, end, nil
}
