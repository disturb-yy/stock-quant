package tushare

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"stock-quant/internal/market/domain"
	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

// MappingMetadata carries persistence metadata supplied by the caller.
type MappingMetadata struct {
	FetchedAt  time.Time
	SourceHash string
	UpdatedAt  time.Time
}

// DecodeStockBasic maps a dynamic row to the stock_basic provider schema.
func DecodeStockBasic(row Row) (StockBasicDTO, error) {
	var result StockBasicDTO
	var err error
	if result.TSCode, err = requiredText(row, "ts_code"); err != nil {
		return result, err
	}
	if result.Symbol, err = requiredText(row, "symbol"); err != nil {
		return result, err
	}
	if result.Name, err = requiredText(row, "name"); err != nil {
		return result, err
	}
	if result.Area, err = requiredText(row, "area"); err != nil {
		return result, err
	}
	if result.Industry, err = requiredText(row, "industry"); err != nil {
		return result, err
	}
	if result.CNSPELL, err = requiredText(row, "cnspell"); err != nil {
		return result, err
	}
	if result.Market, err = requiredText(row, "market"); err != nil {
		return result, err
	}
	if result.ListDate, err = requiredText(row, "list_date"); err != nil {
		return result, err
	}
	optionalFields := []struct {
		name   string
		target **string
	}{
		{"fullname", &result.FullName}, {"enname", &result.EnglishName}, {"exchange", &result.Exchange},
		{"curr_type", &result.Currency}, {"list_status", &result.ListStatus}, {"delist_date", &result.DelistDate},
		{"is_hs", &result.IsHS}, {"act_name", &result.ActName}, {"act_ent_type", &result.ActEntType},
	}
	for _, field := range optionalFields {
		*field.target, err = optionalText(row, field.name)
		if err != nil {
			return StockBasicDTO{}, err
		}
	}
	return result, nil
}

// DecodeTradeCalendar maps a dynamic row to the trade_cal provider schema.
func DecodeTradeCalendar(row Row) (TradeCalendarDTO, error) {
	var result TradeCalendarDTO
	var err error
	if result.Exchange, err = requiredText(row, "exchange"); err != nil {
		return result, err
	}
	if result.CalDate, err = requiredText(row, "cal_date"); err != nil {
		return result, err
	}
	if result.IsOpen, err = requiredZeroOne(row, "is_open"); err != nil {
		return result, err
	}
	if result.PretradeDate, err = optionalText(row, "pretrade_date"); err != nil {
		return result, err
	}
	return result, nil
}

// DecodeDaily maps a dynamic row to the daily provider schema.
func DecodeDaily(row Row) (DailyDTO, error) {
	var result DailyDTO
	var err error
	if result.TSCode, err = requiredText(row, "ts_code"); err != nil {
		return result, err
	}
	if result.TradeDate, err = requiredText(row, "trade_date"); err != nil {
		return result, err
	}
	for _, field := range []struct {
		name   string
		target *types.Decimal
	}{{"open", &result.Open}, {"high", &result.High}, {"low", &result.Low}, {"close", &result.Close}, {"vol", &result.Volume}, {"amount", &result.Amount}} {
		*field.target, err = requiredDecimal(row, field.name)
		if err != nil {
			return DailyDTO{}, err
		}
	}
	for _, field := range []struct {
		name   string
		target **types.Decimal
	}{{"pre_close", &result.PreClose}, {"change", &result.Change}, {"pct_chg", &result.PctChg}, {"ah_vol", &result.AHVolume}, {"ah_amount", &result.AHAmount}} {
		*field.target, err = optionalDecimal(row, field.name)
		if err != nil {
			return DailyDTO{}, err
		}
	}
	return result, nil
}

