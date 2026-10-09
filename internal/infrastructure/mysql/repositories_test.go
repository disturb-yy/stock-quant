package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"stock-quant/internal/infrastructure/tushare"
	"stock-quant/internal/market/domain"
	"stock-quant/internal/shared/types"
)

func TestDailyPriceUpsertRevisionAndStableQueryOrderMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewDailyPriceRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	older := tradingDate(t, "2026-06-03")
	newer := tradingDate(t, "2026-06-04")
	first := dailyPrice(t, "000001.SZ", newer, 'a')
	second := dailyPrice(t, "000001.SZ", older, 'b')
	if err := repository.Upsert(ctx, []domain.DailyPrice{first, second}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}
	if err := repository.Upsert(ctx, []domain.DailyPrice{first}); err != nil {
		t.Fatalf("duplicate upsert: %v", err)
	}
	got, err := repository.ListByCode(ctx, "000001.SZ", older, newer)
	if err != nil {
		t.Fatalf("list by code: %v", err)
	}
	if len(got) != 2 || got[0].TradeDate != older || got[1].TradeDate != newer {
		t.Fatalf("daily rows are not in ascending trade-date order: %#v", got)
	}
	if got[1].Revision != 1 || got[1].AmountYuan.String() != "12500500.25" || got[1].VolumeLot.String() != "12500.5" {
		t.Fatalf("stored amount/volume/revision = (%s, %s, %d)", got[1].AmountYuan.String(), got[1].VolumeLot.String(), got[1].Revision)
	}

	changed := first
	changed.Close = decimalValue(t, "10.35")
	changed.SourceHash = sourceHash('c')
	changed.FetchedAt = first.FetchedAt.Add(time.Minute)
	if err := repository.Upsert(ctx, []domain.DailyPrice{changed}); err != nil {
		t.Fatalf("changed upsert: %v", err)
	}
	got, err = repository.ListByCode(ctx, "000001.SZ", newer, newer)
	if err != nil || len(got) != 1 || got[0].Revision != 2 || got[0].Close.String() != changed.Close.String() {
		t.Fatalf("changed row = %#v, error %v; want close=%v revision=2", got, err, changed.Close)
	}

	changed.FetchedAt = changed.FetchedAt.Add(time.Minute)
	if err := repository.Upsert(ctx, []domain.DailyPrice{changed}); err != nil {
		t.Fatalf("fetched_at-only upsert: %v", err)
	}
	got, err = repository.ListByCode(ctx, "000001.SZ", newer, newer)
	if err != nil || len(got) != 1 || got[0].Revision != 2 {
		t.Fatalf("fetched_at-only change must not increment revision: rows=%#v err=%v", got, err)
	}
}

