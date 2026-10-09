package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/market/ports"
	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

var errConflictingInitialStock = errors.New("conflicting stock identity across listing states")
var errConflictingInitialCalendar = errors.New("conflicting calendar rows for the same exchange date")

// InitialMarketSyncService 导入历史股票身份与交易所日历。
type InitialMarketSyncService struct {
	provider  ports.InitialMarketDataProvider
	stocks    ports.StockRepository
	calendars ports.TradeCalendarRepository
}

// InitialMarketSyncResult 汇总一次初始同步写入的记录数。
type InitialMarketSyncResult struct {
	StockCount    int
	CalendarCount int
}

// NewInitialMarketSync 创建初始市场数据同步用例。
func NewInitialMarketSync(provider ports.InitialMarketDataProvider, stocks ports.StockRepository, calendars ports.TradeCalendarRepository) *InitialMarketSyncService {
	return &InitialMarketSyncService{provider: provider, stocks: stocks, calendars: calendars}
}

// Sync 在写入任一仓储前先获取并校验全部必需数据。
func (service *InitialMarketSyncService) Sync(ctx context.Context, from, through types.TradingDate) (InitialMarketSyncResult, error) {
	if service == nil || service.provider == nil || service.stocks == nil || service.calendars == nil {
		return InitialMarketSyncResult{}, fmt.Errorf("initial market sync: provider and repositories are required")
	}
	if !from.Valid() || !through.Valid() || from.String() > through.String() {
		return InitialMarketSyncResult{}, apperror.New(apperror.CodeInvalidArgument, fmt.Errorf("initial market sync: valid ordered date range is required"))
	}
	if err := ctx.Err(); err != nil {
		return InitialMarketSyncResult{}, fmt.Errorf("initial market sync context: %w", err)
	}

	stocks, err := service.fetchAndMergeStocks(ctx)
	if err != nil {
		return InitialMarketSyncResult{}, err
	}
	calendars, err := service.fetchCalendars(ctx, from, through)
	if err != nil {
		return InitialMarketSyncResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return InitialMarketSyncResult{}, fmt.Errorf("initial market sync context: %w", err)
	}
	if err := service.stocks.Upsert(ctx, stocks); err != nil {
		return InitialMarketSyncResult{}, fmt.Errorf("initial market sync: upsert stocks: %w", err)
	}
	if err := service.calendars.Upsert(ctx, calendars); err != nil {
		return InitialMarketSyncResult{}, fmt.Errorf("initial market sync: upsert calendars: %w", err)
	}
	return InitialMarketSyncResult{StockCount: len(stocks), CalendarCount: len(calendars)}, nil
}

func (service *InitialMarketSyncService) fetchAndMergeStocks(ctx context.Context) ([]domain.Stock, error) {
	merged := make(map[string]domain.Stock)
	for _, status := range []string{"L", "D", "P"} {
		rows, err := service.provider.StockBasics(ctx, status)
		if err != nil {
			return nil, fmt.Errorf("initial market sync: fetch stock_basic status %s: %w", status, err)
		}
		for index, row := range rows {
			if err := validateInitialStock(row, status); err != nil {
				return nil, fmt.Errorf("initial market sync: stock_basic status %s row %d: %w", status, index, err)
			}
			current, exists := merged[row.TSCode]
			if !exists {
				merged[row.TSCode] = row
				continue
			}
			combined, err := mergeInitialStock(current, row)
			if err != nil {
				return nil, fmt.Errorf("initial market sync: stock %s: %w", row.TSCode, err)
			}
			merged[row.TSCode] = combined
		}
	}
	result := make([]domain.Stock, 0, len(merged))
	for _, stock := range merged {
		result = append(result, stock)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].TSCode < result[j].TSCode })
	return result, nil
}