// DecodeAdjFactor maps a dynamic row to the adj_factor provider schema.
func DecodeAdjFactor(row Row) (AdjFactorDTO, error) {
	var result AdjFactorDTO
	var err error
	if result.TSCode, err = requiredText(row, "ts_code"); err != nil {
		return result, err
	}
	if result.TradeDate, err = requiredText(row, "trade_date"); err != nil {
		return result, err
	}
	if result.AdjFactor, err = requiredDecimal(row, "adj_factor"); err != nil {
		return result, err
	}
	return result, nil
}

// DecodeSuspendD maps a dynamic row to the suspend_d event schema.
func DecodeSuspendD(row Row) (SuspendDDTO, error) {
	var result SuspendDDTO
	var err error
	if result.TSCode, err = requiredText(row, "ts_code"); err != nil {
		return result, err
	}
	if result.TradeDate, err = requiredText(row, "trade_date"); err != nil {
		return result, err
	}
	if result.SuspendType, err = requiredText(row, "suspend_type"); err != nil {
		return result, err
	}
	if result.SuspendType != "S" && result.SuspendType != "R" {
		return SuspendDDTO{}, incompleteField("suspend_type", "unsupported value")
	}
	if result.SuspendTiming, err = optionalText(row, "suspend_timing"); err != nil {
		return SuspendDDTO{}, err
	}
	if _, err := parseDate(result.TradeDate, "trade_date"); err != nil {
		return SuspendDDTO{}, err
	}
	return result, nil
}

// DecodeStkLimit maps a dynamic row to the stk_limit provider schema.
func DecodeStkLimit(row Row) (StkLimitDTO, error) {
	var result StkLimitDTO
	var err error
	if result.TradeDate, err = requiredText(row, "trade_date"); err != nil {
		return result, err
	}
	if result.TSCode, err = requiredText(row, "ts_code"); err != nil {
		return result, err
	}
	if result.UpLimit, err = requiredDecimal(row, "up_limit"); err != nil {
		return result, err
	}
	if result.DownLimit, err = requiredDecimal(row, "down_limit"); err != nil {
		return result, err
	}
	if result.PreClose, err = optionalDecimal(row, "pre_close"); err != nil {
		return result, err
	}
	if result.AssetType, err = optionalText(row, "asset_type"); err != nil {
		return result, err
	}
	if result.Exchange, err = optionalText(row, "exchange"); err != nil {
		return result, err
	}
	if _, err := parseDate(result.TradeDate, "trade_date"); err != nil {
		return StkLimitDTO{}, err
	}
	for name, value := range map[string]types.Decimal{"up_limit": result.UpLimit, "down_limit": result.DownLimit} {
		if !value.Fits(20, 6) || !positiveDecimal(value) {
			return StkLimitDTO{}, incompleteField(name, "must be a positive price within DECIMAL(20,6)")
		}
	}
	return result, nil
}

// MapStockBasic converts provider fields to the market stock domain record.
func MapStockBasic(row Row, metadata MappingMetadata) (domain.Stock, error) {
	dto, err := DecodeStockBasic(row)
	if err != nil {
		return domain.Stock{}, err
	}
	if metadata.UpdatedAt.IsZero() {
		return domain.Stock{}, incompleteField("updated_at", "explicit metadata is required")
	}
	if dto.Exchange == nil || dto.ListStatus == nil {
		return domain.Stock{}, incompleteField("exchange/list_status", "required by market domain")
	}
	if *dto.ListStatus != "L" && *dto.ListStatus != "D" && *dto.ListStatus != "P" {
		return domain.Stock{}, incompleteField("list_status", "only L, D, and P are supported")
	}
	listDate, err := parseDate(dto.ListDate, "list_date")
	if err != nil {
		return domain.Stock{}, err
	}
	var delistDate *types.TradingDate
	if dto.DelistDate != nil {
		parsed, parseErr := parseDate(*dto.DelistDate, "delist_date")
		if parseErr != nil {
			return domain.Stock{}, parseErr
		}
		delistDate = &parsed
	}
	return domain.Stock{TSCode: dto.TSCode, Name: dto.Name, Exchange: *dto.Exchange, Market: dto.Market, ListDate: listDate, DelistDate: delistDate, ListStatus: *dto.ListStatus, UpdatedAt: metadata.UpdatedAt}, nil
}

