package api

import (
	"net/url"
	"testing"
)

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name      string
		values    url.Values
		want      PaginationRequest
		wantErr   bool
		wantField string
	}{
		{
			name:   "defaults",
			values: url.Values{},
			want:   PaginationRequest{Page: DefaultPage, PageSize: DefaultPageSize},
		},
		{
			name:   "minimum values",
			values: url.Values{"page": {"1"}, "page_size": {"1"}},
			want:   PaginationRequest{Page: 1, PageSize: 1},
		},
		{
			name:   "maximum page size",
			values: url.Values{"page": {"4"}, "page_size": {"100"}},
			want:   PaginationRequest{Page: 4, PageSize: MaxPageSize},
		},
		{
			name:      "page below minimum",
			values:    url.Values{"page": {"0"}},
			wantErr:   true,
			wantField: "page",
		},
		{
			name:      "page size above maximum",
			values:    url.Values{"page_size": {"101"}},
			wantErr:   true,
			wantField: "page_size",
		},
		{
			name:      "non integer",
			values:    url.Values{"page": {"first"}},
			wantErr:   true,
			wantField: "page",
		},
		{
			name:      "duplicate value",
			values:    url.Values{"page_size": {"10", "20"}},
			wantErr:   true,
			wantField: "page_size",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParsePagination(test.values)
			if test.wantErr {
				if err == nil {
					t.Fatal("ParsePagination() error = nil, want error")
				}
				validationError, ok := err.(*PaginationValidationError)
				if !ok {
					t.Fatalf("error type = %T, want *PaginationValidationError", err)
				}
				if _, ok := validationError.Fields[test.wantField]; !ok {
					t.Fatalf("validation fields = %#v, want field %q", validationError.Fields, test.wantField)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParsePagination() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ParsePagination() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPaginationValidationErrorDetails(t *testing.T) {
	_, err := ParsePagination(url.Values{"page": {"0"}, "page_size": {"101"}})
	if err == nil {
		t.Fatal("ParsePagination() error = nil, want error")
	}

	validationError, ok := err.(*PaginationValidationError)
	if !ok {
		t.Fatalf("error type = %T, want *PaginationValidationError", err)
	}
	details := validationError.Details()
	fields, ok := details["fields"].(map[string]any)
	if !ok {
		t.Fatalf("details fields type = %T, want map[string]any", details["fields"])
	}
	if len(fields) != 2 {
		t.Fatalf("details fields count = %d, want 2", len(fields))
	}
}
