package domain

import (
	"strings"
	"testing"
)

func TestNewManualStockPoolNormalizesOnlyClientMetadata(t *testing.T) {
	description := "  长期观察高股息股票  "
	pool, err := NewManualStockPool(StockPoolInput{Name: "  红利观察  ", Description: &description})
	if err != nil {
		t.Fatalf("NewManualStockPool() error = %v", err)
	}
	if pool.Name != "红利观察" || pool.Description == nil || *pool.Description != "长期观察高股息股票" {
		t.Fatalf("pool = %#v, want normalized metadata", pool)
	}
	if pool.Source != SourceManual || pool.ID != 0 || pool.MemberCount != 0 {
		t.Fatalf("pool = %#v, want server-owned defaults", pool)
	}
}

func TestNewManualStockPoolRejectsInvalidMetadata(t *testing.T) {
	_, err := NewManualStockPool(StockPoolInput{
		Name:        " ",
		Description: stringPointer(strings.Repeat("描述", MaxStockPoolDescriptionLength+1)),
	})
	validation, ok := err.(*ValidationError)
	if !ok || validation.Fields["name"] == "" || validation.Fields["description"] == "" {
		t.Fatalf("error = %#v, want name and description validation", err)
	}
}

func TestNewStockPoolMemberPreservesMarketCode(t *testing.T) {
	member, err := NewStockPoolMember("000001.SZ")
	if err != nil {
		t.Fatalf("NewStockPoolMember() error = %v", err)
	}
	if member.Symbol != "000001.SZ" || member.Name != "" {
		t.Fatalf("member = %#v, want original symbol and server-owned name", member)
	}
}

func TestNewStockPoolMemberRejectsTransformedOrMalformedCode(t *testing.T) {
	for _, symbol := range []string{"", "000001", " 平安银行 ", "000001.SZ "} {
		if _, err := NewStockPoolMember(symbol); err == nil {
			t.Fatalf("NewStockPoolMember(%q) error = nil, want validation", symbol)
		}
	}
}

func stringPointer(value string) *string {
	return &value
}