// MapTradeCalendar converts provider fields to the market trade-calendar domain record.
func MapTradeCalendar(row Row) (domain.TradeCalendar, error) {
	dto, err := DecodeTradeCalendar(row)
	if err != nil {
		return domain.TradeCalendar{}, err
	}
	date, err := parseDate(dto.CalDate, "cal_date")
	if err != nil {
		return domain.TradeCalendar{}, err
	}
	isOpen, err := parseOpen(dto.IsOpen)
	if err != nil {
		return domain.TradeCalendar{}, err
	}
	var pretradeDate *types.TradingDate
	if dto.PretradeDate != nil {
		parsed, parseErr := parseDate(*dto.PretradeDate, "pretrade_date")
		if parseErr != nil {
			return domain.TradeCalendar{}, parseErr
		}
		pretradeDate = &parsed
	}
	return domain.TradeCalendar{Exchange: dto.Exchange, Date: date, IsOpen: isOpen, PretradeDate: pretradeDate}, nil
}

// MapDailyPrice converts daily fields to the market domain, including amount from thousand yuan.
func MapDailyPrice(row Row, metadata MappingMetadata) (domain.DailyPrice, error) {
	dto, err := DecodeDaily(row)
	if err != nil {
		return domain.DailyPrice{}, err
	}
	if metadata.FetchedAt.IsZero() || !validSourceHash(metadata.SourceHash) {
		return domain.DailyPrice{}, incompleteField("fetched_at/source_hash", "explicit valid metadata is required")
	}
	date, err := parseDate(dto.TradeDate, "trade_date")
	if err != nil {
		return domain.DailyPrice{}, err
	}
	convertedAmount, err := dto.Amount.MultiplyInt64(1000)
	if err != nil {
		return domain.DailyPrice{}, incompleteField("amount", "cannot be converted to yuan")
	}
	if !convertedAmount.Fits(24, 4) || !dto.Volume.Fits(24, 4) {
		return domain.DailyPrice{}, incompleteField("amount/vol", "converted value exceeds DECIMAL(24,4)")
	}
	prices := []struct {
		name  string
		value types.Decimal
	}{{"open", dto.Open}, {"high", dto.High}, {"low", dto.Low}, {"close", dto.Close}}
	for _, price := range prices {
		if !price.value.Fits(20, 6) || !positiveDecimal(price.value) {
			return domain.DailyPrice{}, incompleteField(price.name, "must be positive and fit DECIMAL(20,6)")
		}
	}
	if negativeDecimal(dto.Volume) || negativeDecimal(dto.Amount) {
		return domain.DailyPrice{}, incompleteField("vol/amount", "negative values are invalid")
	}
	if err := validateOHLC(dto.Open, dto.High, dto.Low, dto.Close); err != nil {
		return domain.DailyPrice{}, err
	}
	amount, err := types.NewAmountYuanDecimal(convertedAmount)
	if err != nil {
		return domain.DailyPrice{}, fmt.Errorf("map daily amount: %w", err)
	}
	return domain.DailyPrice{TSCode: dto.TSCode, TradeDate: date, Open: dto.Open, High: dto.High, Low: dto.Low, Close: dto.Close, AmountYuan: amount, VolumeLot: dto.Volume, SourceHash: metadata.SourceHash, FetchedAt: metadata.FetchedAt}, nil
}

