package tushare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

func TestDecodeSourceSchemasAndMapDomainRows(t *testing.T) {
	stockRows := sourceRows(t, "stock_basic", "stock_basic_schema.json")
	stock, err := MapStockBasic(stockRows[0], MappingMetadata{UpdatedAt: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("MapStockBasic() error = %v", err)
	}
	if stock.TSCode != "600000.SH" || stock.Name != "浦发银行" || stock.Exchange != "SSE" || stock.ListStatus != "L" || stock.ListDate.String() != "1999-11-10" || stock.DelistDate != nil {
		t.Fatalf("mapped stock = %+v", stock)
	}

	calendarRows := sourceRows(t, "trade_cal", "trade_cal_schema.json")
	calendar, err := MapTradeCalendar(calendarRows[0])
	if err != nil {
		t.Fatalf("MapTradeCalendar() error = %v", err)
	}
	if !calendar.IsOpen || calendar.Date.String() != "2026-10-09" || calendar.PretradeDate == nil || calendar.PretradeDate.String() != "2026-10-08" {
		t.Fatalf("mapped calendar = %+v", calendar)
	}
	closed, err := MapTradeCalendar(calendarRows[1])
	if err != nil || closed.IsOpen || closed.PretradeDate != nil {
		t.Fatalf("closed calendar = %+v, error = %v", closed, err)
	}

	dailyRows := sourceRows(t, "daily", "daily_schema.json")
	daily, err := MapDailyPrice(dailyRows[0], MappingMetadata{FetchedAt: time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC), SourceHash: testSourceHash()})
	if err != nil {
		t.Fatalf("MapDailyPrice() error = %v", err)
	}
	if daily.AmountYuan.String() != "12500500" || daily.VolumeLot.String() != "12500.5" || daily.Close.String() != "10.123456" || daily.TradeDate.String() != "2026-10-09" {
		t.Fatalf("mapped daily = %+v", daily)
	}

	factorRows := sourceRows(t, "adj_factor", "adj_factor_schema.json")
	factor, err := MapAdjFactor(factorRows[0], MappingMetadata{FetchedAt: time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC), SourceHash: testSourceHash()})
	if err != nil || factor.Factor.String() != "1.2345678901" {
		t.Fatalf("mapped adjustment factor = %+v, error = %v", factor, err)
	}
	largeAmountRow := cloneRow(dailyRows[0])
	largeAmountRow["amount"] = json.RawMessage(`9007199254740993.0000`)
	largeAmount, err := MapDailyPrice(largeAmountRow, MappingMetadata{FetchedAt: time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC), SourceHash: testSourceHash()})
	if err != nil || largeAmount.AmountYuan.String() != "9007199254740993000" {
		t.Fatalf("large exact mapped amount = %q, error = %v", largeAmount.AmountYuan.String(), err)
	}

	suspendRows := sourceRows(t, "suspend_d", "suspend_d_schema.json")
	suspend, err := DecodeSuspendD(suspendRows[0])
	if err != nil || suspend.SuspendTiming != nil || suspend.SuspendType != "S" {
		t.Fatalf("decoded suspend row = %+v, error = %v", suspend, err)
	}
	limitRows := sourceRows(t, "stk_limit", "stk_limit_schema.json")
	limit, err := DecodeStkLimit(limitRows[0])
	if err != nil || limit.UpLimit.String() != "11" || limit.DownLimit.String() != "9" || limit.AssetType == nil || *limit.AssetType != "STK" {
		t.Fatalf("decoded limit row = %+v, error = %v", limit, err)
	}
}

func TestMappingRejectsMissingNullAndEmptyRequiredFields(t *testing.T) {
	row := sourceRows(t, "daily", "daily_schema.json")[0]
	cases := []struct {
		name  string
		field string
		value json.RawMessage
	}{
		{name: "missing required field", field: "amount"},
		{name: "null required field", field: "amount", value: json.RawMessage("null")},
		{name: "empty required date", field: "trade_date", value: json.RawMessage(`""`)},
		{name: "wrong numeric type", field: "close", value: json.RawMessage(`"10.5"`)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			candidate := cloneRow(row)
			if tt.value == nil {
				delete(candidate, tt.field)
			} else {
				candidate[tt.field] = tt.value
			}
			if _, err := MapDailyPrice(candidate, MappingMetadata{FetchedAt: time.Now().UTC(), SourceHash: testSourceHash()}); err == nil {
				t.Fatal("MapDailyPrice() error = nil, want validation error")
			}
		})
	}
}

