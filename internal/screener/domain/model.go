package domain

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

const (
	// ActiveAShareUniverse 是服务端唯一已发布的可执行 Universe。
	ActiveAShareUniverse = "cn_a_share_active"
	ActiveAShareName     = "A 股在市股票"
	MaxTopN              = 100
	MaxFilters           = 20
)

type ValueType string

const ValueTypeDecimal ValueType = "decimal"

type Operator string

const (
	OperatorEqual        Operator = "eq"
	OperatorNotEqual     Operator = "neq"
	OperatorGreater      Operator = "gt"
	OperatorGreaterEqual Operator = "gte"
	OperatorLess         Operator = "lt"
	OperatorLessEqual    Operator = "lte"
	OperatorBetween      Operator = "between"
)

var allOperators = []Operator{
	OperatorEqual, OperatorNotEqual, OperatorGreater, OperatorGreaterEqual,
	OperatorLess, OperatorLessEqual, OperatorBetween,
}

// FieldDefinition 是后端 canonical registry 的一项字段定义。
type FieldDefinition struct {
	ID        string     `json:"field_id"`
	Category  string     `json:"category"`
	Label     string     `json:"label"`
	Unit      string     `json:"unit"`
	ValueType ValueType  `json:"value_type"`
	Operators []Operator `json:"operators"`
	Sortable  bool       `json:"sortable"`
}

