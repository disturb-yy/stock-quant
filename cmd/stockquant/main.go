package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"stock-quant/internal/app"
)

type healthStatus struct {
	Status string `json:"status"`
	Scope  string `json:"scope"`
}

type healthChecker interface {
	Check(context.Context) (app.HealthStatus, error)
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, app.HealthService{}); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer, checker healthChecker) error {
	if len(args) != 1 || args[0] != "health" {
		return fmt.Errorf("usage: stockquant health")
	}

	status, err := checker.Check(ctx)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	response := healthStatus{Status: status.Status, Scope: status.Scope}
	if err := json.NewEncoder(output).Encode(response); err != nil {
		return fmt.Errorf("write health output: %w", err)
	}
	return nil
}
