package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

const (
	defaultSignalWindow     = 20
	defaultSignalMultiple   = 1.5
	defaultSignalTopPercent = 10
)

// SignalRequest 是信号扫描用例的原始请求参数。
type SignalRequest struct {
	Type   string
	Params string
}

// SignalParameters 是参数规范化后的 API 响应 DTO。
type SignalParameters struct {
	Window     int      `json:"window"`
	Multiple   *float64 `json:"multiple,omitempty"`
	TopPercent *int     `json:"top_percent,omitempty"`
}

// SignalResult 是一只命中股票的稳定身份和信号。
type SignalResult struct {
	Code   string            `json:"code"`
	Name   string            `json:"name"`
	Signal domain.SignalType `json:"signal"`
}

// MarketSignals 是 GET /api/v1/markets/signals 的成功响应。
type MarketSignals struct {
	Type    domain.SignalType `json:"type"`
	Params  SignalParameters  `json:"params"`
	AsOf    string            `json:"as_of"`
	Source  DataSource        `json:"source"`
	Signals []SignalResult    `json:"signals"`
}

// SignalSnapshot 是扫描所需的最新交易日行情快照。
type SignalSnapshot struct {
	SeedVersion string
	AsOf        string
	Series      []domain.SignalSeries
}

// SignalReader 是信号扫描所需的最小数据访问边界。
type SignalReader interface {
	ReadSignalSnapshot(context.Context, int) (SignalSnapshot, error)
}

// SignalValidationError 表示信号类型或参数不符合稳定契约。
type SignalValidationError struct {
	Fields map[string]string
}

func (e *SignalValidationError) Error() string {
	return "signal request parameters are invalid"
}

// Details 返回可直接放入统一错误响应的诊断字段。
func (e *SignalValidationError) Details() map[string]any {
	fields := make(map[string]any, len(e.Fields))
	for field, message := range e.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// SignalHistoryError 表示请求窗口超出可用历史日线范围。
type SignalHistoryError struct {
	InstrumentCode string
	Required       int
	Available      int
}

func (e *SignalHistoryError) Error() string {
	return fmt.Sprintf("instrument %q has insufficient signal history", e.InstrumentCode)
}

// Details 返回历史数据不足的安全诊断字段。
func (e *SignalHistoryError) Details() map[string]any {
	return map[string]any{
		"instrument_code": e.InstrumentCode,
		"required_bars":   e.Required,
		"available_bars":  e.Available,
	}
}

// SignalService 编排参数规范化、行情读取和领域规则计算。
type SignalService struct {
	reader    SignalReader
	selection ProviderSelection
}

// NewSignalService 创建市场信号扫描服务。
func NewSignalService(reader SignalReader, selection ProviderSelection) (*SignalService, error) {
	if reader == nil {
		return nil, errors.New("market signal reader is required")
	}
	if strings.TrimSpace(selection.Provider) == "" {
		return nil, errors.New("market signal provider is required")
	}
	return &SignalService{reader: reader, selection: selection}, nil
}

// Scan 按最新交易日执行一个内置信号扫描请求。
func (service *SignalService) Scan(ctx context.Context, request SignalRequest) (MarketSignals, error) {
	signalType, params, ruleParams, err := normalizeSignalRequest(request)
	if err != nil {
		return MarketSignals{}, err
	}
	snapshot, err := service.reader.ReadSignalSnapshot(ctx, ruleParams.Window)
	if err != nil {
		return MarketSignals{}, fmt.Errorf("read market signal snapshot: %w", err)
	}
	if err := validateSignalSnapshot(snapshot, ruleParams.Window); err != nil {
		return MarketSignals{}, err
	}
	matches, err := domain.ScanSignals(signalType, snapshot.Series, ruleParams)
	if err != nil {
		return MarketSignals{}, fmt.Errorf("scan market signal %q: %w", signalType, err)
	}
	return buildMarketSignals(snapshot, signalType, params, matches, service.selection), nil
}

func normalizeSignalRequest(request SignalRequest) (domain.SignalType, SignalParameters, domain.SignalRuleParameters, error) {
	signalType := domain.SignalType(strings.TrimSpace(request.Type))
	if !containsSignalType(signalType) {
		return "", SignalParameters{}, domain.SignalRuleParameters{}, newSignalValidationError("type", "必须是 volume_surge、breakout、new_high 或 strong")
	}
	raw, err := decodeSignalParameters(request.Params)
	if err != nil {
		return "", SignalParameters{}, domain.SignalRuleParameters{}, err
	}
	params, err := normalizeSignalParameters(signalType, raw)
	if err != nil {
		return "", SignalParameters{}, domain.SignalRuleParameters{}, err
	}
	ruleParams, err := params.toRuleParameters()
	if err != nil {
		return "", SignalParameters{}, domain.SignalRuleParameters{}, newSignalValidationError("params", "参数值无效")
	}
	return signalType, params, ruleParams, nil
}

func containsSignalType(signalType domain.SignalType) bool {
	for _, supported := range domain.SignalTypes() {
		if signalType == supported {
			return true
		}
	}
	return false
}

type rawSignalParameters struct {
	Window     *int
	Multiple   *float64
	TopPercent *int
}

func decodeSignalParameters(value string) (rawSignalParameters, error) {
	if strings.TrimSpace(value) == "" {
		return rawSignalParameters{}, nil
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(value))
	if err := decoder.Decode(&fields); err != nil {
		return rawSignalParameters{}, newSignalValidationError("params", "必须是合法 JSON 对象")
	}
	if fields == nil {
		return rawSignalParameters{}, newSignalValidationError("params", "必须是 JSON 对象")
	}
	var result rawSignalParameters
	for field, raw := range fields {
		switch field {
		case "window":
			parsed, err := decodeSignalInt(raw)
			if err != nil {
				return rawSignalParameters{}, newSignalValidationError("params.window", "必须是整数")
			}
			result.Window = &parsed
		case "multiple":
			parsed, err := decodeSignalFloat(raw)
			if err != nil {
				return rawSignalParameters{}, newSignalValidationError("params.multiple", "必须是数字")
			}
			result.Multiple = &parsed
		case "top_percent":
			parsed, err := decodeSignalInt(raw)
			if err != nil {
				return rawSignalParameters{}, newSignalValidationError("params.top_percent", "必须是整数")
			}
			result.TopPercent = &parsed
		default:
			return rawSignalParameters{}, newSignalValidationError("params."+field, "不是支持的参数")
		}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return rawSignalParameters{}, newSignalValidationError("params", "只能包含一个 JSON 对象")
	}
	return result, nil
}

func decodeSignalInt(raw json.RawMessage) (int, error) {
	var value *int
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return 0, errors.New("invalid integer")
	}
	return *value, nil
}

