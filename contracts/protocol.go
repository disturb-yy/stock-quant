package contracts

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	requestSchemaURI = "https://stockquant.local/contracts/strategy-request.v1.json"
	resultSchemaURI  = "https://stockquant.local/contracts/strategy-result.v1.json"
)

//go:embed strategy-request.schema.json
var requestSchemaFile []byte

//go:embed strategy-result.schema.json
var resultSchemaFile []byte

var requestSchema, requestSchemaErr = compileSchema(requestSchemaFile, requestSchemaURI)
var resultSchema, resultSchemaErr = compileSchema(resultSchemaFile, resultSchemaURI)

// DataReference 指向本地只读输入快照。
type DataReference struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Format string `json:"format"`
}

// FactorRequest 是 raw_factors v1 请求。
type FactorRequest struct {
	SchemaVersion   string         `json:"schema_version"`
	RequestID       string         `json:"request_id"`
	Mode            string         `json:"mode"`
	StrategyID      string         `json:"strategy_id"`
	StrategyVersion string         `json:"strategy_version"`
	AsOf            string         `json:"as_of"`
	SnapshotHash    string         `json:"snapshot_hash"`
	ConfigHash      string         `json:"config_hash"`
	DataRef         DataReference  `json:"data_ref"`
	Params          map[string]any `json:"params,omitempty"`
}

// RawFactors 是单只证券的一组原始因子。
type RawFactors struct {
	TSCode           string  `json:"ts_code"`
	Momentum60       float64 `json:"momentum_60"`
	Momentum20       float64 `json:"momentum_20"`
	AmountActivity20 float64 `json:"amount_activity_20"`
	Volatility20     float64 `json:"volatility_20"`
	MA20             float64 `json:"ma20"`
	AvgAmount20Yuan  float64 `json:"avg_amount_20_yuan"`
}

// ProtocolError 描述一条证券数据的质量错误。
type ProtocolError struct {
	TSCode  string `json:"ts_code,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FactorResult 是 raw_factors v1 响应。
type FactorResult struct {
	SchemaVersion   string          `json:"schema_version"`
	RequestID       string          `json:"request_id"`
	StrategyID      string          `json:"strategy_id"`
	StrategyVersion string          `json:"strategy_version"`
	AsOf            string          `json:"as_of"`
	SnapshotHash    string          `json:"snapshot_hash"`
	Status          string          `json:"status"`
	Results         []RawFactors    `json:"results"`
	Errors          []ProtocolError `json:"errors"`
}

// DecodeRequest 校验并解码一份完整的请求 JSON 文档。
func DecodeRequest(data []byte) (FactorRequest, error) {
	var request FactorRequest
	if err := decodeValidated(data, requestSchema, requestSchemaErr, &request); err != nil {
		return FactorRequest{}, fmt.Errorf("decode strategy request: %w", err)
	}
	return request, nil
}

// DecodeResult 校验并解码 stdout 中恰好一份响应 JSON 文档。
func DecodeResult(data []byte) (FactorResult, error) {
	var result FactorResult
	if err := decodeValidated(data, resultSchema, resultSchemaErr, &result); err != nil {
		return FactorResult{}, fmt.Errorf("decode strategy result: %w", err)
	}
	return result, nil
}

// EncodeRequest 校验 DTO 后编码为一份请求 JSON 文档。
func EncodeRequest(request FactorRequest) ([]byte, error) {
	return encodeValidated(request, requestSchema, requestSchemaErr, "strategy request")
}

// EncodeResult 校验 DTO 后编码为一份响应 JSON 文档。
func EncodeResult(result FactorResult) ([]byte, error) {
	return encodeValidated(result, resultSchema, resultSchemaErr, "strategy result")
}

func decodeValidated(data []byte, schema *jsonschema.Schema, schemaErr error, target any) error {
	if schemaErr != nil {
		return fmt.Errorf("load embedded contract schema: %w", schemaErr)
	}
	instance, err := decodeSingleJSON(data)
	if err != nil {
		return err
	}
	if err := schema.Validate(instance); err != nil {
		return fmt.Errorf("validate against schema: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode DTO: %w", err)
	}
	return nil
}

func encodeValidated(value any, schema *jsonschema.Schema, schemaErr error, description string) ([]byte, error) {
	if schemaErr != nil {
		return nil, fmt.Errorf("load embedded contract schema: %w", schemaErr)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode %s DTO: %w", description, err)
	}
	instance, err := decodeSingleJSON(data)
	if err != nil {
		return nil, fmt.Errorf("encode %s DTO: %w", description, err)
	}
	if err := schema.Validate(instance); err != nil {
		return nil, fmt.Errorf("validate encoded %s: %w", description, err)
	}
	return data, nil
}

func decodeSingleJSON(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var instance any
	if err := decoder.Decode(&instance); err != nil {
		return nil, fmt.Errorf("decode JSON document: %w", err)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("expected exactly one JSON document")
		}
		return nil, fmt.Errorf("trailing data after JSON document: %w", err)
	}
	return instance, nil
}

func compileSchema(source []byte, uri string) (*jsonschema.Schema, error) {
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("parse embedded schema %s: %w", uri, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	if err := compiler.AddResource(uri, document); err != nil {
		return nil, fmt.Errorf("register embedded schema %s: %w", uri, err)
	}
	schema, err := compiler.Compile(uri)
	if err != nil {
		return nil, fmt.Errorf("compile embedded schema %s: %w", uri, err)
	}
	return schema, nil
}
