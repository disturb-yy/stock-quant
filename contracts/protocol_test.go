package contracts

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func requestFixture(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("examples/request.json")
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}

	var request map[string]any
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatalf("decode request fixture: %v", err)
	}
	return request
}

func encodeFixture(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	return data
}

func TestDecodeRequestExample(t *testing.T) {
	data, err := os.ReadFile("examples/request.json")
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}

	request, err := DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}
	if request.SchemaVersion != "v1" || request.Mode != "raw_factors" {
		t.Fatalf("DecodeRequest() = %#v, want v1 raw_factors", request)
	}
	if request.DataRef.SHA256 == "" || request.AsOf != "2026-10-08" {
		t.Fatalf("DecodeRequest() lost contract fields: %#v", request)
	}
}

func TestDecodeRequestRejectsInvalidContract(t *testing.T) {
	tests := []struct {
		name   string
		change func(map[string]any)
	}{
		{name: "missing request id", change: func(request map[string]any) { delete(request, "request_id") }},
		{name: "wrong mode", change: func(request map[string]any) { request["mode"] = "final_score" }},
		{name: "invalid hash", change: func(request map[string]any) { request["snapshot_hash"] = "abc123" }},
		{name: "invalid calendar date", change: func(request map[string]any) { request["as_of"] = "2026-02-30" }},
		{name: "year zero date", change: func(request map[string]any) { request["as_of"] = "0000-01-01" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := requestFixture(t)
			test.change(request)
			if _, err := DecodeRequest(encodeFixture(t, request)); err == nil {
				t.Fatal("DecodeRequest() error = nil, want contract validation error")
			}
		})
	}
}

func TestDecodeRequestIgnoresObjectFieldOrder(t *testing.T) {
	request := encodeFixture(t, requestFixture(t))
	if _, err := DecodeRequest(request); err != nil {
		t.Fatalf("DecodeRequest() rejected reordered properties: %v", err)
	}
}

func resultFixture(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode result fixture: %v", err)
	}
	return result
}

func TestDecodeResultExample(t *testing.T) {
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}

	result, err := DecodeResult(data)
	if err != nil {
		t.Fatalf("DecodeResult() error = %v", err)
	}
	if result.SchemaVersion != "v1" || result.Status != "success" || len(result.Results) != 1 {
		t.Fatalf("DecodeResult() = %#v, want one successful factor result", result)
	}
}

func TestDecodeResultRejectsInvalidContract(t *testing.T) {
	tests := []struct {
		name   string
		change func(map[string]any)
	}{
		{name: "missing errors", change: func(result map[string]any) { delete(result, "errors") }},
		{name: "invalid error entry", change: func(result map[string]any) { result["errors"] = []any{map[string]any{"message": "bad"}} }},
		{name: "invalid calendar date", change: func(result map[string]any) { result["as_of"] = "2026-02-30" }},
		{name: "year zero date", change: func(result map[string]any) { result["as_of"] = "0000-01-01" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := resultFixture(t)
			test.change(result)
			if _, err := DecodeResult(encodeFixture(t, result)); err == nil {
				t.Fatal("DecodeResult() error = nil, want contract validation error")
			}
		})
	}
}

func TestDecodeResultRejectsMultipleJSONDocuments(t *testing.T) {
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}

	data = append(data, []byte("\n{}")...)
	if _, err := DecodeResult(data); err == nil {
		t.Fatal("DecodeResult() error = nil, want multiple-document error")
	}
}

func TestDecodeResultRejectsNumbersOutsideFiniteFloatRange(t *testing.T) {
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}
	data = bytes.Replace(data, []byte(`"momentum_60":0.1`), []byte(`"momentum_60":1e9999`), 1)
	if _, err := DecodeResult(data); err == nil {
		t.Fatal("DecodeResult() error = nil, want non-finite number error")
	}
}

func TestDecodeResultUsesFloat64SemanticsForLargeIntegers(t *testing.T) {
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}
	data = bytes.Replace(data, []byte(`"momentum_60":0.1`), []byte(`"momentum_60":9007199254740993`), 1)
	result, err := DecodeResult(data)
	if err != nil {
		t.Fatalf("DecodeResult() error = %v", err)
	}
	want := float64(9007199254740993)
	if result.Results[0].Momentum60 != want {
		t.Fatalf("Momentum60 = %.0f, want Go float64 conversion %.0f", result.Results[0].Momentum60, want)
	}
}

func TestEncodeRequestValidatesDTOAgainstSchema(t *testing.T) {
	data, err := os.ReadFile("examples/request.json")
	if err != nil {
		t.Fatalf("read request fixture: %v", err)
	}
	request, err := DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	encoded, err := EncodeRequest(request)
	if err != nil {
		t.Fatalf("EncodeRequest() error = %v", err)
	}
	if _, err := DecodeRequest(encoded); err != nil {
		t.Fatalf("DecodeRequest(EncodeRequest()) error = %v", err)
	}

	request.Mode = "final_score"
	if _, err := EncodeRequest(request); err == nil {
		t.Fatal("EncodeRequest() error = nil for invalid DTO")
	}
}

func TestEncodeResultValidatesDTOAgainstSchema(t *testing.T) {
	data, err := os.ReadFile("examples/result.json")
	if err != nil {
		t.Fatalf("read result fixture: %v", err)
	}
	result, err := DecodeResult(data)
	if err != nil {
		t.Fatalf("DecodeResult() error = %v", err)
	}

	encoded, err := EncodeResult(result)
	if err != nil {
		t.Fatalf("EncodeResult() error = %v", err)
	}
	if _, err := DecodeResult(encoded); err != nil {
		t.Fatalf("DecodeResult(EncodeResult()) error = %v", err)
	}
}
