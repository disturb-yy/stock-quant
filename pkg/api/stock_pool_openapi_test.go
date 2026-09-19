package api

import "testing"

func TestOpenAPIStockPoolContract(t *testing.T) {
	document := OpenAPIDocument(false)
	paths := document["paths"].(map[string]any)
	for _, path := range []string{"/api/v1/stock-pools", "/api/v1/stock-pools/{id}", "/api/v1/stock-pools/{id}/summary"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("OpenAPI missing stock pool path %q", path)
		}
	}
	for _, path := range []string{"/api/v1/stock-pools/{id}/members", "/api/v1/stock-pools/{id}/members/{symbol}"} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("OpenAPI missing stock pool member path %q", path)
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
	for _, name := range []string{"StockPoolCreateRequest", "StockPool", "StockPoolSummary", "StockPoolSummarySource", "StockPoolIndustrySummary", "StockPoolIndustryBucket", "StockPoolMetricSummary", "StockPoolListResponse", "StockPoolMember", "StockPoolMemberAddRequest", "StockPoolMemberListResponse", "StockPoolMemberAddResponse", "StockPoolMemberDeleteResponse"} {
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
	summary := paths["/api/v1/stock-pools/{id}/summary"].(map[string]any)["get"].(map[string]any)
	if summary["operationId"] != "getStockPoolSummary" {
		t.Fatalf("stock pool summary operation = %#v", summary["operationId"])
	}
	if _, ok := summary["responses"].(map[string]any)["503"]; !ok {
		t.Fatal("stock pool summary must publish dependency response")
	}
	summarySource := schemas["StockPoolSummarySource"].(map[string]any)["properties"].(map[string]any)["type"].(map[string]any)
	if len(summarySource["enum"].([]string)) != 2 {
		t.Fatalf("summary source types = %#v, want manual/screener", summarySource["enum"])
	}
	members := paths["/api/v1/stock-pools/{id}/members"].(map[string]any)
	if _, ok := members["get"].(map[string]any); !ok {
		t.Fatal("stock pool members must publish GET")
	}
	if _, ok := members["post"].(map[string]any); !ok {
		t.Fatal("stock pool members must publish POST")
	}
	deleteMember := paths["/api/v1/stock-pools/{id}/members/{symbol}"].(map[string]any)["delete"].(map[string]any)
	if _, ok := deleteMember["responses"].(map[string]any)["404"]; !ok {
		t.Fatal("stock pool member delete must publish not found response")
	}
}