var fieldDefinitions = []FieldDefinition{
	{ID: "market.market_cap", Category: "Market", Label: "总市值", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "market.turnover_rate", Category: "Market", Label: "换手率", Unit: "%", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "technical.close", Category: "Technical", Label: "收盘价", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "technical.volume", Category: "Technical", Label: "成交量", Unit: "股", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "technical.turnover_amount", Category: "Technical", Label: "成交额", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "valuation.pe_ttm", Category: "Valuation", Label: "市盈率 TTM", Unit: "倍", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "valuation.pb", Category: "Valuation", Label: "市净率", Unit: "倍", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "valuation.ps_ttm", Category: "Valuation", Label: "市销率 TTM", Unit: "倍", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.revenue", Category: "Fundamental", Label: "营业收入", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.net_profit", Category: "Fundamental", Label: "净利润", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.total_assets", Category: "Fundamental", Label: "总资产", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.total_liabilities", Category: "Fundamental", Label: "总负债", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.total_equity", Category: "Fundamental", Label: "股东权益", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.operating_cash_flow", Category: "Fundamental", Label: "经营现金流", Unit: "CNY", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
	{ID: "fundamental.roe_pct", Category: "Fundamental", Label: "ROE", Unit: "%", ValueType: ValueTypeDecimal, Operators: allOperators, Sortable: true},
}

// FieldRegistry 返回 registry 的副本，避免调用方修改全局定义。
func FieldRegistry() []FieldDefinition {
	result := make([]FieldDefinition, len(fieldDefinitions))
	for index, definition := range fieldDefinitions {
		result[index] = definition
		result[index].Operators = append([]Operator(nil), definition.Operators...)
	}
	return result
}

func FieldByID(id string) (FieldDefinition, bool) {
	for _, definition := range fieldDefinitions {
		if definition.ID == id {
			definition.Operators = append([]Operator(nil), definition.Operators...)
			return definition, true
		}
	}
	return FieldDefinition{}, false
}

type Filter struct {
	FieldID  string   `json:"field_id"`
	Operator Operator `json:"operator"`
	Value    any      `json:"value"`
}

type Ranking struct {
	FieldID   string `json:"field_id"`
	Direction string `json:"direction"`
}

type ScreenerSpec struct {
	UniverseID string   `json:"universe_id"`
	Filters    []Filter `json:"filters"`
	Ranking    Ranking  `json:"ranking"`
	TopN       int      `json:"top_n"`
}

var ErrUniverseNotFound = errors.New("screener universe not found")

// ValidationError 表示请求规格不符合 canonical 契约。
type ValidationError struct {
	Fields map[string]string
}

func (err *ValidationError) Error() string { return "screener specification is invalid" }

func (err *ValidationError) Details() map[string]any {
	fields := make(map[string]any, len(err.Fields))
	for field, message := range err.Fields {
		fields[field] = message
	}
	return map[string]any{"fields": fields}
}

// NormalizeAndValidateSpec 只接受服务端 registry 定义的平面 AND 规格。
func NormalizeAndValidateSpec(spec ScreenerSpec) (ScreenerSpec, error) {
	normalized := spec
	normalized.UniverseID = strings.TrimSpace(spec.UniverseID)
	normalized.Ranking.FieldID = strings.TrimSpace(spec.Ranking.FieldID)
	normalized.Ranking.Direction = strings.ToLower(strings.TrimSpace(spec.Ranking.Direction))
	fields := make(map[string]string)
	if normalized.UniverseID == "" {
		fields["universe_id"] = "必须提供已发布的 Universe ID"
	} else if normalized.UniverseID != ActiveAShareUniverse {
		return ScreenerSpec{}, ErrUniverseNotFound
	}
	if len(spec.Filters) > MaxFilters {
		fields["filters"] = fmt.Sprintf("最多支持 %d 个条件", MaxFilters)
	}
	normalized.Filters = make([]Filter, len(spec.Filters))
	for index, filter := range spec.Filters {
		normalizedFilter, err := normalizeFilter(filter)
		if err != nil {
			fields[fmt.Sprintf("filters[%d]", index)] = err.Error()
			continue
		}
		normalized.Filters[index] = normalizedFilter
	}
	if normalized.Ranking.FieldID == "" {
		fields["ranking.field_id"] = "必须提供可排序字段"
	} else if definition, ok := FieldByID(normalized.Ranking.FieldID); !ok || !definition.Sortable {
		fields["ranking.field_id"] = "字段不存在或不可排序"
	}
	if normalized.Ranking.Direction != "asc" && normalized.Ranking.Direction != "desc" {
		fields["ranking.direction"] = "必须是 asc 或 desc"
	}
	if normalized.TopN < 1 || normalized.TopN > MaxTopN {
		fields["top_n"] = fmt.Sprintf("必须是 1 到 %d 之间的整数", MaxTopN)
	}
	if len(fields) > 0 {
		return ScreenerSpec{}, &ValidationError{Fields: fields}
	}
	return normalized, nil
}

func normalizeFilter(filter Filter) (Filter, error) {
	filter.FieldID = strings.TrimSpace(filter.FieldID)
	filter.Operator = Operator(strings.ToLower(strings.TrimSpace(string(filter.Operator))))
	definition, ok := FieldByID(filter.FieldID)
	if !ok {
		return Filter{}, errors.New("字段不存在或未发布")
	}
	if !containsOperator(definition.Operators, filter.Operator) {
		return Filter{}, fmt.Errorf("操作符 %q 不适用于该字段", filter.Operator)
	}
	value, err := normalizeDecimalValue(filter.Value, filter.Operator)
	if err != nil {
		return Filter{}, err
	}
	filter.Value = value
	return filter, nil
}

func containsOperator(operators []Operator, wanted Operator) bool {
	for _, operator := range operators {
		if operator == wanted {
			return true
		}
	}
	return false
}

func normalizeDecimalValue(value any, operator Operator) (any, error) {
	if operator == OperatorBetween {
		values, ok := value.([]any)
		if !ok || len(values) != 2 {
			return nil, errors.New("between 的 value 必须是两个十进制字符串")
		}
		first, err := decimalString(values[0])
		if err != nil {
			return nil, err
		}
		second, err := decimalString(values[1])
		if err != nil {
			return nil, err
		}
		if left, _ := new(big.Rat).SetString(first); left.Cmp(mustRat(second)) > 0 {
			return nil, errors.New("between 的下限不能大于上限")
		}
		return []any{first, second}, nil
	}
	return decimalString(value)
}

func decimalString(value any) (string, error) {
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", errors.New("value 必须是非空十进制字符串")
	}
	text = strings.TrimSpace(text)
	if _, ok := new(big.Rat).SetString(text); !ok {
		return "", fmt.Errorf("value %q 不是合法十进制数", text)
	}
	return text, nil
}

