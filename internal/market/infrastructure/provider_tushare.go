package infrastructure

import (
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disturb-yy/stock-quant/pkg/config"
)

const (
	demoMetadataName    = "fnd-003-demo"
	tushareMetadataName = "tushare-real"
	tushareProviderName = "tushare"

	tushareStockBasicFields = "ts_code,name,industry,exchange,list_status,list_date"
	tushareDailyFields      = "ts_code,trade_date,open,high,low,close,vol,amount"
	tushareDailyBasicFields = "ts_code,trade_date,turnover_rate,pe_ttm,pb,ps_ttm,total_mv"
	tushareFactorFields     = "ts_code,trade_date,adj_factor"
	tushareIndexFields      = "ts_code,trade_date,close,change,pct_chg"
	// Tushare 单接口限制为 200 次/分钟；4 个 worker 可提升吞吐并保持在配额内。
	tushareSyncWorkerCount = 4
	tushareWriteChunkSize  = 500
	tushareRequestInterval = 310 * time.Millisecond
)

var tushareIndexNames = map[string]string{
	"000001.SH": "上证指数",
	"399001.SZ": "深证成指",
	"399006.SZ": "创业板指",
	"000300.SH": "沪深300",
}

// TushareQueryClient 是 Tushare HTTP 查询的最小边界，便于同步用例测试。
type TushareQueryClient interface {
	Query(context.Context, string, map[string]string, string) ([]map[string]string, error)
}

// TushareClient 调用 Tushare Pro 的 REST API。
type TushareClient struct {
	endpoint    string
	token       string
	rateMu      sync.Mutex
	nextRequest time.Time
	httpClient  interface {
		Do(*http.Request) (*http.Response, error)
	}
}

// NewTushareClient 创建带超时的 Tushare 客户端。
func NewTushareClient(settings config.Tushare, clients ...interface {
	Do(*http.Request) (*http.Response, error)
}) (*TushareClient, error) {
	if strings.TrimSpace(settings.Token) == "" {
		return nil, errors.New("Tushare token is required")
	}
	parsed, err := url.Parse(strings.TrimRight(strings.TrimSpace(settings.Endpoint), "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid Tushare endpoint %q", settings.Endpoint)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported Tushare endpoint scheme %q", parsed.Scheme)
	}
	timeout := settings.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	httpClient := interface {
		Do(*http.Request) (*http.Response, error)
	}(&http.Client{Timeout: timeout})
	if len(clients) > 0 && clients[0] != nil {
		httpClient = clients[0]
	}
	return &TushareClient{endpoint: parsed.String(), token: settings.Token, httpClient: httpClient}, nil
}

type tushareResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data *tushareResult `json:"data"`
}

type tushareResult struct {
	Fields []string `json:"fields"`
	Items  [][]any  `json:"items"`
}

// Query 执行一次 Tushare POST 请求并将行转换为字段映射。
func (client *TushareClient) Query(ctx context.Context, apiName string, params map[string]string, fields string) ([]map[string]string, error) {
	if ctx == nil {
		return nil, errors.New("Tushare query context is required")
	}
	if strings.TrimSpace(apiName) == "" {
		return nil, errors.New("Tushare API name is required")
	}
	if err := client.waitForRateLimit(ctx); err != nil {
		return nil, fmt.Errorf("wait for Tushare request: %w", err)
	}
	payload, err := json.Marshal(map[string]any{"api_name": apiName, "token": client.token, "params": params, "fields": fields})
	if err != nil {
		return nil, fmt.Errorf("encode Tushare request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create Tushare request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Tushare %s: %w", apiName, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Tushare %s returned HTTP %d", apiName, response.StatusCode)
	}
	var decoded tushareResponse
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode Tushare %s response: %w", apiName, err)
	}
	if decoded.Code != 0 {
		return nil, fmt.Errorf("Tushare %s failed with code %d: %s", apiName, decoded.Code, strings.TrimSpace(decoded.Msg))
	}
	if decoded.Data == nil {
		return []map[string]string{}, nil
	}
	return mapTushareRows(decoded.Data.Fields, decoded.Data.Items)
}

