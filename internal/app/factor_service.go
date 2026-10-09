package app

import (
	"context"
	"errors"
	"fmt"

	"stock-quant/contracts"
	factorports "stock-quant/internal/factor/ports"
	marketports "stock-quant/internal/market/ports"
	"stock-quant/internal/shared/apperror"
	"stock-quant/internal/shared/types"
)

// FactorLookbackDays is the fixed raw-factor input window in market trading days.
const FactorLookbackDays = 61

// FactorTask contains strategy inputs that are independent of snapshot storage.
type FactorTask struct {
	RequestID       string
	StrategyID      string
	StrategyVersion string
	AsOf            types.TradingDate
	ConfigHash      string
	Params          map[string]any
}

// FactorService composes a ready snapshot with an injected raw-factor runner.
type FactorService struct {
	snapshots marketports.SnapshotRepository
	runner    factorports.FactorRunner
}

// NewFactorService wires the snapshot and factor execution ports.
func NewFactorService(snapshots marketports.SnapshotRepository, runner factorports.FactorRunner) *FactorService {
	return &FactorService{snapshots: snapshots, runner: runner}
}

// ComputeRawFactors loads the required snapshot and returns the runner's protocol result.
func (service *FactorService) ComputeRawFactors(ctx context.Context, task FactorTask) (contracts.FactorResult, error) {
	if err := ctx.Err(); err != nil {
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeCancelled, "factor task context", err)
	}
	if !task.AsOf.Valid() {
		err := errors.New("valid trading date is required")
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeInvalidArgument, "factor task", err)
	}

	snapshot, err := service.snapshots.Load(ctx, task.AsOf, FactorLookbackDays)
	if err != nil {
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeDataIncomplete, "load factor snapshot", err)
	}
	if !snapshot.AsOf.Valid() || snapshot.AsOf != task.AsOf || snapshot.SnapshotHash == "" {
		err := errors.New("snapshot date or hash does not match the requested task")
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeDataIncomplete, "validate factor snapshot", err)
	}

	request := contracts.FactorRequest{
		SchemaVersion:   "v1",
		RequestID:       task.RequestID,
		Mode:            "raw_factors",
		StrategyID:      task.StrategyID,
		StrategyVersion: task.StrategyVersion,
		AsOf:            task.AsOf.String(),
		SnapshotHash:    snapshot.SnapshotHash,
		ConfigHash:      task.ConfigHash,
		DataRef: contracts.DataReference{
			Kind:   snapshot.Reference.Kind,
			Path:   snapshot.Reference.Path,
			SHA256: snapshot.Reference.SHA256,
			Format: snapshot.Reference.Format,
		},
		Params: task.Params,
	}
	result, err := service.runner.Compute(ctx, request)
	if err != nil {
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeStrategyFailed, "run raw-factor strategy", err)
	}
	if !resultMatchesRequest(request, result) {
		err := errors.New("factor runner response does not match request identity")
		return contracts.FactorResult{}, classifiedFailure(apperror.CodeInvalidWorkerResponse, "validate factor result", err)
	}
	return result, nil
}

func classifiedFailure(fallback apperror.Code, operation string, cause error) error {
	code := fallback
	if existing := apperror.CodeOf(cause); existing != "" && existing != apperror.CodeInternal {
		code = existing
	}
	if errors.Is(cause, context.Canceled) {
		code = apperror.CodeCancelled
	} else if errors.Is(cause, context.DeadlineExceeded) {
		code = apperror.CodeTimeout
	}
	return apperror.New(code, fmt.Errorf("%s: %w", operation, cause))
}

func resultMatchesRequest(request contracts.FactorRequest, result contracts.FactorResult) bool {
	return result.SchemaVersion == request.SchemaVersion &&
		result.RequestID == request.RequestID &&
		result.StrategyID == request.StrategyID &&
		result.StrategyVersion == request.StrategyVersion &&
		result.AsOf == request.AsOf &&
		result.SnapshotHash == request.SnapshotHash
}
