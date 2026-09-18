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

func stringPointer(value string) *string {
	return &value
}