func (client *TushareClient) waitForRateLimit(ctx context.Context) error {
	client.rateMu.Lock()
	now := time.Now()
	wait := time.Until(client.nextRequest)
	if wait < 0 {
		wait = 0
	}
	client.nextRequest = now.Add(wait + tushareRequestInterval)
	client.rateMu.Unlock()
	if wait == 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func mapTushareRows(fields []string, items [][]any) ([]map[string]string, error) {
	rows := make([]map[string]string, 0, len(items))
	for index, item := range items {
		if len(item) != len(fields) {
			return nil, fmt.Errorf("Tushare row %d has %d values, want %d", index, len(item), len(fields))
		}
		row := make(map[string]string, len(fields))
		for fieldIndex, field := range fields {
			row[field] = tushareValue(item[fieldIndex])
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func tushareValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

// TushareSyncOptions 控制一次有界的 Tushare 同步。
type TushareSyncOptions struct {
	StartDate    string
	EndDate      string
	LookbackDays int
	MetadataName string
}

// TushareSyncSummary 描述已提交的同步快照。
type TushareSyncSummary struct {
	Provider     string
	MetadataName string
	Version      string
	AsOf         string
	Instruments  int
	DailyBars    int
	DailyBasics  int
	Factors      int
	Indexes      int
}

// TushareSyncer 将 Tushare 数据幂等写入现有 MySQL 读模型。
type TushareSyncer struct {
	db     *sql.DB
	client TushareQueryClient
	now    func() time.Time
}

// NewTushareSyncer 创建同步用例。
func NewTushareSyncer(db *sql.DB, client TushareQueryClient) (*TushareSyncer, error) {
	if db == nil {
		return nil, errors.New("Tushare sync database connection is required")
	}
	if client == nil {
		return nil, errors.New("Tushare query client is required")
	}
	return &TushareSyncer{db: db, client: client, now: time.Now}, nil
}

type tushareInstrument struct {
	Code, Name, Industry, Exchange, Status, ListDate string
}

type tushareBar struct {
	Code, TradeDate, Open, High, Low, Close, TurnoverAmount string
	Volume                                                  int64
}

type tushareBasic struct {
	Code, TradeDate, MarketCap, PB, PETTM, PSTTM, TurnoverRate string
}

type tushareFactor struct {
	Code, TradeDate, Value string
}

type tushareIndex struct {
	Code, Name, TradeDate, Close, Change, ChangePercent string
}

type tushareBatch struct {
	Instruments []tushareInstrument
	Bars        []tushareBar
	Basics      []tushareBasic
	Factors     []tushareFactor
	Indexes     []tushareIndex
	AsOf        string
}

// Sync 拉取股票、日线、每日指标、复权因子和四个市场指数后一次性提交。
func (syncer *TushareSyncer) Sync(ctx context.Context, options TushareSyncOptions) (TushareSyncSummary, error) {
	if ctx == nil {
		return TushareSyncSummary{}, errors.New("Tushare sync context is required")
	}
	metadataName := strings.TrimSpace(options.MetadataName)
	if metadataName == "" {
		metadataName = tushareMetadataName
	}
	if err := syncer.checkSourceIsolation(ctx); err != nil {
		return TushareSyncSummary{}, err
	}
	startDate, endDate, dates, err := syncDateRange(options, syncer.now().UTC())
	if err != nil {
		return TushareSyncSummary{}, err
	}
	instruments, err := syncer.fetchInstruments(ctx)
	if err != nil {
		return TushareSyncSummary{}, err
	}
	indexes, err := syncer.fetchIndexes(ctx, startDate, endDate)
	if err != nil {
		return TushareSyncSummary{}, err
	}
	batch := tushareBatch{Instruments: instruments, Indexes: indexes}
	if err := syncer.fetchDays(ctx, dates, &batch); err != nil {
		return TushareSyncSummary{}, err
	}
	filterTushareBatchToInstruments(&batch)
	if err := validateTushareBatch(batch); err != nil {
		return TushareSyncSummary{}, err
	}
	batch.AsOf = batchAsOf(batch)
	version := "tushare-" + strings.ReplaceAll(batch.AsOf, "-", "")
	if err := syncer.writeBatch(ctx, batch, metadataName, version); err != nil {
		return TushareSyncSummary{}, err
	}
	return TushareSyncSummary{Provider: tushareProviderName, MetadataName: metadataName, Version: version, AsOf: batch.AsOf, Instruments: len(batch.Instruments), DailyBars: len(batch.Bars), DailyBasics: len(batch.Basics), Factors: len(batch.Factors), Indexes: len(batch.Indexes)}, nil
}

func (syncer *TushareSyncer) checkSourceIsolation(ctx context.Context) error {
	var provider string
	err := syncer.db.QueryRowContext(ctx, `SELECT provider FROM demo_seed_metadata WHERE seed_name = ?`, demoMetadataName).Scan(&provider)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check existing data source before Tushare sync: %w", err)
	}
	if provider != "" && provider != tushareProviderName {
		return fmt.Errorf("Tushare sync requires an isolated database; existing source is %q", provider)
	}
	return nil
}

func syncDateRange(options TushareSyncOptions, now time.Time) (string, string, []string, error) {
	start := strings.TrimSpace(options.StartDate)
	end := strings.TrimSpace(options.EndDate)
	lookback := options.LookbackDays
	if lookback <= 0 {
		lookback = 30
	}
	if end == "" {
		end = now.Format("20060102")
	}
	endTime, err := parseTushareDate(end)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse Tushare end date: %w", err)
	}
	if start == "" {
		start = endTime.AddDate(0, 0, -lookback).Format("20060102")
	}
	startTime, err := parseTushareDate(start)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse Tushare start date: %w", err)
	}
	if startTime.After(endTime) {
		return "", "", nil, errors.New("Tushare start date must not be after end date")
	}
	dates := make([]string, 0)
	for current := endTime; !current.Before(startTime); current = current.AddDate(0, 0, -1) {
		if len(dates) >= 3660 {
			return "", "", nil, errors.New("Tushare sync date range must not exceed 10 years")
		}
		dates = append(dates, current.Format("20060102"))
	}
	return startTime.Format("20060102"), endTime.Format("20060102"), dates, nil
}

func parseTushareDate(value string) (time.Time, error) {
	for _, layout := range []string{"20060102", "2006-01-02"} {
		if parsed, err := time.Parse(layout, strings.TrimSpace(value)); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q", value)
}

func (syncer *TushareSyncer) fetchInstruments(ctx context.Context) ([]tushareInstrument, error) {
	rows, err := syncer.client.Query(ctx, "stock_basic", map[string]string{"exchange": "", "list_status": "L"}, tushareStockBasicFields)
	if err != nil {
		return nil, fmt.Errorf("fetch Tushare stock basic: %w", err)
	}
	instruments := make([]tushareInstrument, 0, len(rows))
	for index, row := range rows {
		instrument, err := buildTushareInstrument(row)
		if err != nil {
			return nil, fmt.Errorf("parse Tushare stock basic row %d: %w", index, err)
		}
		instruments = append(instruments, instrument)
	}
	return instruments, nil
}

func buildTushareInstrument(row map[string]string) (tushareInstrument, error) {
	code, err := requiredTushareValue(row, "ts_code")
	if err != nil {
		return tushareInstrument{}, err
	}
	name, err := requiredTushareValue(row, "name")
	if err != nil {
		return tushareInstrument{}, err
	}
	listDate := normalizeTushareDate(row["list_date"])
	status := "active"
	if strings.TrimSpace(row["list_status"]) != "L" {
		status = "inactive"
	}
	return tushareInstrument{Code: code, Name: name, Industry: strings.TrimSpace(row["industry"]), Exchange: exchangeName(row["exchange"], code), Status: status, ListDate: listDate}, nil
}

func exchangeName(value, code string) string {
	switch strings.TrimSpace(value) {
	case "SSE":
		return "SSE"
	case "SZSE":
		return "SZSE"
	case "BSE":
		return "BSE"
	}
	if strings.HasSuffix(code, ".SH") {
		return "SSE"
	}
	if strings.HasSuffix(code, ".SZ") {
		return "SZSE"
	}
	return ""
}

func (syncer *TushareSyncer) fetchIndexes(ctx context.Context, startDate, endDate string) ([]tushareIndex, error) {
	indexes := make([]tushareIndex, 0)
	for code, name := range tushareIndexNames {
		rows, err := syncer.client.Query(ctx, "index_daily", map[string]string{"ts_code": code, "start_date": startDate, "end_date": endDate}, tushareIndexFields)
		if err != nil {
			return nil, fmt.Errorf("fetch Tushare index %s: %w", code, err)
		}
		for index, row := range rows {
			item, err := buildTushareIndex(row, name)
			if err != nil {
				return nil, fmt.Errorf("parse Tushare index %s row %d: %w", code, index, err)
			}
			indexes = append(indexes, item)
		}
	}
	return indexes, nil
}

func buildTushareIndex(row map[string]string, name string) (tushareIndex, error) {
	code, err := requiredTushareValue(row, "ts_code")
	if err != nil {
		return tushareIndex{}, err
	}
	tradeDate, err := requiredTushareDate(row, "trade_date")
	if err != nil {
		return tushareIndex{}, err
	}
	close, err := requiredTushareNumber(row, "close", 6)
	if err != nil {
		return tushareIndex{}, err
	}
	change, err := requiredTushareNumber(row, "change", 6)
	if err != nil {
		return tushareIndex{}, err
	}
	changePercent, err := requiredTushareNumber(row, "pct_chg", 6)
	if err != nil {
		return tushareIndex{}, err
	}
	return tushareIndex{Code: code, Name: name, TradeDate: tradeDate, Close: close, Change: change, ChangePercent: changePercent}, nil
}

type tushareDay struct {
	Bars    []tushareBar
	Basics  []tushareBasic
	Factors []tushareFactor
}

func filterTushareBatchToInstruments(batch *tushareBatch) {
	allowed := make(map[string]struct{}, len(batch.Instruments))
	for _, instrument := range batch.Instruments {
		allowed[instrument.Code] = struct{}{}
	}
	batch.Bars = filterTushareBars(batch.Bars, allowed)
	batch.Basics = filterTushareBasics(batch.Basics, allowed)
	batch.Factors = filterTushareFactors(batch.Factors, allowed)
}

func filterTushareBars(items []tushareBar, allowed map[string]struct{}) []tushareBar {
	filtered := items[:0]
	for _, item := range items {
		if _, ok := allowed[item.Code]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterTushareBasics(items []tushareBasic, allowed map[string]struct{}) []tushareBasic {
	filtered := items[:0]
	for _, item := range items {
		if _, ok := allowed[item.Code]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterTushareFactors(items []tushareFactor, allowed map[string]struct{}) []tushareFactor {
	filtered := items[:0]
	for _, item := range items {
		if _, ok := allowed[item.Code]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

type tushareDayResult struct {
	date string
	day  tushareDay
	err  error
}

func (syncer *TushareSyncer) fetchDays(ctx context.Context, dates []string, batch *tushareBatch) error {
	if len(dates) == 0 {
		return nil
	}
	workerCount := tushareSyncWorkerCount
	if len(dates) < workerCount {
		workerCount = len(dates)
	}
	jobs := make(chan string, len(dates))
	results := make(chan tushareDayResult, len(dates))
	for _, date := range dates {
		jobs <- date
	}
	close(jobs)

	var workers sync.WaitGroup
	workers.Add(workerCount)
	for index := 0; index < workerCount; index++ {
		go syncer.fetchDayWorker(ctx, jobs, results, &workers)
	}
	workers.Wait()
	close(results)
	return collectTushareDays(dates, results, batch)
}

func (syncer *TushareSyncer) fetchDayWorker(ctx context.Context, jobs <-chan string, results chan<- tushareDayResult, workers *sync.WaitGroup) {
	defer workers.Done()
	for date := range jobs {
		day, err := syncer.fetchDay(ctx, date)
		results <- tushareDayResult{date: date, day: day, err: err}
		if err != nil {
			return
		}
	}
}

func collectTushareDays(dates []string, results <-chan tushareDayResult, batch *tushareBatch) error {
	days := make(map[string]tushareDay, len(dates))
	for result := range results {
		if result.err != nil {
			return result.err
		}
		days[result.date] = result.day
	}
	for _, date := range dates {
		day, ok := days[date]
		if !ok {
			return fmt.Errorf("Tushare sync did not return data for %s", date)
		}
		batch.Bars = append(batch.Bars, day.Bars...)
		batch.Basics = append(batch.Basics, day.Basics...)
		batch.Factors = append(batch.Factors, day.Factors...)
	}
	return nil
}

func (syncer *TushareSyncer) fetchDay(ctx context.Context, date string) (tushareDay, error) {
	params := map[string]string{"ts_code": "", "trade_date": date}
	dailyRows, err := syncer.client.Query(ctx, "daily", params, tushareDailyFields)
	if err != nil {
		return tushareDay{}, fmt.Errorf("fetch Tushare daily %s: %w", date, err)
	}
	basicRows, err := syncer.client.Query(ctx, "daily_basic", params, tushareDailyBasicFields)
	if err != nil {
		return tushareDay{}, fmt.Errorf("fetch Tushare daily basic %s: %w", date, err)
	}
	factorRows, err := syncer.client.Query(ctx, "adj_factor", params, tushareFactorFields)
	if err != nil {
		return tushareDay{}, fmt.Errorf("fetch Tushare adjustment factors %s: %w", date, err)
	}
	day := tushareDay{Bars: make([]tushareBar, 0, len(dailyRows)), Basics: make([]tushareBasic, 0, len(basicRows)), Factors: make([]tushareFactor, 0, len(factorRows))}
	for index, row := range dailyRows {
		bar, err := buildTushareBar(row)
		if err != nil {
			return tushareDay{}, fmt.Errorf("parse Tushare daily %s row %d: %w", date, index, err)
		}
		day.Bars = append(day.Bars, bar)
	}
	for index, row := range basicRows {
		basic, err := buildTushareBasic(row)
		if err != nil {
			return tushareDay{}, fmt.Errorf("parse Tushare daily basic %s row %d: %w", date, index, err)
		}
		day.Basics = append(day.Basics, basic)
	}
	for index, row := range factorRows {
		factor, err := buildTushareFactor(row)
		if err != nil {
			return tushareDay{}, fmt.Errorf("parse Tushare factor %s row %d: %w", date, index, err)
		}
		day.Factors = append(day.Factors, factor)
	}
	return day, nil
}

func buildTushareBar(row map[string]string) (tushareBar, error) {
	code, err := requiredTushareValue(row, "ts_code")
	if err != nil {
		return tushareBar{}, err
	}
	tradeDate, err := requiredTushareDate(row, "trade_date")
	if err != nil {
		return tushareBar{}, err
	}
	prices := make([]string, 4)
	for index, field := range []string{"open", "high", "low", "close"} {
		prices[index], err = requiredTushareNumber(row, field, 6)
		if err != nil {
			return tushareBar{}, err
		}
	}
	volume, err := scaledInteger(row["vol"], 100)
	if err != nil {
		return tushareBar{}, fmt.Errorf("parse volume: %w", err)
	}
	amount, err := scaledTushareNumber(row["amount"], 1000, 6)
	if err != nil {
		return tushareBar{}, fmt.Errorf("parse amount: %w", err)
	}
	return tushareBar{Code: code, TradeDate: tradeDate, Open: prices[0], High: prices[1], Low: prices[2], Close: prices[3], Volume: volume, TurnoverAmount: amount}, nil
}

func buildTushareBasic(row map[string]string) (tushareBasic, error) {
	code, err := requiredTushareValue(row, "ts_code")
	if err != nil {
		return tushareBasic{}, err
	}
	tradeDate, err := requiredTushareDate(row, "trade_date")
	if err != nil {
		return tushareBasic{}, err
	}
	marketCap, err := optionalScaledTushareNumber(row["total_mv"], 10000, 6)
	if err != nil {
		return tushareBasic{}, fmt.Errorf("parse total market value: %w", err)
	}
	pb, err := optionalTushareNumber(row["pb"], 6)
	if err != nil {
		return tushareBasic{}, fmt.Errorf("parse PB: %w", err)
	}
	pe, err := optionalTushareNumber(row["pe_ttm"], 6)
	if err != nil {
		return tushareBasic{}, fmt.Errorf("parse PE TTM: %w", err)
	}
	ps, err := optionalTushareNumber(row["ps_ttm"], 6)
	if err != nil {
		return tushareBasic{}, fmt.Errorf("parse PS TTM: %w", err)
	}
	turnover, err := optionalTushareNumber(row["turnover_rate"], 6)
	if err != nil {
		return tushareBasic{}, fmt.Errorf("parse turnover rate: %w", err)
	}
	return tushareBasic{Code: code, TradeDate: tradeDate, MarketCap: marketCap, PB: pb, PETTM: pe, PSTTM: ps, TurnoverRate: turnover}, nil
}

func buildTushareFactor(row map[string]string) (tushareFactor, error) {
	code, err := requiredTushareValue(row, "ts_code")
	if err != nil {
		return tushareFactor{}, err
	}
	tradeDate, err := requiredTushareDate(row, "trade_date")
	if err != nil {
		return tushareFactor{}, err
	}
	value, err := requiredTushareNumber(row, "adj_factor", 8)
	if err != nil {
		return tushareFactor{}, err
	}
	return tushareFactor{Code: code, TradeDate: tradeDate, Value: value}, nil
}

func validateTushareBatch(batch tushareBatch) error {
	if len(batch.Instruments) == 0 {
		return errors.New("Tushare stock basic returned no active instruments")
	}
	if len(batch.Bars) == 0 || len(batch.Basics) == 0 || len(batch.Factors) == 0 {
		return errors.New("Tushare sync returned an incomplete market snapshot")
	}
	if len(batch.Indexes) == 0 {
		return errors.New("Tushare index daily returned no snapshots")
	}
	return nil
}

func batchAsOf(batch tushareBatch) string {
	latest := ""
	for _, date := range appendDates(batch.Bars, batch.Basics, batch.Factors, batch.Indexes) {
		if date > latest {
			latest = date
		}
	}
	return latest
}

func appendDates(bars []tushareBar, basics []tushareBasic, factors []tushareFactor, indexes []tushareIndex) []string {
	dates := make([]string, 0, len(bars)+len(basics)+len(factors)+len(indexes))
	for _, item := range bars {
		dates = append(dates, item.TradeDate)
	}
	for _, item := range basics {
		dates = append(dates, item.TradeDate)
	}
	for _, item := range factors {
		dates = append(dates, item.TradeDate)
	}
	for _, item := range indexes {
		dates = append(dates, item.TradeDate)
	}
	return dates
}

func (syncer *TushareSyncer) writeBatch(ctx context.Context, batch tushareBatch, metadataName, version string) error {
	tx, err := syncer.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Tushare sync: %w", err)
	}
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback()
		}
	}()
	if err := writeTushareInstruments(ctx, tx, batch.Instruments, batch.AsOf); err != nil {
		return err
	}
	if err := writeTushareSectors(ctx, tx, batch.Instruments); err != nil {
		return err
	}
	if err := writeTushareBars(ctx, tx, batch.Bars); err != nil {
		return err
	}
	if err := writeTushareBasics(ctx, tx, batch.Basics, version); err != nil {
		return err
	}
	if err := writeTushareFactors(ctx, tx, batch.Factors); err != nil {
		return err
	}
	if err := writeTushareIndexes(ctx, tx, batch.Indexes); err != nil {
		return err
	}
	if err := writeTushareMetadata(ctx, tx, metadataName, version, batch.AsOf); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Tushare sync: %w", err)
	}
	rollback = false
	return nil
}

func writeTushareInstruments(ctx context.Context, tx *sql.Tx, instruments []tushareInstrument, asOf string) error {
	const prefix = `INSERT INTO instruments (code, name, exchange, status, as_of) VALUES `
	const suffix = ` ON DUPLICATE KEY UPDATE name = VALUES(name), exchange = VALUES(exchange), status = VALUES(status), as_of = VALUES(as_of)`
	for offset := 0; offset < len(instruments); offset += tushareWriteChunkSize {
		end := minTushareIndex(offset+tushareWriteChunkSize, len(instruments))
		args := make([]any, 0, (end-offset)*5)
		for _, instrument := range instruments[offset:end] {
			listDate := instrument.ListDate
			if listDate == "" {
				listDate = asOf
			}
			args = append(args, instrument.Code, instrument.Name, instrument.Exchange, instrument.Status, listDate)
		}
		if err := executeTushareChunk(ctx, tx, prefix, suffix, end-offset, 5, args, "upsert Tushare instruments"); err != nil {
			return err
		}
	}
	return nil
}

func writeTushareSectors(ctx context.Context, tx *sql.Tx, instruments []tushareInstrument) error {
	for _, instrument := range instruments {
		industry := strings.TrimSpace(instrument.Industry)
		if industry == "" {
			continue
		}
		code := tushareIndustryCode(industry)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sector_categories (code, name) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE name = VALUES(name)`, code, industry); err != nil {
			return fmt.Errorf("upsert Tushare sector %q: %w", industry, err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sector_memberships (sector_code, instrument_code) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE sector_code = VALUES(sector_code)`, code, instrument.Code); err != nil {
			return fmt.Errorf("upsert Tushare sector membership %q/%q: %w", code, instrument.Code, err)
		}
	}
	return nil
}

func tushareIndustryCode(industry string) string {
	digest := sha1.Sum([]byte(industry))
	return "TS-" + hex.EncodeToString(digest[:])[:20]
}

func writeTushareBars(ctx context.Context, tx *sql.Tx, bars []tushareBar) error {
	const prefix = `INSERT INTO daily_bars (instrument_code, trade_date, open_price, high_price, low_price, close_price, volume, turnover_amount) VALUES `
	const suffix = ` ON DUPLICATE KEY UPDATE open_price = VALUES(open_price), high_price = VALUES(high_price), low_price = VALUES(low_price), close_price = VALUES(close_price), volume = VALUES(volume), turnover_amount = VALUES(turnover_amount)`
	for offset := 0; offset < len(bars); offset += tushareWriteChunkSize {
		end := minTushareIndex(offset+tushareWriteChunkSize, len(bars))
		args := make([]any, 0, (end-offset)*8)
		for _, bar := range bars[offset:end] {
			args = append(args, bar.Code, bar.TradeDate, bar.Open, bar.High, bar.Low, bar.Close, bar.Volume, bar.TurnoverAmount)
		}
		if err := executeTushareChunk(ctx, tx, prefix, suffix, end-offset, 8, args, "upsert Tushare daily bars"); err != nil {
			return err
		}
	}
	return nil
}

func writeTushareBasics(ctx context.Context, tx *sql.Tx, basics []tushareBasic, version string) error {
	if err := writeTushareDailyBasics(ctx, tx, basics); err != nil {
		return err
	}
	if err := writeTushareMetricRows(ctx, tx, basics); err != nil {
		return err
	}
	return writeTushareValuationRows(ctx, tx, basics, version)
}

func writeTushareDailyBasics(ctx context.Context, tx *sql.Tx, basics []tushareBasic) error {
	const prefix = `INSERT INTO daily_basic (instrument_code, trade_date, market_cap, pb) VALUES `
	const suffix = ` ON DUPLICATE KEY UPDATE market_cap = VALUES(market_cap), pb = VALUES(pb)`
	for offset := 0; offset < len(basics); offset += tushareWriteChunkSize {
		end := minTushareIndex(offset+tushareWriteChunkSize, len(basics))
		args := make([]any, 0, (end-offset)*4)
		for _, basic := range basics[offset:end] {
			args = append(args, basic.Code, basic.TradeDate, nullableTushareValue(basic.MarketCap), nullableTushareValue(basic.PB))
		}
		if err := executeTushareChunk(ctx, tx, prefix, suffix, end-offset, 4, args, "upsert Tushare daily basics"); err != nil {
			return err
		}
	}
	return nil
}

func writeTushareMetricRows(ctx context.Context, tx *sql.Tx, basics []tushareBasic) error {
	rows := make([][]any, 0, len(basics)*2)
	for _, basic := range basics {
		if basic.PETTM != "" {
			rows = append(rows, []any{basic.Code, basic.TradeDate, "pe_ttm", tushareProviderName + ".daily_basic", basic.PETTM})
		}
		if basic.TurnoverRate != "" {
			rows = append(rows, []any{basic.Code, basic.TradeDate, "turnover_rate", tushareProviderName + ".daily_basic", basic.TurnoverRate})
		}
	}
	const prefix = `INSERT INTO financial_metrics (instrument_code, metric_date, metric_name, basis, metric_value) VALUES `
	const suffix = ` ON DUPLICATE KEY UPDATE basis = VALUES(basis), metric_value = VALUES(metric_value)`
	return writeTushareRows(ctx, tx, prefix, suffix, rows, 5, "upsert Tushare financial metrics")
}

func writeTushareValuationRows(ctx context.Context, tx *sql.Tx, basics []tushareBasic, version string) error {
	rows := make([][]any, 0, len(basics))
	for _, basic := range basics {
		basis := tushareProviderName + ".daily_basic"
		rows = append(rows, []any{basic.Code, basic.TradeDate, nullableTushareValue(basic.PETTM), basis, nullableTushareValue(basic.PB), basis, nullableTushareValue(basic.PSTTM), basis, tushareProviderName, version, basic.TradeDate})
	}
	const prefix = `INSERT INTO stock_valuation_snapshots (instrument_code, trade_date, pe_ttm, pe_ttm_basis, pb, pb_basis, ps_ttm, ps_ttm_basis, provider, seed_version, as_of) VALUES `
	const suffix = ` ON DUPLICATE KEY UPDATE pe_ttm = VALUES(pe_ttm), pe_ttm_basis = VALUES(pe_ttm_basis), pb = VALUES(pb), pb_basis = VALUES(pb_basis), ps_ttm = VALUES(ps_ttm), ps_ttm_basis = VALUES(ps_ttm_basis), provider = VALUES(provider), seed_version = VALUES(seed_version), as_of = VALUES(as_of)`
	return writeTushareRows(ctx, tx, prefix, suffix, rows, 11, "upsert Tushare valuation snapshots")
}

func writeTushareFactors(ctx context.Context, tx *sql.Tx, factors []tushareFactor) error {
	latest := make(map[string]string, len(factors))
	rows := make([][]any, 0, len(factors))
	for _, factor := range factors {
		if factor.TradeDate > latest[factor.Code] {
			latest[factor.Code] = factor.TradeDate
		}
	}
	latestValue := make(map[string]string, len(latest))
	for _, factor := range factors {
		if factor.TradeDate == latest[factor.Code] {
			latestValue[factor.Code] = factor.Value
		}
	}
	for _, factor := range factors {
		qfq, err := divideTushareNumber(factor.Value, latestValue[factor.Code], 8)
		if err != nil {
			return fmt.Errorf("calculate Tushare qfq factor %q/%q: %w", factor.Code, factor.TradeDate, err)
		}
		rows = append(rows, []any{factor.Code, factor.TradeDate, qfq, factor.Value})
	}
	return writeTushareRows(ctx, tx,
		`INSERT INTO daily_adjustment_factors (instrument_code, trade_date, qfq_factor, hfq_factor) VALUES `,
		` ON DUPLICATE KEY UPDATE qfq_factor = VALUES(qfq_factor), hfq_factor = VALUES(hfq_factor)`, rows, 4,
		"upsert Tushare adjustment factors")
}

func writeTushareIndexes(ctx context.Context, tx *sql.Tx, indexes []tushareIndex) error {
	observedAt := time.Now().UTC().Format("2006-01-02 15:04:05")
	rows := make([][]any, 0, len(indexes))
	for _, index := range indexes {
		rows = append(rows, []any{index.Code, index.Name, index.TradeDate, observedAt, index.Close, index.Change, index.ChangePercent})
	}
	return writeTushareRows(ctx, tx,
		`INSERT INTO index_snapshots (code, name, trade_date, observed_at, close_price, change_amount, change_percent) VALUES `,
		` ON DUPLICATE KEY UPDATE name = VALUES(name), observed_at = VALUES(observed_at), close_price = VALUES(close_price), change_amount = VALUES(change_amount), change_percent = VALUES(change_percent)`, rows, 7,
		"upsert Tushare index snapshots")
}

func writeTushareRows(ctx context.Context, tx *sql.Tx, prefix, suffix string, rows [][]any, columns int, description string) error {
	for offset := 0; offset < len(rows); offset += tushareWriteChunkSize {
		end := minTushareIndex(offset+tushareWriteChunkSize, len(rows))
		args := make([]any, 0, (end-offset)*columns)
		for _, row := range rows[offset:end] {
			if len(row) != columns {
				return fmt.Errorf("%s row has %d values, want %d", description, len(row), columns)
			}
			args = append(args, row...)
		}
		if err := executeTushareChunk(ctx, tx, prefix, suffix, end-offset, columns, args, description); err != nil {
			return err
		}
	}
	return nil
}

func executeTushareChunk(ctx context.Context, tx *sql.Tx, prefix, suffix string, rows, columns int, args []any, description string) error {
	if rows == 0 {
		return nil
	}
	if len(args) != rows*columns {
		return fmt.Errorf("%s has %d arguments, want %d", description, len(args), rows*columns)
	}
	query := prefix + tusharePlaceholders(rows, columns) + suffix
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s batch: %w", description, err)
	}
	return nil
}

func tusharePlaceholders(rows, columns int) string {
	var builder strings.Builder
	for row := 0; row < rows; row++ {
		if row > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("(")
		for column := 0; column < columns; column++ {
			if column > 0 {
				builder.WriteString(",")
			}
			builder.WriteString("?")
		}
		builder.WriteString(")")
	}
	return builder.String()
}

func minTushareIndex(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func nullableTushareValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func writeTushareMetadata(ctx context.Context, tx *sql.Tx, metadataName, version, asOf string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO demo_seed_metadata (seed_name, seed_version, as_of, provider)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE seed_version = VALUES(seed_version), as_of = VALUES(as_of), provider = VALUES(provider)`, metadataName, version, asOf, tushareProviderName); err != nil {
		return fmt.Errorf("upsert Tushare source metadata: %w", err)
	}
	return nil
}

func requiredTushareValue(row map[string]string, field string) (string, error) {
	value := strings.TrimSpace(row[field])
	if value == "" {
		return "", fmt.Errorf("Tushare field %q is required", field)
	}
	return value, nil
}

func requiredTushareDate(row map[string]string, field string) (string, error) {
	value := normalizeTushareDate(row[field])
	if value == "" {
		return "", fmt.Errorf("Tushare date field %q is required", field)
	}
	return value, nil
}

func normalizeTushareDate(value string) string {
	parsed, err := parseTushareDate(value)
	if err != nil {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func requiredTushareNumber(row map[string]string, field string, decimals int) (string, error) {
	value, err := optionalTushareNumber(row[field], decimals)
	if err != nil {
		return "", fmt.Errorf("parse Tushare field %q: %w", field, err)
	}
	if value == "" {
		return "", fmt.Errorf("Tushare field %q is required", field)
	}
	return value, nil
}

func optionalTushareNumber(value string, decimals int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") || strings.EqualFold(value, "null") || strings.EqualFold(value, "nan") {
		return "", nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number %q", value)
	}
	return strconv.FormatFloat(parsed, 'f', decimals, 64), nil
}

func optionalScaledTushareNumber(value string, scale float64, decimals int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") || strings.EqualFold(value, "null") || strings.EqualFold(value, "nan") {
		return "", nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid number %q", value)
	}
	return strconv.FormatFloat(parsed*scale, 'f', decimals, 64), nil
}

func requiredTushareNumberValue(value string, scale float64) (int64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid non-negative number %q", value)
	}
	return int64(parsed*scale + 0.5), nil
}

func scaledInteger(value string, scale float64) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, errors.New("number is required")
	}
	return requiredTushareNumberValue(value, scale)
}

func scaledTushareNumber(value string, scale float64, decimals int) (string, error) {
	result, err := optionalScaledTushareNumber(value, scale, decimals)
	if err != nil {
		return "", err
	}
	if result == "" {
		return "", errors.New("number is required")
	}
	return result, nil
}

func divideTushareNumber(value, divisor string, decimals int) (string, error) {
	numerator, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return "", fmt.Errorf("invalid numerator %q", value)
	}
	denominator, err := strconv.ParseFloat(divisor, 64)
	if err != nil || denominator <= 0 {
		return "", fmt.Errorf("invalid divisor %q", divisor)
	}
	return strconv.FormatFloat(numerator/denominator, 'f', decimals, 64), nil
}