func mustRat(value string) *big.Rat {
	parsed, _ := new(big.Rat).SetString(value)
	return parsed
}

// Observation 是一个字段在执行快照中的实际值和来源元数据。
type Observation struct {
	Value             *string `json:"value"`
	Unit              string  `json:"unit"`
	Basis             string  `json:"basis"`
	AsOf              string  `json:"as_of"`
	UnavailableReason *string `json:"unavailable_reason"`
}

type Candidate struct {
	Symbol     string                 `json:"symbol"`
	Name       string                 `json:"name"`
	Industries []string               `json:"industries"`
	Values     map[string]Observation `json:"-"`
}

type Universe struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Snapshot struct {
	AsOf               string            `json:"as_of"`
	FieldAsOf          map[string]string `json:"field_as_of"`
	DefinitionVersions map[string]string `json:"definition_versions"`
}

type Source struct {
	Mode        string `json:"mode"`
	Provider    string `json:"provider"`
	SeedVersion string `json:"seed_version"`
	AsOf        string `json:"as_of"`
}

type ExecutionInput struct {
	Universe Universe
	Eligible []Candidate
	Snapshot Snapshot
	Source   Source
}

type FieldResult struct {
	FieldID           string  `json:"field_id"`
	Label             string  `json:"label"`
	Value             *string `json:"value"`
	Unit              string  `json:"unit"`
	Basis             string  `json:"basis"`
	AsOf              string  `json:"as_of"`
	UnavailableReason *string `json:"unavailable_reason"`
}

type RankingResult struct {
	FieldID           string  `json:"field_id"`
	Label             string  `json:"label"`
	Value             *string `json:"value"`
	Unit              string  `json:"unit"`
	Basis             string  `json:"basis"`
	AsOf              string  `json:"as_of"`
	UnavailableReason *string `json:"unavailable_reason"`
}

type ResultRow struct {
	Symbol     string        `json:"symbol"`
	Name       string        `json:"name"`
	Industries []string      `json:"industries"`
	Rank       int           `json:"rank"`
	Ranking    RankingResult `json:"ranking"`
	Fields     []FieldResult `json:"fields"`
}

type ExecutionResult struct {
	Spec     ScreenerSpec `json:"spec"`
	Snapshot Snapshot     `json:"snapshot"`
	Universe struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		EligibleCount int    `json:"eligible_count"`
	} `json:"universe"`
	MatchedCount  int         `json:"matched_count"`
	ReturnedCount int         `json:"returned_count"`
	Results       []ResultRow `json:"results"`
	Source        Source      `json:"source"`
}

