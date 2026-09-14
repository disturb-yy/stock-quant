package domain

import (
	"reflect"
	"testing"
)

func TestAggregateSectorPerformancesUsesEqualWeightAndSelectsLeader(t *testing.T) {
	performances, err := AggregateSectorPerformances([]SectorComponent{
		{SectorCode: "ALPHA", SectorName: "行业甲", InstrumentCode: "000002.SZ", InstrumentName: "股票乙", TradeDate: "2024-06-28", PreviousClose: "100", CurrentClose: "110"},
		{SectorCode: "ALPHA", SectorName: "行业甲", InstrumentCode: "000001.SZ", InstrumentName: "股票甲", TradeDate: "2024-06-28", PreviousClose: "100", CurrentClose: "101"},
		{SectorCode: "BETA", SectorName: "行业乙", InstrumentCode: "000003.SZ", InstrumentName: "股票丙", TradeDate: "2024-06-28", PreviousClose: "200", CurrentClose: "199"},
	})
	if err != nil {
		t.Fatalf("AggregateSectorPerformances() error = %v", err)
	}

	want := []SectorPerformance{
		{Code: "ALPHA", Name: "行业甲", ChangePercent: "5.50", ComponentCount: 2, Leader: SectorLeader{Code: "000002.SZ", Name: "股票乙", ChangePercent: "10.00"}},
		{Code: "BETA", Name: "行业乙", ChangePercent: "-0.50", ComponentCount: 1, Leader: SectorLeader{Code: "000003.SZ", Name: "股票丙", ChangePercent: "-0.50"}},
	}
	if !reflect.DeepEqual(performances, want) {
		t.Fatalf("performances = %#v, want %#v", performances, want)
	}
}

func TestAggregateSectorPerformancesRejectsInvalidComponent(t *testing.T) {
	_, err := AggregateSectorPerformances([]SectorComponent{{
		SectorCode: "ALPHA", SectorName: "行业甲", InstrumentCode: "000001.SZ", InstrumentName: "股票甲",
		TradeDate: "2024-06-28", PreviousClose: "0", CurrentClose: "101",
	}})
	if err == nil {
		t.Fatal("AggregateSectorPerformances() error = nil, want invalid close error")
	}
}

func TestAggregateSectorPerformancesRejectsDuplicateMember(t *testing.T) {
	component := SectorComponent{
		SectorCode: "ALPHA", SectorName: "行业甲", InstrumentCode: "000001.SZ", InstrumentName: "股票甲",
		TradeDate: "2024-06-28", PreviousClose: "100", CurrentClose: "101",
	}
	_, err := AggregateSectorPerformances([]SectorComponent{component, component})
	if err == nil {
		t.Fatal("AggregateSectorPerformances() error = nil, want duplicate member error")
	}
}