func TestMappingRejectsInvalidDatesNumbersAndUnsupportedStockStatuses(t *testing.T) {
	daily := sourceRows(t, "daily", "daily_schema.json")[0]
	daily["trade_date"] = json.RawMessage(`"20260230"`)
	if _, err := MapDailyPrice(daily, MappingMetadata{FetchedAt: time.Now().UTC(), SourceHash: testSourceHash()}); err == nil {
		t.Fatal("invalid date was accepted")
	}

	daily = sourceRows(t, "daily", "daily_schema.json")[0]
	daily["amount"] = json.RawMessage(`1e999`)
	if _, err := MapDailyPrice(daily, MappingMetadata{FetchedAt: time.Now().UTC(), SourceHash: testSourceHash()}); err == nil {
		t.Fatal("non-finite converted amount was accepted")
	}

	stock := sourceRows(t, "stock_basic", "stock_basic_schema.json")[0]
	stock["list_status"] = json.RawMessage(`"G"`)
	if _, err := MapStockBasic(stock, MappingMetadata{UpdatedAt: time.Now().UTC()}); apperror.CodeOf(err) != apperror.CodeDataIncomplete {
		t.Fatalf("unsupported stock status error = %v, want DATA_INCOMPLETE", err)
	}
}

func TestMappingRejectsZeroAdjustmentFactor(t *testing.T) {
	row := sourceRows(t, "adj_factor", "adj_factor_schema.json")[0]
	row["adj_factor"] = json.RawMessage(`0`)
	if _, err := MapAdjFactor(row, MappingMetadata{FetchedAt: time.Now().UTC(), SourceHash: testSourceHash()}); err == nil {
		t.Fatal("zero adjustment factor was accepted")
	}
}

func TestDecimalPreservesBaseTenValueThroughAmountConversion(t *testing.T) {
	input := json.RawMessage(`12500.5000`)
	amount, err := types.ParseDecimal(string(input))
	if err != nil {
		t.Fatalf("decodeDecimal() error = %v", err)
	}
	converted, err := amount.MultiplyInt64(1000)
	if err != nil || converted.String() != "12500500" {
		t.Fatalf("amount × 1000 = %s, %v; want 12500500", converted.String(), err)
	}
	if _, err := types.ParseDecimal(`1.00000000000000001`); err != nil {
		t.Fatalf("exact decimal was rejected: %v", err)
	}
}

func TestOptionalSourceFieldsAcceptNullAndEmpty(t *testing.T) {
	row := sourceRows(t, "stock_basic", "stock_basic_schema.json")[0]
	row["delist_date"] = json.RawMessage(`null`)
	row["fullname"] = json.RawMessage(`""`)
	dto, err := DecodeStockBasic(row)
	if err != nil {
		t.Fatalf("DecodeStockBasic() error = %v", err)
	}
	if dto.DelistDate != nil || dto.FullName != nil {
		t.Fatalf("optional fields = delist %v fullname %v; want nil", dto.DelistDate, dto.FullName)
	}
}

func TestMappingRequiresExplicitPersistenceMetadata(t *testing.T) {
	daily := sourceRows(t, "daily", "daily_schema.json")[0]
	if _, err := MapDailyPrice(daily, MappingMetadata{SourceHash: testSourceHash()}); err == nil {
		t.Fatal("missing fetched_at was accepted")
	}
	if _, err := MapDailyPrice(daily, MappingMetadata{FetchedAt: time.Now().UTC()}); err == nil {
		t.Fatal("missing source_hash was accepted")
	}
	if _, err := MapStockBasic(sourceRows(t, "stock_basic", "stock_basic_schema.json")[0], MappingMetadata{}); err == nil {
		t.Fatal("missing updated_at was accepted")
	}
}

func sourceRows(t *testing.T, apiName, filename string) []Row {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("read fixture %s: %v", filename, err)
	}
	return mustParseFixture(t, apiName, content)
}

func mustParseFixture(t *testing.T, apiName string, content []byte) []Row {
	t.Helper()
	rows, err := (&Client{}).parseResponse(apiName, nil, content)
	if err != nil {
		t.Fatalf("parse %s source fixture: %v", apiName, err)
	}
	return rows
}

func cloneRow(row Row) Row {
	result := make(Row, len(row))
	for name, value := range row {
		result[name] = append(json.RawMessage(nil), value...)
	}
	return result
}

func testSourceHash() string {
	sum := sha256.Sum256([]byte("p03-03 fixture row"))
	return hex.EncodeToString(sum[:])
}

func TestFixtureSchemasDoNotContainCredentials(t *testing.T) {
	entries, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.Contains(entry.Name(), "_schema.json") {
			continue
		}
		content, err := os.ReadFile(filepath.Join("testdata", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "token") || strings.Contains(string(content), "api.tushare.pro") {
			t.Fatalf("fixture %s contains provider request details", entry.Name())
		}
	}
}
