package screener

import (
	"context"
	"errors"
	"testing"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

type fakeSnapshotReader struct {
	input  domain.ExecutionInput
	err    error
	fields []string
}

func (reader *fakeSnapshotReader) ReadSnapshot(_ context.Context, fields []string) (domain.ExecutionInput, error) {
	reader.fields = append([]string(nil), fields...)
	return reader.input, reader.err
}

func TestServiceRunReadsRequestedFieldsAndReturnsSnapshot(t *testing.T) {
	value := "10"
	reader := &fakeSnapshotReader{input: domain.ExecutionInput{
		Universe: domain.Universe{ID: domain.ActiveAShareUniverse, Name: domain.ActiveAShareName},
		Eligible: []domain.Candidate{{Symbol: "000001.SZ", Name: "平安银行", Values: map[string]domain.Observation{"technical.close": {Value: &value, Basis: "daily_close", AsOf: "2024-06-28"}}}},
		Snapshot: domain.Snapshot{AsOf: "2024-06-28", FieldAsOf: map[string]string{"technical.close": "2024-06-28"}, DefinitionVersions: map[string]string{}},
		Source:   domain.Source{Mode: "demo", Provider: "mysql-demo-fixture", SeedVersion: "fnd-003-demo-v8", AsOf: "2024-06-28"},
	}}
	service, err := NewService(reader)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	response, err := service.Run(context.Background(), ScreenerRunRequest{Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Ranking: domain.Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(response.Results) != 1 || response.Results[0].Symbol != "000001.SZ" || response.Snapshot.AsOf != "2024-06-28" {
		t.Fatalf("response = %#v", response)
	}
	if len(reader.fields) != 1 || reader.fields[0] != "technical.close" {
		t.Fatalf("requested fields = %#v", reader.fields)
	}
}

func TestServiceRunPreservesDependencyFailure(t *testing.T) {
	reader := &fakeSnapshotReader{err: errors.New("database unavailable")}
	service, err := NewService(reader)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, err = service.Run(context.Background(), ScreenerRunRequest{Spec: domain.ScreenerSpec{UniverseID: domain.ActiveAShareUniverse, Ranking: domain.Ranking{FieldID: "technical.close", Direction: "desc"}, TopN: 1}})
	if err == nil || !errors.Is(err, reader.err) {
		t.Fatalf("Run() error = %v, want dependency failure", err)
	}
}
