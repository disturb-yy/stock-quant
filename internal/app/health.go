package app

import (
	"context"
	"fmt"
)

// HealthStatus describes the process-level liveness result.
type HealthStatus struct {
	Status string
	Scope  string
}

// HealthService provides a process-level health check without external I/O.
type HealthService struct{}

// Check reports that the CLI process is responsive unless its context is canceled.
func (HealthService) Check(ctx context.Context) (HealthStatus, error) {
	if err := ctx.Err(); err != nil {
		return HealthStatus{}, fmt.Errorf("health check context: %w", err)
	}
	return HealthStatus{Status: "ok", Scope: "process"}, nil
}
