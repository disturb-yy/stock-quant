package infrastructure

import (
	"testing"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

func TestSavedSpecSerializationUsesCanonicalShape(t *testing.T) {
	normalized, err := domain.NormalizeAndValidateSpec(domain.ScreenerSpec{
		UniverseID: " cn_a_share_active ",
		Filters:    []domain.Filter{{FieldID: " technical.close ", Operator: "GTE", Value: " 10.00 "}},
		Ranking:    domain.Ranking{FieldID: " technical.volume ", Direction: "DESC"},
		TopN:       2,
	})
	if err != nil {
		t.Fatalf("NormalizeAndValidateSpec() error = %v", err)
	}
	raw, err := marshalSavedSpec(normalized)
	if err != nil {
		t.Fatalf("marshalSavedSpec() error = %v", err)
	}
	decoded, err := decodeSavedSpec(raw)
	if err != nil {
		t.Fatalf("decodeSavedSpec() error = %v", err)
	}
	if decoded.UniverseID != domain.ActiveAShareUniverse || decoded.Filters[0].Operator != domain.OperatorGreaterEqual || decoded.Filters[0].Value != "10.00" || decoded.Ranking.Direction != "desc" {
		t.Fatalf("decoded spec = %#v, want canonical spec", decoded)
	}
}

func TestDecodeSavedSpecRejectsUnknownFields(t *testing.T) {
	_, err := decodeSavedSpec([]byte(`{"universe_id":"cn_a_share_active","filters":[],"ranking":{"field_id":"technical.close","direction":"desc"},"top_n":1,"private_state":true}`))
	if err == nil {
		t.Fatal("decodeSavedSpec() error = nil, want unknown field error")
	}
}
