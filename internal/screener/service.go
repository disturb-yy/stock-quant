package screener

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
)

// ScreenerRunRequest 是 POST /api/v1/screeners/run 的应用请求。
type ScreenerRunRequest struct {
	Spec domain.ScreenerSpec `json:"spec"`
}

// ScreenerRunResponse 是一次临时选股执行结果，不持久化方案或结果行。
type ScreenerRunResponse = domain.ExecutionResult

// Source 是基础设施返回的执行来源元数据。
type Source = domain.Source

// SnapshotReader 读取同一执行快照内的已发布真实字段。
type SnapshotReader interface {
	ReadSnapshot(context.Context, []string) (domain.ExecutionInput, error)
}

// Service 编排规格校验、快照读取和 Domain 执行。
type Service struct {
	reader SnapshotReader
}

func NewService(reader SnapshotReader) (*Service, error) {
	if reader == nil {
		return nil, errors.New("screener snapshot reader is required")
	}
	return &Service{reader: reader}, nil
}

func (service *Service) Run(ctx context.Context, request ScreenerRunRequest) (ScreenerRunResponse, error) {
	normalized, err := domain.NormalizeAndValidateSpec(request.Spec)
	if err != nil {
		return ScreenerRunResponse{}, err
	}
	fieldIDs := requestedFieldIDs(normalized)
	input, err := service.reader.ReadSnapshot(ctx, fieldIDs)
	if err != nil {
		return ScreenerRunResponse{}, fmt.Errorf("read screener snapshot: %w", err)
	}
	if err := validateExecutionInput(input, normalized); err != nil {
		return ScreenerRunResponse{}, err
	}
	result, err := domain.Execute(normalized, input)
	if err != nil {
		return ScreenerRunResponse{}, fmt.Errorf("execute screener: %w", err)
	}
	return result, nil
}

func validateExecutionInput(input domain.ExecutionInput, spec domain.ScreenerSpec) error {
	if input.Universe.ID != spec.UniverseID || strings.TrimSpace(input.Universe.Name) == "" {
		return errors.New("screener execution universe is incomplete")
	}
	if strings.TrimSpace(input.Snapshot.AsOf) == "" {
		return errors.New("screener execution snapshot as-of is missing")
	}
	for _, fieldID := range requestedFieldIDs(spec) {
		if strings.TrimSpace(input.Snapshot.FieldAsOf[fieldID]) == "" {
			return fmt.Errorf("screener field %q snapshot as-of is missing", fieldID)
		}
	}
	if strings.TrimSpace(input.Source.Mode) == "" || strings.TrimSpace(input.Source.Provider) == "" || strings.TrimSpace(input.Source.SeedVersion) == "" || strings.TrimSpace(input.Source.AsOf) == "" {
		return errors.New("screener execution source is incomplete")
	}
	return nil
}

func requestedFieldIDs(spec domain.ScreenerSpec) []string {
	seen := map[string]struct{}{spec.Ranking.FieldID: {}}
	fieldIDs := make([]string, 0, len(spec.Filters)+1)
	for _, filter := range spec.Filters {
		if _, ok := seen[filter.FieldID]; ok {
			continue
		}
		seen[filter.FieldID] = struct{}{}
		fieldIDs = append(fieldIDs, filter.FieldID)
	}
	fieldIDs = append(fieldIDs, spec.Ranking.FieldID)
	return fieldIDs
}