func TestMarketDecimalTextRoundTripsAndRevisionIsStableMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	dailyRepository, err := NewDailyPriceRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	factorRepository, err := NewAdjFactorRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	date := tradingDate(t, "2026-06-03")
	metadata := tushare.MappingMetadata{FetchedAt: fixedFetchedAt(), SourceHash: sourceHash('a')}
	bar, err := tushare.MapDailyPrice(tushare.Row{
		"ts_code": json.RawMessage(`"000001.SZ"`), "trade_date": json.RawMessage(`"20260603"`),
		"open": json.RawMessage(`10.000001`), "high": json.RawMessage(`10.000002`),
		"low": json.RawMessage(`9.999999`), "close": json.RawMessage(`10.000001`),
		"vol": json.RawMessage(`12500.5000`), "amount": json.RawMessage(`9007199254740.993`),
	}, metadata)
	if err != nil {
		t.Fatalf("map exact daily provider decimals: %v", err)
	}
	factor, err := tushare.MapAdjFactor(tushare.Row{
		"ts_code": json.RawMessage(`"000001.SZ"`), "trade_date": json.RawMessage(`"20260603"`), "adj_factor": json.RawMessage(`1.2345678901`),
	}, tushare.MappingMetadata{FetchedAt: fixedFetchedAt(), SourceHash: sourceHash('b')})
	if err != nil {
		t.Fatalf("map exact adjustment factor: %v", err)
	}
	if err := dailyRepository.Upsert(ctx, []domain.DailyPrice{bar}); err != nil {
		t.Fatalf("upsert exact daily decimals: %v", err)
	}
	if err := factorRepository.Upsert(ctx, []domain.AdjFactor{factor}); err != nil {
		t.Fatalf("upsert exact adjustment factor: %v", err)
	}
	assertDecimalText := func(table, code, column, want string) {
		t.Helper()
		var got string
		if err := db.QueryRowContext(ctx, "SELECT CAST("+column+" AS CHAR) FROM "+table+" WHERE ts_code=? AND trade_date=?", code, date.String()).Scan(&got); err != nil {
			t.Fatalf("read exact %s.%s text: %v", table, column, err)
		}
		if got != want {
			t.Fatalf("stored %s.%s = %q, want exact %q", table, column, got, want)
		}
	}
	assertDecimalText("t_daily_price", bar.TSCode, "amount_yuan", "9007199254740993.0000")
	assertDecimalText("t_daily_price", bar.TSCode, "open", "10.000001")
	assertDecimalText("t_daily_price", bar.TSCode, "volume_lot", "12500.5000")
	assertDecimalText("t_adj_factor", factor.TSCode, "adj_factor", "1.2345678901")
	loadedDaily, err := dailyRepository.ListByCode(ctx, bar.TSCode, date, date)
	if err != nil || len(loadedDaily) != 1 || loadedDaily[0].AmountYuan.String() != "9007199254740993" || loadedDaily[0].Open.String() != "10.000001" {
		t.Fatalf("repository exact daily read = %#v, error = %v", loadedDaily, err)
	}
	loadedFactors, err := factorRepository.ListByCode(ctx, factor.TSCode, date, date)
	if err != nil || len(loadedFactors) != 1 || loadedFactors[0].Factor.String() != "1.2345678901" {
		t.Fatalf("repository exact adjustment-factor read = %#v, error = %v", loadedFactors, err)
	}
	bar.FetchedAt = bar.FetchedAt.Add(time.Second)
	factor.FetchedAt = factor.FetchedAt.Add(time.Second)
	if err := dailyRepository.Upsert(ctx, []domain.DailyPrice{bar}); err != nil {
		t.Fatalf("re-upsert exact daily decimals: %v", err)
	}
	if err := factorRepository.Upsert(ctx, []domain.AdjFactor{factor}); err != nil {
		t.Fatalf("re-upsert exact adjustment factor: %v", err)
	}
	var dailyRevision, factorRevision int
	if err := db.QueryRowContext(ctx, "SELECT revision FROM t_daily_price WHERE ts_code=? AND trade_date=?", bar.TSCode, date.String()).Scan(&dailyRevision); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, "SELECT revision FROM t_adj_factor WHERE ts_code=? AND trade_date=?", factor.TSCode, date.String()).Scan(&factorRevision); err != nil {
		t.Fatal(err)
	}
	if dailyRevision != 1 || factorRevision != 1 {
		t.Fatalf("unchanged decimal revisions = daily %d, factor %d; want both 1", dailyRevision, factorRevision)
	}
}

