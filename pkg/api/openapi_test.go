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
	for _, path := range []string{"/api/v1/health", "/api/v1/markets/overview", "/api/v1/markets/sectors", "/api/v1/openapi.json"} {
		if _, ok := document.Paths[path]; !ok {
			t.Fatalf("OpenAPI paths missing %q", path)
		}
	}
	if _, ok := document.Paths["/api/v1/dev/demo-status"]; !ok {
		t.Fatal("OpenAPI paths missing development demo status")
	}
	for _, schema := range []string{"Response", "ErrorResponse", "PaginationRequest", "PaginationMeta", "PaginatedResponse", "HealthResponse", "DemoCounts", "DemoSampleStock", "DemoStatus", "MarketOverview", "MarketSectors", "MarketSector", "SectorLeader", "MarketDataSource", "MarketIndex", "MarketBreadth", "MarketTurnover"} {
		if _, ok := document.Components.Schemas[schema]; !ok {
			t.Fatalf("OpenAPI schemas missing %q", schema)
		}
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
