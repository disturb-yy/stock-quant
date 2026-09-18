package api

import "testing"

func TestOpenAPIStockPoolContract(t *testing.T) {
	document := OpenAPIDocument(false)
	paths := document["paths"].(map[string]any)
	for _, path := range []string{"/api/v1/stock-pools", "/api/v1/stock-pools/{id}"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("OpenAPI missing stock pool path %q", path)
		}
	}
	list := paths["/api/v1/stock-pools"].(map[string]any)["get"].(map[string]any)
	parameters := list["parameters"].([]any)
	if len(parameters) != 3 || parameters[0].(map[string]any)["name"] != "q" {
		t.Fatalf("stock pool list parameters = %#v, want q/page/page_size", parameters)
	}
	if list["description"] == "" {
		t.Fatal("stock pool list must publish stable sort and search semantics")
	}
	schemas := document["components"].(map[string]any)["schemas"].(map[string]any)
	for _, name := range []string{"StockPoolCreateRequest", "StockPool", "StockPoolListResponse"} {
		if _, ok := schemas[name]; !ok {
			t.Fatalf("OpenAPI missing stock pool schema %q", name)
		}
	}
	request := schemas["StockPoolCreateRequest"].(map[string]any)
	if request["additionalProperties"] != false {
		t.Fatal("create request must reject server-owned fields")
	}
	pool := schemas["StockPool"].(map[string]any)["properties"].(map[string]any)
	if source := pool["source"].(map[string]any)["enum"].([]string); len(source) != 1 || source[0] != "manual" {
		t.Fatalf("stock pool source = %#v, want manual only", source)
	}
	if _, ok := paths["/api/v1/stock-pools/{id}"].(map[string]any)["get"].(map[string]any)["responses"].(map[string]any)["404"]; !ok {
		t.Fatal("stock pool detail must publish not found response")
	}
}
