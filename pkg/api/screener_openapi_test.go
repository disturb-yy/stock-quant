package api

import "testing"

func TestOpenAPIScreenerContract(t *testing.T) {
	document := OpenAPIDocument(false)
	paths := document["paths"].(map[string]any)
	path, ok := paths["/api/v1/screeners/run"].(map[string]any)
	if !ok {
		t.Fatal("OpenAPI missing screener run path")
	}
	post := path["post"].(map[string]any)
	if post["operationId"] != "runScreener" {
		t.Fatalf("operationId = %#v, want runScreener", post["operationId"])
	}
	responses := post["responses"].(map[string]any)
	for _, status := range []string{"200", "400", "404", "405", "503"} {
		if _, ok := responses[status]; !ok {
			t.Fatalf("screener responses missing %s", status)
		}
	}
	schemas := document["components"].(map[string]any)["schemas"].(map[string]any)
	spec := schemas["ScreenerSpec"].(map[string]any)
	registry := spec["x-field-registry"].([]map[string]any)
	if len(registry) == 0 {
		t.Fatal("screener field registry must be published")
	}
	if registry[0]["field_id"] != "market.market_cap" {
		t.Fatalf("first registry field = %#v", registry[0]["field_id"])
	}
	filter := schemas["ScreenerFilter"].(map[string]any)["properties"].(map[string]any)
	if len(filter["operator"].(map[string]any)["enum"].([]string)) != 7 {
		t.Fatal("screener operator enum must expose all supported operators")
	}
	for _, path := range []string{"/api/v1/screeners", "/api/v1/screeners/{id}"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("OpenAPI missing saved screener path %s", path)
		}
	}
	for _, schemaName := range []string{"Screener", "ScreenerCreateRequest", "ScreenerUpdateRequest", "ScreenerListResponse"} {
		if _, ok := schemas[schemaName]; !ok {
			t.Fatalf("OpenAPI missing saved screener schema %s", schemaName)
		}
	}
	updatePath := paths["/api/v1/screeners/{id}"].(map[string]any)["put"].(map[string]any)
	if _, ok := updatePath["responses"].(map[string]any)["409"]; !ok {
		t.Fatal("saved screener update must publish 409 conflict response")
	}
}