// MapAdjFactor converts provider fields to the market adjustment-factor domain record.
func MapAdjFactor(row Row, metadata MappingMetadata) (domain.AdjFactor, error) {
	dto, err := DecodeAdjFactor(row)
	if err != nil {
		return domain.AdjFactor{}, err
	}
	if metadata.FetchedAt.IsZero() || !validSourceHash(metadata.SourceHash) {
		return domain.AdjFactor{}, incompleteField("fetched_at/source_hash", "explicit valid metadata is required")
	}
	date, err := parseDate(dto.TradeDate, "trade_date")
	if err != nil {
		return domain.AdjFactor{}, err
	}
	if !positiveDecimal(dto.AdjFactor) || !dto.AdjFactor.Fits(24, 10) {
		return domain.AdjFactor{}, incompleteField("adj_factor", "must be positive and fit DECIMAL(24,10)")
	}
	return domain.AdjFactor{TSCode: dto.TSCode, TradeDate: date, Factor: dto.AdjFactor, SourceHash: metadata.SourceHash, FetchedAt: metadata.FetchedAt}, nil
}

func requiredText(row Row, name string) (string, error) {
	raw, exists := row[name]
	if !exists || strings.TrimSpace(string(raw)) == "" || string(raw) == "null" {
		return "", incompleteField(name, "required value is missing")
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", incompleteField(name, "must be a non-empty string")
	}
	return value, nil
}

func requiredZeroOne(row Row, name string) (string, error) {
	raw, exists := row[name]
	if !exists || string(raw) == "null" {
		return "", incompleteField(name, "required value is missing")
	}
	value := string(raw)
	if err := json.Unmarshal(raw, &value); err != nil {
		value = strings.TrimSpace(value)
	}
	if value != "0" && value != "1" {
		return "", incompleteField(name, "must be the integer 0 or 1")
	}
	return value, nil
}

func optionalText(row Row, name string) (*string, error) {
	raw, exists := row[name]
	if !exists || string(raw) == "null" {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, incompleteField(name, "must be a string or null")
	}
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	return &value, nil
}

func requiredDecimal(row Row, name string) (types.Decimal, error) {
	raw, exists := row[name]
	if !exists || string(raw) == "null" {
		return types.Decimal{}, incompleteField(name, "required numeric value is missing")
	}
	value, err := types.ParseDecimal(string(raw))
	if err != nil {
		return types.Decimal{}, incompleteField(name, "must be a JSON decimal number")
	}
	return value, nil
}

func optionalDecimal(row Row, name string) (*types.Decimal, error) {
	raw, exists := row[name]
	if !exists || string(raw) == "null" {
		return nil, nil
	}
	value, err := types.ParseDecimal(string(raw))
	if err != nil {
		return nil, incompleteField(name, "must be a JSON decimal number or null")
	}
	return &value, nil
}

func parseDate(value, field string) (types.TradingDate, error) {
	date, err := types.ParseTushareDate(value)
	if err != nil {
		return types.TradingDate{}, incompleteField(field, "must be a valid YYYYMMDD date")
	}
	return date, nil
}

func parseOpen(value string) (bool, error) {
	switch value {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, incompleteField("is_open", "must be 0 or 1")
	}
}

func positiveDecimal(value types.Decimal) bool {
	zero, err := types.ParseDecimal("0")
	if err != nil {
		return false
	}
	comparison, err := value.Cmp(zero)
	return err == nil && comparison > 0
}

func negativeDecimal(value types.Decimal) bool {
	zero, err := types.ParseDecimal("0")
	if err != nil {
		return true
	}
	comparison, err := value.Cmp(zero)
	return err != nil || comparison < 0
}

func validateOHLC(open, high, low, close types.Decimal) error {
	highOpen, err := high.Cmp(open)
	if err != nil {
		return incompleteField("open/high/low/close", "invalid OHLC decimal")
	}
	highClose, _ := high.Cmp(close)
	lowOpen, _ := low.Cmp(open)
	lowClose, _ := low.Cmp(close)
	highLow, _ := high.Cmp(low)
	if highOpen < 0 || highClose < 0 || lowOpen > 0 || lowClose > 0 || highLow < 0 {
		return incompleteField("open/high/low/close", "OHLC values are inconsistent")
	}
	return nil
}

func validSourceHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func incompleteField(field, reason string) error {
	return apperror.New(apperror.CodeDataIncomplete, fmt.Errorf("Tushare field %s: %s", field, reason))
}
