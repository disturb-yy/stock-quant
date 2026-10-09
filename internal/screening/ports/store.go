// Package ports defines screening persistence dependencies.
package ports

import (
	"context"

	"stock-quant/internal/screening/domain"
)

// ScreeningStore persists screening run identity before result storage is added.
type ScreeningStore interface {
	SaveRun(ctx context.Context, run domain.RunMetadata) error
}
