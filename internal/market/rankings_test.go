package market

import (
	"context"
	"errors"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
)

type fakeRankingReader struct {
	snapshot RankingSnapshot
	err      error
}

func (reader fakeRankingReader) ReadRankingSnapshot(context.Context) (RankingSnapshot, error) {
	return reader.snapshot, reader.err
}

func TestRankingServiceSortsAndPaginatesLatestObservations(t *testing.T) {
	service, err := NewRankingService(fakeRankingReader{snapshot: RankingSnapshot{
		SeedVersion: "mkt-004-demo-v1",
		AsOf:        "2024-06-28",
		Observations: []domain.RankingObservation{
			{InstrumentCode: "600519.SH", InstrumentName: "贵州茅台", CurrentClose: "100", PreviousClose: "100", TurnoverAmount: "300", TurnoverRate: "1.20"},
			{InstrumentCode: "000001.SZ", InstrumentName: "平安银行", CurrentClose: "110", PreviousClose: "100", TurnoverAmount: "300", TurnoverRate: "2.40"},
			{InstrumentCode: "300750.SZ", InstrumentName: "宁德时代", CurrentClose: "90", PreviousClose: "100", TurnoverAmount: "500", TurnoverRate: "2.40"},
		},
	}}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}

	result, err := service.Rank(context.Background(), RankingRequest{Metric: "gain", Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("Rank() error = %v", err)
	}
	if result.Metric != domain.RankingMetricGain || result.AsOf != "2024-06-28" {
		t.Fatalf("result identity = %#v, want gain and as-of", result)
	}
	if result.Pagination.Page != 2 || result.Pagination.PageSize != 2 || result.Pagination.Total != 3 || result.Pagination.TotalPages != 2 {
		t.Fatalf("pagination = %#v, want page 2 of 2", result.Pagination)
	}
	if len(result.Data) != 1 || result.Data[0].Code != "300750.SZ" || result.Data[0].Rank != 3 {
		t.Fatalf("page data = %#v, want rank 3 loss stock", result.Data)
	}
}

func TestRankingServiceRejectsInvalidMetricPaginationAndHistory(t *testing.T) {
	tests := []struct {
		name     string
		request  RankingRequest
		snapshot RankingSnapshot
		wantType any
	}{
		{name: "invalid metric", request: RankingRequest{Metric: "unknown", Page: 1, PageSize: 20}, wantType: &RankingValidationError{}},
		{name: "invalid page", request: RankingRequest{Metric: "gain", Page: -1, PageSize: 20}, wantType: &RankingValidationError{}},
		{name: "invalid page size", request: RankingRequest{Metric: "gain", Page: 1, PageSize: 101}, wantType: &RankingValidationError{}},
		{name: "history insufficient", request: RankingRequest{Metric: "gain", Page: 1, PageSize: 20}, snapshot: RankingSnapshot{SeedVersion: "seed", AsOf: "2024-06-28", Observations: []domain.RankingObservation{{InstrumentCode: "A", InstrumentName: "甲", CurrentClose: "10"}}}, wantType: &RankingHistoryError{}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := NewRankingService(fakeRankingReader{snapshot: test.snapshot}, ProviderSelection{Provider: DemoProviderName})
			if err != nil {
				t.Fatalf("NewRankingService() error = %v", err)
			}
			_, err = service.Rank(context.Background(), test.request)
			if err == nil {
				t.Fatal("Rank() error = nil, want validation/history error")
			}
			switch test.wantType.(type) {
			case *RankingValidationError:
				var target *RankingValidationError
				if !errors.As(err, &target) {
					t.Fatalf("error = %v, want RankingValidationError", err)
				}
			case *RankingHistoryError:
				var target *RankingHistoryError
				if !errors.As(err, &target) {
					t.Fatalf("error = %v, want RankingHistoryError", err)
				}
			}
		})
	}
}

func TestRankingServiceWrapsReaderError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewRankingService(fakeRankingReader{err: wantErr}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	if _, err := service.Rank(context.Background(), RankingRequest{Metric: "gain", Page: 1, PageSize: 20}); !errors.Is(err, wantErr) {
		t.Fatalf("Rank() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestRankingServiceReturnsEmptyPageWhenLatestDataHasNoActiveStocks(t *testing.T) {
	service, err := NewRankingService(fakeRankingReader{snapshot: RankingSnapshot{
		SeedVersion: "seed", AsOf: "2024-06-28",
	}}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	result, err := service.Rank(context.Background(), RankingRequest{Metric: "gain", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("Rank() error = %v", err)
	}
	if result.Data == nil || len(result.Data) != 0 || result.Pagination.Total != 0 || result.Pagination.TotalPages != 0 {
		t.Fatalf("empty result = %#v, want empty data and zero pagination totals", result)
	}
}

func TestRankingServiceAcceptsMaximumPageSize(t *testing.T) {
	service, err := NewRankingService(fakeRankingReader{snapshot: RankingSnapshot{
		SeedVersion: "seed", AsOf: "2024-06-28", Observations: []domain.RankingObservation{
			{InstrumentCode: "A", InstrumentName: "甲", CurrentClose: "110", PreviousClose: "100", TurnoverAmount: "100", TurnoverRate: "1.00"},
			{InstrumentCode: "B", InstrumentName: "乙", CurrentClose: "100", PreviousClose: "100", TurnoverAmount: "200", TurnoverRate: "2.00"},
		},
	}}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	result, err := service.Rank(context.Background(), RankingRequest{Metric: "gain", Page: 1, PageSize: api.MaxPageSize})
	if err != nil {
		t.Fatalf("Rank() error = %v", err)
	}
	if result.Pagination.PageSize != api.MaxPageSize || len(result.Data) != 2 {
		t.Fatalf("pagination/data = %#v/%#v, want page size %d and all results", result.Pagination, result.Data, api.MaxPageSize)
	}
}