func validateInitialStock(stock domain.Stock, requestedStatus string) error {
	if strings.TrimSpace(stock.TSCode) == "" || strings.TrimSpace(stock.Name) == "" || strings.TrimSpace(stock.Exchange) == "" || strings.TrimSpace(stock.Market) == "" {
		return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("stock code, name, exchange and market are required"))
	}
	if !stock.ListDate.Valid() || (stock.DelistDate != nil && (!stock.DelistDate.Valid() || stock.DelistDate.String() < stock.ListDate.String())) {
		return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("stock listing interval is invalid"))
	}
	if stock.ListStatus == "D" && stock.DelistDate == nil {
		return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("delisted stock requires delist_date"))
	}
	if stock.ListStatus != requestedStatus || (stock.ListStatus != "L" && stock.ListStatus != "D" && stock.ListStatus != "P") {
		return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("stock list_status does not match requested status %s", requestedStatus))
	}
	if stock.UpdatedAt.IsZero() {
		return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("stock updated_at is required"))
	}
	return nil
}

func mergeInitialStock(current, candidate domain.Stock) (domain.Stock, error) {
	if current.Exchange != candidate.Exchange || current.Market != candidate.Market || current.ListDate != candidate.ListDate {
		return domain.Stock{}, fmt.Errorf("%w: exchange, market or list_date differs", errConflictingInitialStock)
	}
	if current.DelistDate != nil && candidate.DelistDate != nil && *current.DelistDate != *candidate.DelistDate {
		return domain.Stock{}, fmt.Errorf("%w: delist_date differs", errConflictingInitialStock)
	}
	result := current
	if stockStatusPriority(candidate.ListStatus) > stockStatusPriority(current.ListStatus) {
		result.Name = candidate.Name
		result.ListStatus = candidate.ListStatus
	}
	if result.DelistDate == nil && candidate.DelistDate != nil {
		date := *candidate.DelistDate
		result.DelistDate = &date
	}
	if candidate.UpdatedAt.After(result.UpdatedAt) {
		result.UpdatedAt = candidate.UpdatedAt
	}
	return result, nil
}

func stockStatusPriority(status string) int {
	switch status {
	case "D":
		return 3
	case "L":
		return 2
	case "P":
		return 1
	default:
		return 0
	}
}

func (service *InitialMarketSyncService) fetchCalendars(ctx context.Context, from, through types.TradingDate) ([]domain.TradeCalendar, error) {
	merged := make(map[string]domain.TradeCalendar)
	expectedDates := inclusiveCalendarDayCount(from, through)
	for _, exchange := range []string{"SSE", "SZSE"} {
		rows, err := service.provider.TradeCalendars(ctx, exchange, from, through)
		if err != nil {
			return nil, fmt.Errorf("initial market sync: fetch trade_cal %s: %w", exchange, err)
		}
		seenDates := make(map[string]struct{}, len(rows))
		for index, row := range rows {
			if row.Exchange != exchange || !row.Date.Valid() || row.Date.String() < from.String() || row.Date.String() > through.String() {
				return nil, apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("initial market sync: trade_cal %s row %d is outside the requested range", exchange, index))
			}
			key := row.Exchange + "/" + row.Date.String()
			seenDates[row.Date.String()] = struct{}{}
			current, exists := merged[key]
			if exists && !sameCalendar(current, row) {
				return nil, fmt.Errorf("initial market sync: %w: %s", errConflictingInitialCalendar, key)
			}
			merged[key] = row
		}
		if len(seenDates) != expectedDates {
			return nil, apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("initial market sync: trade_cal %s returned %d of %d calendar dates", exchange, len(seenDates), expectedDates))
		}
	}
	result := make([]domain.TradeCalendar, 0, len(merged))
	for _, row := range merged {
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Exchange == result[j].Exchange {
			return result[i].Date.String() < result[j].Date.String()
		}
		return result[i].Exchange < result[j].Exchange
	})
	return result, nil
}

func inclusiveCalendarDayCount(from, through types.TradingDate) int {
	start, _ := time.Parse("2006-01-02", from.String())
	end, _ := time.Parse("2006-01-02", through.String())
	return int(end.Sub(start)/(24*time.Hour)) + 1
}

func sameCalendar(left, right domain.TradeCalendar) bool {
	if left.Exchange != right.Exchange || left.Date != right.Date || left.IsOpen != right.IsOpen {
		return false
	}
	if left.PretradeDate == nil || right.PretradeDate == nil {
		return left.PretradeDate == nil && right.PretradeDate == nil
	}
	return *left.PretradeDate == *right.PretradeDate
}
