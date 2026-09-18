package screener

import (
	"context"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

type fakeSavedScreenerStore struct {
	createdInput domain.SavedScreenerInput
	listPage     int
	listPageSize int
	listItems    []domain.SavedScreener
	listTotal    int64
}

func (store *fakeSavedScreenerStore) Create(_ context.Context, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	store.createdInput = input
	return domain.SavedScreener{ID: 7, Name: input.Name, Description: input.Description, Spec: input.Spec, Version: 1, CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()}, nil
}

func (store *fakeSavedScreenerStore) List(_ context.Context, page, pageSize int) ([]domain.SavedScreener, int64, error) {
	store.listPage, store.listPageSize = page, pageSize
	return store.listItems, store.listTotal, nil
}

func (store *fakeSavedScreenerStore) Get(context.Context, int64) (domain.SavedScreener, error) {
	return domain.SavedScreener{}, nil
}

func (store *fakeSavedScreenerStore) Update(context.Context, int64, int64, domain.SavedScreenerInput) (domain.SavedScreener, error) {
	return domain.SavedScreener{}, nil
}

func TestSavedScreenerServiceNormalizesNameAndSpecBeforePersisting(t *testing.T) {
	store := &fakeSavedScreenerStore{}
	service, err := NewSavedScreenerService(store)
	if err != nil {
		t.Fatalf("NewSavedScreenerService() error = %v", err)
	}

	description := "  仅保存条件  "
	result, err := service.Create(context.Background(), domain.SavedScreenerInput{
		Name:        "  低估值方案  ",
		Description: &description,
		Spec: domain.ScreenerSpec{
			UniverseID: " cn_a_share_active ",
			Filters:    []domain.Filter{{FieldID: " valuation.pe_ttm ", Operator: "LTE", Value: " 15.00 "}},
			Ranking:    domain.Ranking{FieldID: " technical.volume ", Direction: "DESC"},
			TopN:       10,
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Name != "低估值方案" || *result.Description != "仅保存条件" {
		t.Fatalf("result = %#v, want normalized identity", result)
	}
	if store.createdInput.Spec.Filters[0].FieldID != "valuation.pe_ttm" || store.createdInput.Spec.Filters[0].Value != "15.00" || store.createdInput.Spec.Ranking.Direction != "desc" {
		t.Fatalf("persisted input = %#v, want canonical spec", store.createdInput)
	}
}

func TestSavedScreenerServiceListUsesDefaultPagination(t *testing.T) {
	store := &fakeSavedScreenerStore{listItems: []domain.SavedScreener{}, listTotal: 0}
	service, err := NewSavedScreenerService(store)
	if err != nil {
		t.Fatalf("NewSavedScreenerService() error = %v", err)
	}
	result, err := service.List(context.Background(), SavedScreenerListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if store.listPage != 1 || store.listPageSize != 20 || result.Pagination.TotalPages != 0 {
		t.Fatalf("pagination = %#v, store request = %d/%d", result.Pagination, store.listPage, store.listPageSize)
	}
}

func TestNormalizeSavedScreenerInputRejectsInvalidNameAndSpec(t *testing.T) {
	_, err := domain.NormalizeSavedScreenerInput(domain.SavedScreenerInput{
		Name: " ",
		Spec: domain.ScreenerSpec{UniverseID: "unknown", Ranking: domain.Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1},
	})
	if err == nil {
		t.Fatal("NormalizeSavedScreenerInput() error = nil, want validation error")
	}
	validationError, ok := err.(*domain.ValidationError)
	if !ok || validationError.Fields["name"] == "" || validationError.Fields["spec.universe_id"] == "" {
		t.Fatalf("error = %#v, want name and universe validation fields", err)
	}
}
