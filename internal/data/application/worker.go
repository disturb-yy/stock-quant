package application

import (
	"context"
	"time"
)

type Worker struct {
	service      *Service
	pollInterval time.Duration
}

func NewWorker(service *Service, pollInterval time.Duration) *Worker {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &Worker{service: service, pollInterval: pollInterval}
}

func (worker *Worker) Run(ctx context.Context) error {
	for {
		didWork, err := worker.service.ExecuteNext(ctx)
		if err != nil {
			return err
		}
		if didWork {
			continue
		}
		timer := time.NewTimer(worker.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
