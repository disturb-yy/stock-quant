package api

import (
	"encoding/json"
	"testing"
)

func TestOpenAPIDocumentContainsStableContract(t *testing.T) {
	payload, err := json.Marshal(OpenAPIDocument())
	if err != nil {
		t.Fatalf("marshal OpenAPI document: %v", err)
	}
	if !json.Valid(payload) {
		t.Fatal("OpenAPI document is not valid JSON")
	}

	var document struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Version string `json:"version"`
		} `json:"info"`
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]json.RawMessage `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("decode OpenAPI document: %v", err)
	}

	if document.OpenAPI != "3.0.3" {
		t.Fatalf("openapi = %q, want %q", document.OpenAPI, "3.0.3")
	}
	if document.Info.Version != "v1" {
		t.Fatalf("info.version = %q, want %q", document.Info.Version, "v1")
	}
	for _, path := range []string{"/api/v1/health", "/api/v1/markets/overview", "/api/v1/markets/sectors", "/api/v1/markets/signals", "/api/v1/markets/rankings", "/api/v1/stocks/{symbol}", "/api/v1/stocks/{symbol}/bars", "/api/v1/openapi.json"} {
		if _, ok := document.Paths[path]; !ok {
			t.Fatalf("OpenAPI paths missing %q", path)
		}
	}
	if _, ok := document.Paths["/api/v1/dev/demo-status"]; !ok {
		t.Fatal("OpenAPI paths missing development demo status")
	}
	for _, schema := range []string{"Response", "ErrorResponse", "PaginationRequest", "PaginationMeta", "PaginatedResponse", "HealthResponse", "DemoCounts", "DemoSampleStock", "DemoStatus", "MarketOverview", "MarketSectors", "MarketSector", "SectorLeader", "MarketDataSource", "MarketIndex", "MarketBreadth", "MarketTurnover", "MarketSignals", "SignalParameters", "SignalResult", "MarketRankings", "MarketRanking", "StockOverview", "StockQuote", "StockMetrics", "StockMetric", "StockSparkline", "StockSparklinePoint", "StockBars", "StockBar", "StockEffectiveRange", "StockBenchmark", "StockBenchmarkPoint"} {
		if _, ok := document.Components.Schemas[schema]; !ok {
			t.Fatalf("OpenAPI schemas missing %q", schema)
		}
	}
}

func TestOpenAPIStockBarsContract(t *testing.T) {
	document := OpenAPIDocument()
	paths := document["paths"].(map[string]any)
	get := paths["/api/v1/stocks/{symbol}/bars"].(map[string]any)["get"].(map[string]any)
	if get["operationId"] != "getStockBars" {
		t.Fatalf("stock bars operation = %#v, want getStockBars", get["operationId"])
	}
	parameters := get["parameters"].([]any)
	if len(parameters) != 7 {
		t.Fatalf("stock bars parameter count = %d, want 7", len(parameters))
	}
	assertParameterEnum := func(index int, name string, values []string) {
		t.Helper()
		parameter := parameters[index].(map[string]any)
		if parameter["name"] != name {
			t.Fatalf("parameter[%d] name = %#v, want %q", index, parameter["name"], name)
		}
		got := parameter["schema"].(map[string]any)["enum"].([]string)
		if len(got) != len(values) {
			t.Fatalf("parameter %q enum = %#v, want %#v", name, got, values)
		}
		for valueIndex := range values {
			if got[valueIndex] != values[valueIndex] {
				t.Fatalf("parameter %q enum = %#v, want %#v", name, got, values)
			}
		}
	}
	assertParameterEnum(1, "timeframe", []string{"1d"})
	assertParameterEnum(2, "adjust", []string{"none", "qfq", "hfq"})
	assertParameterEnum(3, "range", []string{"20d", "60d", "120d", "all"})
	assertParameterEnum(6, "benchmark", []string{"000300.SH"})

	responses := get["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "404", "405", "503"} {
		if _, ok := responses[status]; !ok {
			t.Fatalf("stock bars responses missing %s", status)
		}
	}
	responseSchema := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if responseSchema["$ref"] != "#/components/schemas/StockBars" {
		t.Fatalf("stock bars response schema = %#v, want StockBars", responseSchema)
	}
	schemas := document["components"].(map[string]any)["schemas"].(map[string]any)
	stockBar := schemas["StockBar"].(map[string]any)["properties"].(map[string]any)
	if stockBar["ma5"].(map[string]any)["nullable"] != true || stockBar["ma20"].(map[string]any)["nullable"] != true {
		t.Fatal("StockBar MA fields must be nullable")
	}
	stockBars := schemas["StockBars"].(map[string]any)["properties"].(map[string]any)
	if stockBars["benchmark"].(map[string]any)["nullable"] != true {
		t.Fatal("StockBars.benchmark must be nullable")
	}
	rangeSchema := schemas["StockEffectiveRange"].(map[string]any)["properties"].(map[string]any)
	if rangeSchema["from"].(map[string]any)["nullable"] != true || rangeSchema["to"].(map[string]any)["nullable"] != true {
		t.Fatal("StockEffectiveRange dates must be nullable")
	}
}

func TestOpenAPIStockOverviewContract(t *testing.T) {
	document := OpenAPIDocument()
	paths := document["paths"].(map[string]any)
	get := paths["/api/v1/stocks/{symbol}"].(map[string]any)["get"].(map[string]any)
	if get["operationId"] != "getStockOverview" {
		t.Fatalf("stock overview operation = %#v, want getStockOverview", get["operationId"])
	}
	parameters := get["parameters"].([]any)
	if len(parameters) != 1 || parameters[0].(map[string]any)["name"] != "symbol" || parameters[0].(map[string]any)["in"] != "path" {
		t.Fatalf("stock overview parameters = %#v, want required path symbol", parameters)
	}
	responses := get["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "404", "405", "503"} {
		if _, ok := responses[status]; !ok {
			t.Fatalf("stock overview responses missing %s", status)
		}
	}
	responseSchema := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if responseSchema["$ref"] != "#/components/schemas/StockOverview" {
		t.Fatalf("stock overview response schema = %#v", responseSchema)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	metrics := components["StockMetrics"].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"market_cap", "pe_ttm", "pb", "roe"} {
		if _, ok := metrics[field]; !ok {
			t.Fatalf("StockMetrics missing property %q", field)
		}
	}
	metric := components["StockMetric"].(map[string]any)
	if metric["properties"].(map[string]any)["value"].(map[string]any)["nullable"] != true {
		t.Fatal("StockMetric.value must be nullable for missing optional metrics")
	}
	sparklinePoint := components["StockSparklinePoint"].(map[string]any)
	properties := sparklinePoint["properties"].(map[string]any)
	for _, field := range []string{"trade_date", "open", "high", "low", "close"} {
		if _, ok := properties[field]; !ok {
			t.Fatalf("StockSparklinePoint missing property %q", field)
		}
	}
}

func TestOpenAPIMarketRankingsContract(t *testing.T) {
	document := OpenAPIDocument()
	paths := document["paths"].(map[string]any)
	get := paths["/api/v1/markets/rankings"].(map[string]any)["get"].(map[string]any)
	if get["operationId"] != "getMarketRankings" {
		t.Fatalf("market rankings operation = %#v, want getMarketRankings", get["operationId"])
	}
	parameters := get["parameters"].([]any)
	if len(parameters) != 3 {
		t.Fatalf("market rankings parameter count = %d, want 3", len(parameters))
	}
	metric := parameters[0].(map[string]any)
	if metric["name"] != "metric" || metric["required"] != true {
		t.Fatalf("metric parameter = %#v, want required metric", metric)
	}
	if got := metric["schema"].(map[string]any)["enum"].([]string); len(got) != 4 || got[0] != "gain" || got[3] != "turnover_rate" {
		t.Fatalf("metric enum = %#v, want four ranking metrics", got)
	}
	pageSchema := parameters[1].(map[string]any)["schema"].(map[string]any)
	pageSizeSchema := parameters[2].(map[string]any)["schema"].(map[string]any)
	if pageSchema["default"] != DefaultPage || pageSizeSchema["default"] != DefaultPageSize || pageSizeSchema["maximum"] != MaxPageSize {
		t.Fatalf("ranking pagination schema = %#v/%#v, want FND-002 defaults and boundary", pageSchema, pageSizeSchema)
	}
	responses := get["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "404", "405", "422", "503"} {
		if _, ok := responses[status]; !ok {
			t.Fatalf("market rankings responses missing %s", status)
		}
	}
	responseSchema := responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if responseSchema["$ref"] != "#/components/schemas/MarketRankings" {
		t.Fatalf("market rankings response schema = %#v, want MarketRankings", responseSchema)
	}
	components := document["components"].(map[string]any)["schemas"].(map[string]any)
	ranking := components["MarketRanking"].(map[string]any)
	properties := ranking["properties"].(map[string]any)
	for _, field := range []string{"rank", "code", "name", "value", "close", "change", "change_percent", "turnover_amount", "turnover_rate"} {
		if _, ok := properties[field]; !ok {
			t.Fatalf("MarketRanking missing property %q", field)
		}
	}
}

func TestOpenAPIMarketSignalsContract(t *testing.T) {
	document := OpenAPIDocument()
	paths := document["paths"].(map[string]any)
	path := paths["/api/v1/markets/signals"].(map[string]any)
	get := path["get"].(map[string]any)
	if get["operationId"] != "getMarketSignals" {
		t.Fatalf("market signals operation = %#v, want getMarketSignals", get["operationId"])
	}
	parameters := get["parameters"].([]any)
	if len(parameters) != 2 {
		t.Fatalf("market signals parameter count = %d, want 2", len(parameters))
	}
	typeParameter := parameters[0].(map[string]any)
	typeSchema := typeParameter["schema"].(map[string]any)
	if typeParameter["name"] != "type" || typeParameter["required"] != true {
		t.Fatalf("type parameter = %#v, want required type", typeParameter)
	}
	if got := typeSchema["enum"].([]string); len(got) != 4 || got[0] != "volume_surge" || got[3] != "strong" {
		t.Fatalf("type enum = %#v, want four signal types", got)
	}
	responses := get["responses"].(map[string]any)
	if _, ok := responses["400"]; !ok {
		t.Fatal("market signals responses missing 400")
	}
	if _, ok := responses["422"]; !ok {
		t.Fatal("market signals responses missing 422")
	}
	if _, ok := responses["503"]; !ok {
		t.Fatal("market signals responses missing 503")
	}
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	paramsSchema := schemas["SignalParameters"].(map[string]any)
	properties := paramsSchema["properties"].(map[string]any)
	if properties["window"].(map[string]any)["default"] != 20 || properties["multiple"].(map[string]any)["default"] != 1.5 || properties["top_percent"].(map[string]any)["default"] != 10 {
		t.Fatalf("signal parameter defaults = %#v, want 20/1.5/10", properties)
	}
	if string(responses["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["$ref"].(string)) != "#/components/schemas/MarketSignals" {
		t.Fatal("market signals 200 response must reference MarketSignals")
	}
}

func TestOpenAPIMarketSectorsContract(t *testing.T) {
	document := OpenAPIDocument()
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI paths have unexpected type")
	}
	path, ok := paths["/api/v1/markets/sectors"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI market sectors path has unexpected type")
	}
	get, ok := path["get"].(map[string]any)
	if !ok || get["operationId"] != "getMarketSectors" {
		t.Fatalf("market sectors operation = %#v, want getMarketSectors", get)
	}
	responses, ok := get["responses"].(map[string]any)
	if !ok {
		t.Fatal("market sectors responses have unexpected type")
	}
	response, ok := responses["200"].(map[string]any)
	if !ok {
		t.Fatal("market sectors 200 response has unexpected type")
	}
	content := response["content"].(map[string]any)
	applicationJSON := content["application/json"].(map[string]any)
	schema := applicationJSON["schema"].(map[string]any)
	if schema["$ref"] != "#/components/schemas/MarketSectors" {
		t.Fatalf("market sectors response schema = %#v, want MarketSectors", schema)
	}
}

func TestOpenAPIDocumentOmitsDevelopmentPathWhenDisabled(t *testing.T) {
	document := OpenAPIDocument(false)
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI paths have unexpected type")
	}
	if _, ok := paths["/api/v1/dev/demo-status"]; ok {
		t.Fatal("production OpenAPI must omit development demo status")
	}
}
