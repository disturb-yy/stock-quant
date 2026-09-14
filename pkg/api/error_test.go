package api

import (
	"encoding/json"
	"testing"
)

func TestErrorResponseJSON(t *testing.T) {
	response := NewErrorResponse(CodeInvalidPagination, "分页参数无效", map[string]any{
		"fields": map[string]string{"page": "必须是大于等于 1 的整数"},
	})

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if decoded.Code != CodeInvalidPagination {
		t.Fatalf("code = %q, want %q", decoded.Code, CodeInvalidPagination)
	}
	if decoded.Message != "分页参数无效" {
		t.Fatalf("message = %q, want %q", decoded.Message, "分页参数无效")
	}
	if decoded.Details == nil {
		t.Fatal("details must be preserved")
	}
}

func TestErrorResponseOmitsDetailsWhenEmpty(t *testing.T) {
	payload, err := json.Marshal(NewErrorResponse(CodeNotFound, "请求的资源不存在", nil))
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if _, ok := decoded["details"]; ok {
		t.Fatal("details must be omitted when empty")
	}
}