// Execute 执行 AND 条件、稳定排序和 Top N 截断。
func Execute(spec ScreenerSpec, input ExecutionInput) (ExecutionResult, error) {
	normalized, err := NormalizeAndValidateSpec(spec)
	if err != nil {
		return ExecutionResult{}, err
	}
	if input.Universe.ID != normalized.UniverseID {
		return ExecutionResult{}, fmt.Errorf("snapshot universe %q does not match request %q", input.Universe.ID, normalized.UniverseID)
	}
	definition, _ := FieldByID(normalized.Ranking.FieldID)
	candidates := make([]Candidate, 0, len(input.Eligible))
	for _, candidate := range input.Eligible {
		if matchesAll(candidate, normalized.Filters) {
			observation := candidate.Values[normalized.Ranking.FieldID]
			if observation.Value != nil {
				if _, err := decimalString(*observation.Value); err != nil {
					return ExecutionResult{}, fmt.Errorf("candidate %q ranking field %q is invalid: %w", candidate.Symbol, normalized.Ranking.FieldID, err)
				}
				candidates = append(candidates, candidate)
			}
		}
	}
	sort.SliceStable(candidates, func(left, right int) bool {
		leftValue := mustRat(*candidates[left].Values[normalized.Ranking.FieldID].Value)
		rightValue := mustRat(*candidates[right].Values[normalized.Ranking.FieldID].Value)
		comparison := leftValue.Cmp(rightValue)
		if comparison == 0 {
			return candidates[left].Symbol < candidates[right].Symbol
		}
		if normalized.Ranking.Direction == "asc" {
			return comparison < 0
		}
		return comparison > 0
	})
	matchedCount := len(candidates)
	if len(candidates) > normalized.TopN {
		candidates = candidates[:normalized.TopN]
	}
	results := make([]ResultRow, 0, len(candidates))
	for index, candidate := range candidates {
		results = append(results, ResultRow{
			Symbol: candidate.Symbol, Name: candidate.Name, Industries: append([]string(nil), candidate.Industries...), Rank: index + 1,
			Ranking: resultForField(normalized.Ranking.FieldID, candidate.Values[normalized.Ranking.FieldID], definition),
			Fields:  fieldsForCandidate(candidate, normalized.Filters, normalized.Ranking.FieldID),
		})
	}
	result := ExecutionResult{Spec: normalized, Snapshot: input.Snapshot, MatchedCount: matchedCount, ReturnedCount: len(results), Results: results, Source: input.Source}
	result.Universe.ID, result.Universe.Name = input.Universe.ID, input.Universe.Name
	result.Universe.EligibleCount = len(input.Eligible)
	return result, nil
}

func matchesAll(candidate Candidate, filters []Filter) bool {
	for _, filter := range filters {
		observation := candidate.Values[filter.FieldID]
		if !matches(observation.Value, filter.Operator, filter.Value) {
			return false
		}
	}
	return true
}

func matches(value *string, operator Operator, expected any) bool {
	if value == nil {
		return false
	}
	actual, ok := new(big.Rat).SetString(strings.TrimSpace(*value))
	if !ok {
		return false
	}
	switch operator {
	case OperatorBetween:
		values := expected.([]any)
		lower := mustRat(values[0].(string))
		upper := mustRat(values[1].(string))
		return actual.Cmp(lower) >= 0 && actual.Cmp(upper) <= 0
	default:
		expectedValue := mustRat(expected.(string))
		comparison := actual.Cmp(expectedValue)
		switch operator {
		case OperatorEqual:
			return comparison == 0
		case OperatorNotEqual:
			return comparison != 0
		case OperatorGreater:
			return comparison > 0
		case OperatorGreaterEqual:
			return comparison >= 0
		case OperatorLess:
			return comparison < 0
		case OperatorLessEqual:
			return comparison <= 0
		}
	}
	return false
}

func resultForField(fieldID string, observation Observation, definition FieldDefinition) RankingResult {
	return RankingResult{FieldID: fieldID, Label: definition.Label, Value: observation.Value, Unit: definition.Unit, Basis: observation.Basis, AsOf: observation.AsOf, UnavailableReason: observation.UnavailableReason}
}

func fieldsForCandidate(candidate Candidate, filters []Filter, rankingField string) []FieldResult {
	fieldIDs := make([]string, 0, len(filters))
	seen := make(map[string]struct{}, len(filters))
	for _, filter := range filters {
		if _, ok := seen[filter.FieldID]; ok || filter.FieldID == rankingField {
			continue
		}
		seen[filter.FieldID] = struct{}{}
		fieldIDs = append(fieldIDs, filter.FieldID)
	}
	result := make([]FieldResult, 0, len(fieldIDs))
	for _, fieldID := range fieldIDs {
		definition, _ := FieldByID(fieldID)
		observation := candidate.Values[fieldID]
		result = append(result, FieldResult{FieldID: fieldID, Label: definition.Label, Value: observation.Value, Unit: definition.Unit, Basis: observation.Basis, AsOf: observation.AsOf, UnavailableReason: observation.UnavailableReason})
	}
	return result
}
