// Package ports defines the factor execution boundary.
package ports

import (
	"context"

	"stock-quant/contracts"
)

// FactorRunner executes a protocol v1 request and returns raw factors.
type FactorRunner interface {
	Compute(ctx context.Context, request contracts.FactorRequest) (contracts.FactorResult, error)
}
