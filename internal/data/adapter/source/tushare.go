package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

type TushareAdapter struct {
	client   *http.Client
	endpoint string
	token    string
}

type tushareResponse struct {
	Code int `json:"code"`
	Data struct {
		Fields []string            `json:"fields"`
		Items  [][]json.RawMessage `json:"items"`
	} `json:"data"`
}

func NewTushareAdapter(client *http.Client, endpoint, token string) (*TushareAdapter, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("invalid TUSHARE_ENDPOINT")
	}
	if token == "" {
		return nil, fmt.Errorf("TUSHARE_TOKEN is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &TushareAdapter{client: client, endpoint: endpoint, token: token}, nil
}

func (adapter *TushareAdapter) Provenance() domain.SourceProvenance {
	return domain.SourceProvenance{Provider: "tushare", Mode: "external"}
}

func (adapter *TushareAdapter) FetchBasicInfo(ctx context.Context) (domain.BasicInfoBatch, error) {
	response, err := adapter.request(ctx, "stock_basic", map[string]string{
		"list_status": "L",
	}, "ts_code,symbol,name,exchange,list_status")
	if err != nil {
		return domain.BasicInfoBatch{}, err
	}
	batch := domain.BasicInfoBatch{DataAsOf: timePtr(currentUTCDate())}
	for _, row := range response.Data.Items {
		instrument, ok, err := parseInstrument(response.Data.Fields, row, adapter.Provenance())
		if err != nil {
			return domain.BasicInfoBatch{}, err
		}
		if ok {
			batch.Items = append(batch.Items, instrument)
		}
	}
	return batch, nil
}

func currentUTCDate() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func (adapter *TushareAdapter) FetchDailyBars(ctx context.Context, dateRange domain.DateRange) (domain.DailyBarBatch, error) {
	if err := dateRange.ValidateFor(domain.TargetDailyBars); err != nil {
		return domain.DailyBarBatch{}, err
	}
	response, err := adapter.request(ctx, "daily", map[string]string{
		"start_date": formatDate(*dateRange.Start),
		"end_date":   formatDate(*dateRange.End),
	}, "ts_code,trade_date,open,high,low,close,vol")
	if err != nil {
		return domain.DailyBarBatch{}, err
	}
	batch := domain.DailyBarBatch{}
	for _, row := range response.Data.Items {
		bar, ok, err := parseDailyBar(response.Data.Fields, row, adapter.Provenance())
		if err != nil {
			return domain.DailyBarBatch{}, err
		}
		if ok {
			batch.Items = append(batch.Items, bar)
			batch.DataAsOf = latestDate(batch.DataAsOf, bar.TradeDate)
		}
	}
	return batch, nil
}

func (adapter *TushareAdapter) request(ctx context.Context, apiName string,
	params map[string]string, fields string) (tushareResponse, error) {
	payload := struct {
		APIName string            `json:"api_name"`
		Token   string            `json:"token"`
		Params  map[string]string `json:"params"`
		Fields  string            `json:"fields"`
	}{APIName: apiName, Token: adapter.token, Params: params, Fields: fields}
	body, err := json.Marshal(payload)
	if err != nil {
		return tushareResponse{}, fmt.Errorf("%w: encode %s request", domain.ErrDataSourceUnavailable, apiName)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, adapter.endpoint, bytes.NewReader(body))
	if err != nil {
		return tushareResponse{}, fmt.Errorf("%w: create %s request", domain.ErrDataSourceUnavailable, apiName)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := adapter.client.Do(request)
	if err != nil {
		return tushareResponse{}, fmt.Errorf("%w: request %s failed", domain.ErrDataSourceUnavailable, apiName)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return tushareResponse{}, fmt.Errorf("%w: request %s returned status %d",
			domain.ErrDataSourceUnavailable, apiName, response.StatusCode)
	}
	var result tushareResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 8<<20)).Decode(&result); err != nil {
		return tushareResponse{}, fmt.Errorf("%w: decode %s response", domain.ErrDataSourceUnavailable, apiName)
	}
	if result.Code != 0 {
		return tushareResponse{}, fmt.Errorf("%w: request %s returned code %d",
			domain.ErrDataSourceUnavailable, apiName, result.Code)
	}
	return result, nil
}