func TestAdjFactorUpsertRevisionAndCalendarOrderMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	adjRepository, err := NewAdjFactorRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	calendarRepository, err := NewTradeCalendarRepository(db)
	if err != nil {
		t.Fatal(err)
	}

	firstDate := tradingDate(t, "2026-06-03")
	secondDate := tradingDate(t, "2026-06-04")
	factor := domain.AdjFactor{TSCode: "000001.SZ", TradeDate: firstDate, Factor: decimalValue(t, "1.2345678901"), SourceHash: sourceHash('d'), FetchedAt: fixedFetchedAt()}
	if err := adjRepository.Upsert(ctx, []domain.AdjFactor{factor, {TSCode: "000001.SZ", TradeDate: secondDate, Factor: decimalValue(t, "1.25"), SourceHash: sourceHash('e'), FetchedAt: fixedFetchedAt()}}); err != nil {
		t.Fatalf("adj factor upsert: %v", err)
	}
	if err := adjRepository.Upsert(ctx, []domain.AdjFactor{factor}); err != nil {
		t.Fatalf("repeated adj factor upsert: %v", err)
	}
	factor.Factor = decimalValue(t, "1.5")
	factor.SourceHash = sourceHash('f')
	if err := adjRepository.Upsert(ctx, []domain.AdjFactor{factor}); err != nil {
		t.Fatalf("changed adj factor upsert: %v", err)
	}
	factors, err := adjRepository.ListByCode(ctx, "000001.SZ", firstDate, secondDate)
	if err != nil || len(factors) != 2 || factors[0].TradeDate != firstDate || factors[0].Revision != 2 || factors[0].Factor.String() != "1.5" {
		t.Fatalf("adj factors = %#v err=%v; want sorted rows with revised first factor", factors, err)
	}

	calendars := []domain.TradeCalendar{
		{Exchange: "SSE", Date: secondDate, IsOpen: true},
		{Exchange: "SSE", Date: firstDate, IsOpen: true},
	}
	if err := calendarRepository.Upsert(ctx, calendars); err != nil {
		t.Fatalf("calendar upsert: %v", err)
	}
	gotCalendar, err := calendarRepository.List(ctx, "SSE", firstDate, secondDate)
	if err != nil || len(gotCalendar) != 2 || gotCalendar[0].Date != firstDate || gotCalendar[1].Date != secondDate {
		t.Fatalf("calendar rows = %#v err=%v; want ascending date order", gotCalendar, err)
	}
}

func TestRepositoryBatchValidationRollsBackEarlierRowsMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	repository, err := NewDailyPriceRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	valid := dailyPrice(t, "000001.SZ", tradingDate(t, "2026-06-03"), '7')
	invalid := dailyPrice(t, "000002.SZ", types.TradingDate{}, '8')
	if err := repository.Upsert(ctx, []domain.DailyPrice{valid, invalid}); err == nil || !strings.Contains(err.Error(), "valid trading date is required") {
		t.Fatalf("batch error = %v; want missing required trading date", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM t_daily_price").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("rows committed after failed batch = %d, want 0", count)
	}
}

