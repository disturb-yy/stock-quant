package market

import (
	"context"
	"errors"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/market/domain"
)

type fakeSectorReader struct {
	snapshot SectorSnapshot
	err      error
}

func (reader fakeSectorReader) ReadSectorSnapshot(context.Context) (SectorSnapshot, error) {
	return reader.snapshot, reader.err
}

func TestSectorsReturnsSourceAndAggregatedPerformance(t *testing.T) {
	service, err := NewSectorService(fakeSectorReader{snapshot: SectorSnapshot{
		SeedVersion: "mkt-002-demo-v1",
		AsOf:        "2024-06-28",
		Components: []domain.SectorComponent{
			{SectorCode: "BANK", SectorName: "银行", InstrumentCode: "000002.SZ", InstrumentName: "股票乙", TradeDate: "2024-06-28", PreviousClose: "100", CurrentClose: "110"},
			{SectorCode: "BANK", SectorName: "银行", InstrumentCode: "000001.SZ", InstrumentName: "股票甲", TradeDate: "2024-06-28", PreviousClose: "100", CurrentClose: "101"},
		}}}, ProviderSelection{Mode: ModeDemo, Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSectorService() error = %v", err)
	}

	sectors, err := service.Sectors(context.Background())
	if err != nil {
		t.Fatalf("Sectors() error = %v", err)
	}
	if sectors.AsOf != "2024-06-28" || sectors.Source.Mode != ModeDemo || sectors.Source.Provider != DemoProviderName || sectors.Source.SeedVersion != "mkt-002-demo-v1" {
		t.Fatalf("market sectors identity = %#v, want as-of and source", sectors)
	}
	if len(sectors.Sectors) != 1 {
		t.Fatalf("sector count = %d, want 1", len(sectors.Sectors))
	}
	sector := sectors.Sectors[0]
	if sector.ChangePercent != "5.50" || sector.ComponentCount != 2 || sector.Leader.Code != "000002.SZ" || sector.Leader.ChangePercent != "10.00" {
		t.Fatalf("sector = %#v, want equal-weight result and leader", sector)
	}
}

func TestSectorsWrapsReaderError(t *testing.T) {
	wantErr := errors.New("database unavailable")
	service, err := NewSectorService(fakeSectorReader{err: wantErr}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSectorService() error = %v", err)
	}
	if _, err := service.Sectors(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Sectors() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestSectorsRejectsMixedTradeDates(t *testing.T) {
	service, err := NewSectorService(fakeSectorReader{snapshot: SectorSnapshot{
		SeedVersion: "mkt-002-demo-v1", AsOf: "2024-06-28",
		Components: []domain.SectorComponent{
			{SectorCode: "BANK", SectorName: "银行", InstrumentCode: "000001.SZ", InstrumentName: "股票甲", TradeDate: "2024-06-27", PreviousClose: "100", CurrentClose: "101"},
		},
	}}, ProviderSelection{Provider: DemoProviderName})
	if err != nil {
		t.Fatalf("NewSectorService() error = %v", err)
	}
	if _, err := service.Sectors(context.Background()); err == nil {
		t.Fatal("Sectors() error = nil, want mixed trade date error")
	}
}