func parseInstrument(fields []string, row []json.RawMessage,
	source domain.SourceProvenance) (domain.Instrument, bool, error) {
	code, err := fieldString(fields, row, "ts_code")
	if err != nil {
		return domain.Instrument{}, false, err
	}
	market, symbol, ok := marketAndSymbol(code)
	if !ok {
		return domain.Instrument{}, false, nil
	}
	if value, valueErr := fieldString(fields, row, "exchange"); valueErr == nil && value != "" {
		market = strings.ToUpper(value)
	}
	if market != "SH" && market != "SZ" && market != "BJ" {
		return domain.Instrument{}, false, nil
	}
	name, err := fieldString(fields, row, "name")
	if err != nil {
		return domain.Instrument{}, false, err
	}
	status, err := fieldString(fields, row, "list_status")
	if err != nil {
		return domain.Instrument{}, false, err
	}
	return domain.Instrument{Symbol: symbol, Market: market, Name: name, Status: status,
		Source: source, UpdatedAt: time.Now().UTC()}, true, nil
}

func parseDailyBar(fields []string, row []json.RawMessage,
	source domain.SourceProvenance) (domain.DailyBar, bool, error) {
	code, err := fieldString(fields, row, "ts_code")
	if err != nil {
		return domain.DailyBar{}, false, err
	}
	market, symbol, ok := marketAndSymbol(code)
	if !ok || (market != "SH" && market != "SZ" && market != "BJ") {
		return domain.DailyBar{}, false, nil
	}
	tradeDate, err := fieldDate(fields, row, "trade_date")
	if err != nil {
		return domain.DailyBar{}, false, err
	}
	prices := make([]float64, 5)
	for index, name := range []string{"open", "high", "low", "close", "vol"} {
		prices[index], err = fieldFloat(fields, row, name)
		if err != nil {
			return domain.DailyBar{}, false, err
		}
	}
	return domain.DailyBar{Symbol: symbol, Market: market, TradeDate: tradeDate,
		Open: prices[0], High: prices[1], Low: prices[2], Close: prices[3], Volume: prices[4],
		Source: source, UpdatedAt: time.Now().UTC()}, true, nil
}

func fieldString(fields []string, row []json.RawMessage, name string) (string, error) {
	raw, err := field(fields, row, name)
	if err != nil {
		return "", err
	}
	if string(raw) == "null" {
		return "", fmt.Errorf("empty tushare field %q", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, nil
	}
	return strings.Trim(string(raw), `"`), nil
}

func fieldFloat(fields []string, row []json.RawMessage, name string) (float64, error) {
	value, err := fieldString(fields, row, name)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse tushare field %q: %w", name, err)
	}
	return parsed, nil
}

func fieldDate(fields []string, row []json.RawMessage, name string) (time.Time, error) {
	value, err := fieldString(fields, row, name)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(domain.DateLayout, value)
}

func field(fields []string, row []json.RawMessage, name string) (json.RawMessage, error) {
	for index, fieldName := range fields {
		if fieldName == name && index < len(row) {
			return row[index], nil
		}
	}
	return nil, fmt.Errorf("missing tushare field %q", name)
}

func marketAndSymbol(code string) (string, string, bool) {
	parts := strings.Split(strings.ToUpper(code), ".")
	if len(parts) != 2 || parts[0] == "" {
		return "", "", false
	}
	market := parts[1]
	if market != "SH" && market != "SZ" && market != "BJ" {
		return "", "", false
	}
	return market, parts[0], true
}

func formatDate(value time.Time) string {
	return value.UTC().Format(domain.DateLayout)
}