func TestStockRepositoryKeepsDelistedHistoricalIdentityMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	stocks, err := NewStockRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	stock := domain.Stock{
		TSCode: "600001.SH", Name: "历史证券", Exchange: "SSE", Market: "主板",
		ListDate: tradingDate(t, "2001-01-01"), DelistDate: tradingDatePointer(t, "2020-08-31"),
		ListStatus: "D", UpdatedAt: fixedFetchedAt(),
	}
	if err := stocks.Upsert(ctx, []domain.Stock{stock}); err != nil {
		t.Fatalf("upsert delisted stock: %v", err)
	}
	if err := stocks.Upsert(ctx, []domain.Stock{stock}); err != nil {
		t.Fatalf("repeat delisted stock upsert: %v", err)
	}
	got, err := stocks.Find(ctx, stock.TSCode)
	if err != nil || got.TSCode != stock.TSCode || got.ListStatus != "D" || got.DelistDate == nil || *got.DelistDate != *stock.DelistDate {
		t.Fatalf("delisted stock = %#v, error %v", got, err)
	}
	stock.Name = "历史证券（更新）"
	stock.UpdatedAt = stock.UpdatedAt.Add(time.Minute)
	if err := stocks.Upsert(ctx, []domain.Stock{stock}); err != nil {
		t.Fatalf("update stock identity: %v", err)
	}
	got, err = stocks.Find(ctx, stock.TSCode)
	if err != nil || got.Name != stock.Name || got.UpdatedAt != stock.UpdatedAt {
		t.Fatalf("updated stock = %#v, error %v", got, err)
	}
	var stockCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM t_stock WHERE ts_code=?`, stock.TSCode).Scan(&stockCount); err != nil || stockCount != 1 {
		t.Fatalf("stock rows = %d, error %v; want one idempotent row", stockCount, err)
	}
	daily, err := NewDailyPriceRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	historical := dailyPrice(t, stock.TSCode, tradingDate(t, "2020-08-28"), '9')
	if err := daily.Upsert(ctx, []domain.DailyPrice{historical}); err != nil {
		t.Fatalf("upsert delisted stock historical bar: %v", err)
	}
	history, err := daily.ListByCode(ctx, stock.TSCode, historical.TradeDate, historical.TradeDate)
	if err != nil || len(history) != 1 || history[0].TSCode != stock.TSCode {
		t.Fatalf("delisted stock historical bars = %#v, error %v", history, err)
	}
}

func TestDailyPriceByDateUsesDateLeadingIndexMySQL(t *testing.T) {
	db := openIsolatedMySQL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	migrateForRepositoryTest(t, ctx, db)
	seedDateScanRows(t, ctx, db, 100, 60)
	if _, err := db.ExecContext(ctx, "ANALYZE TABLE t_daily_price"); err != nil {
		t.Fatalf("analyze fixture table: %v", err)
	}
	date := "2026-01-20"
	dailyRepository, err := NewDailyPriceRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	tradeDate := tradingDate(t, date)
	dateRows, err := dailyRepository.ListByDate(ctx, tradeDate)
	if err != nil || len(dateRows) != 60 {
		t.Fatalf("daily date scan returned %d rows, error %v; want 60", len(dateRows), err)
	}
	if dateRows[0].TSCode != "000000.SZ" || dateRows[len(dateRows)-1].TSCode != "000059.SZ" {
		t.Fatalf("daily date scan order = %s..%s; want ascending code", dateRows[0].TSCode, dateRows[len(dateRows)-1].TSCode)
	}
	var plan struct {
		Key sql.NullString
	}
	rows, err := db.QueryContext(ctx, "EXPLAIN SELECT ts_code FROM t_daily_price WHERE trade_date = ? ORDER BY ts_code", date)
	if err != nil {
		t.Fatalf("explain daily date scan: %v", err)
	}
	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}
	if !rows.Next() {
		rows.Close()
		t.Fatal("EXPLAIN returned no plan")
	}
	if err := rows.Scan(pointers...); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	keyColumn := -1
	for i, name := range columns {
		if name == "key" {
			keyColumn = i
		}
	}
	if keyColumn < 0 {
		t.Fatalf("EXPLAIN columns missing key: %v", columns)
	}
	switch key := values[keyColumn].(type) {
	case []byte:
		plan.Key = sql.NullString{String: string(key), Valid: true}
	case string:
		plan.Key = sql.NullString{String: key, Valid: true}
	case nil:
	default:
		t.Fatalf("EXPLAIN key has type %T, value %#v", values[keyColumn], values[keyColumn])
	}
	if !plan.Key.Valid || plan.Key.String != "idx_daily_by_date" {
		t.Fatalf("EXPLAIN date index = %q, want idx_daily_by_date", plan.Key.String)
	}
}

func TestRepositoryReturnsClosedDatabaseFailure(t *testing.T) {
	db, err := sql.Open("mysql", "root@unix(/tmp/stock-quant-missing-mysql.sock)/stock_quant")
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	repository, err := NewStockRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Find(context.Background(), "000001.SZ"); err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("Find error = %v, want closed database resource error", err)
	}
}

func TestMarketRepositoryRejectsValuesThatMySQLWouldRound(t *testing.T) {
	valid := dailyPrice(t, "000001.SZ", tradingDate(t, "2026-06-03"), 'a')
	tests := []struct {
		name   string
		change func(*domain.DailyPrice)
	}{
		{name: "amount scale", change: func(row *domain.DailyPrice) { row.AmountYuan = amountDecimal(t, "0.00001") }},
		{name: "price precision", change: func(row *domain.DailyPrice) { row.Close = decimalValue(t, "100000000000000") }},
		{name: "zero value decimal", change: func(row *domain.DailyPrice) { row.Close = types.Decimal{} }},
		{name: "invalid row hash", change: func(row *domain.DailyPrice) { row.SourceHash = strings.Repeat("z", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			row := valid
			test.change(&row)
			if err := validateDailyPrice(row); err == nil {
				t.Fatal("invalid value unexpectedly passed repository validation")
			}
		})
	}
}

func migrateForRepositoryTest(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	migrator := newEmbeddedMigrator(t, db)
	if err := migrator.Up(ctx); err != nil {
		t.Fatalf("apply schema migration: %v", err)
	}
}

func dailyPrice(t *testing.T, code string, date types.TradingDate, hashMarker byte) domain.DailyPrice {
	t.Helper()
	return domain.DailyPrice{
		TSCode: code, TradeDate: date, Open: decimalValue(t, "10.1"), High: decimalValue(t, "10.5"), Low: decimalValue(t, "9.9"), Close: decimalValue(t, "10.25"),
		AmountYuan: amountDecimal(t, "12500500.25"), VolumeLot: decimalValue(t, "12500.5"),
		SourceHash: sourceHash(hashMarker), FetchedAt: fixedFetchedAt(),
	}
}

func amountDecimal(t *testing.T, value string) types.AmountYuan {
	t.Helper()
	decimal, err := types.ParseDecimal(value)
	if err != nil {
		t.Fatal(err)
	}
	amount, err := types.NewAmountYuanDecimal(decimal)
	if err != nil {
		t.Fatal(err)
	}
	return amount
}

func decimalValue(t *testing.T, value string) types.Decimal {
	t.Helper()
	decimal, err := types.ParseDecimal(value)
	if err != nil {
		t.Fatal(err)
	}
	return decimal
}

func tradingDate(t *testing.T, value string) types.TradingDate {
	t.Helper()
	date, err := types.ParseTradingDate(value)
	if err != nil {
		t.Fatal(err)
	}
	return date
}

func tradingDatePointer(t *testing.T, value string) *types.TradingDate {
	t.Helper()
	date := tradingDate(t, value)
	return &date
}

func fixedFetchedAt() time.Time {
	return time.Date(2026, 6, 4, 8, 30, 0, 123456000, time.UTC)
}

func sourceHash(marker byte) string {
	return strings.Repeat(string(marker), 64)
}

func seedDateScanRows(t *testing.T, ctx context.Context, db *sql.DB, days, codes int) {
	t.Helper()
	const rowsPerInsert = 300
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	parameters := make([]any, 0, rowsPerInsert*11)
	valueGroups := make([]string, 0, rowsPerInsert)
	flush := func() {
		if len(valueGroups) == 0 {
			return
		}
		query := "INSERT INTO t_daily_price (ts_code, trade_date, open, high, low, close, amount_yuan, volume_lot, source_hash, revision, fetched_at) VALUES " + strings.Join(valueGroups, ",")
		if _, err := db.ExecContext(ctx, query, parameters...); err != nil {
			t.Fatalf("seed date scan rows: %v", err)
		}
		parameters = parameters[:0]
		valueGroups = valueGroups[:0]
	}
	for day := 0; day < days; day++ {
		tradeDate := date.AddDate(0, 0, day).Format("2006-01-02")
		for code := 0; code < codes; code++ {
			parameters = append(parameters,
				fmt.Sprintf("%06d.SZ", code), tradeDate,
				10, 10, 10, 10, 1000, 100, strings.Repeat("a", 64), 1, "2026-01-01 00:00:00.000000",
			)
			valueGroups = append(valueGroups, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
			if len(valueGroups) == rowsPerInsert {
				flush()
			}
		}
	}
	flush()
	if days*codes < 1000 {
		t.Fatalf("date index fixture is too small: days=%d codes=%d", days, codes)
	}
}