func decodeSignalFloat(raw json.RawMessage) (float64, error) {
	var value *float64
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return 0, errors.New("invalid number")
	}
	return *value, nil
}

func normalizeSignalParameters(signalType domain.SignalType, raw rawSignalParameters) (SignalParameters, error) {
	params := SignalParameters{Window: defaultSignalWindow}
	if raw.Window != nil {
		params.Window = *raw.Window
	}
	if params.Window != 20 && params.Window != 60 && params.Window != 120 {
		return SignalParameters{}, newSignalValidationError("params.window", "必须是 20、60 或 120")
	}
	switch signalType {
	case domain.SignalVolumeSurge:
		if raw.TopPercent != nil {
			return SignalParameters{}, newSignalValidationError("params.top_percent", "放量信号不支持该参数")
		}
		multiple := defaultSignalMultiple
		if raw.Multiple != nil {
			multiple = *raw.Multiple
		}
		if multiple != 1.5 && multiple != 2 {
			return SignalParameters{}, newSignalValidationError("params.multiple", "必须是 1.5 或 2")
		}
		params.Multiple = &multiple
	case domain.SignalStrong:
		if raw.Multiple != nil {
			return SignalParameters{}, newSignalValidationError("params.multiple", "强势信号不支持该参数")
		}
		topPercent := defaultSignalTopPercent
		if raw.TopPercent != nil {
			topPercent = *raw.TopPercent
		}
		if topPercent != 10 && topPercent != 20 {
			return SignalParameters{}, newSignalValidationError("params.top_percent", "必须是 10 或 20")
		}
		params.TopPercent = &topPercent
	default:
		if raw.Multiple != nil {
			return SignalParameters{}, newSignalValidationError("params.multiple", "该信号不支持该参数")
		}
		if raw.TopPercent != nil {
			return SignalParameters{}, newSignalValidationError("params.top_percent", "该信号不支持该参数")
		}
	}
	return params, nil
}

func (params SignalParameters) toRuleParameters() (domain.SignalRuleParameters, error) {
	ruleParams := domain.SignalRuleParameters{Window: params.Window}
	if params.Multiple != nil {
		multiple, ok := new(big.Rat).SetString(strconv.FormatFloat(*params.Multiple, 'f', -1, 64))
		if !ok {
			return domain.SignalRuleParameters{}, errors.New("invalid multiple")
		}
		ruleParams.Multiple = multiple
	}
	if params.TopPercent != nil {
		ruleParams.TopPercent = *params.TopPercent
	}
	return ruleParams, nil
}

func newSignalValidationError(field, message string) *SignalValidationError {
	return &SignalValidationError{Fields: map[string]string{field: message}}
}

func validateSignalSnapshot(snapshot SignalSnapshot, window int) error {
	if strings.TrimSpace(snapshot.AsOf) == "" || strings.TrimSpace(snapshot.SeedVersion) == "" {
		return errors.New("market signal as-of and seed version are required")
	}
	if len(snapshot.Series) == 0 {
		return &SignalHistoryError{Required: window + 1}
	}
	for _, series := range snapshot.Series {
		if len(series.Bars) < window+1 {
			return &SignalHistoryError{InstrumentCode: series.InstrumentCode, Required: window + 1, Available: len(series.Bars)}
		}
		if series.Bars[len(series.Bars)-1].TradeDate != snapshot.AsOf {
			return fmt.Errorf("instrument %q latest trade date does not match as-of", series.InstrumentCode)
		}
	}
	return nil
}

func buildMarketSignals(snapshot SignalSnapshot, signalType domain.SignalType, params SignalParameters, matches []domain.SignalMatch, selection ProviderSelection) MarketSignals {
	result := MarketSignals{
		Type: signalType, Params: params, AsOf: snapshot.AsOf,
		Source:  DataSource{Mode: selection.Mode, Provider: selection.Provider, SeedVersion: snapshot.SeedVersion},
		Signals: make([]SignalResult, 0, len(matches)),
	}
	for _, match := range matches {
		result.Signals = append(result.Signals, SignalResult{Code: match.InstrumentCode, Name: match.InstrumentName, Signal: match.SignalType})
	}
	return result
}
