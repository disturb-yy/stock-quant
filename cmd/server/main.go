package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/pkg/config"
	"github.com/disturb-yy/stock-quant/pkg/logger"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func run(ctx context.Context) error {
	httpAddress, err := config.LoadHTTPAddress()
	if err != nil {
		return fmt.Errorf("load HTTP address: %w", err)
	}
	applicationEnvironment := config.LoadEnvironment()
	applicationLogger, logOutput, err := newApplicationLogger()
	if err != nil {
		return err
	}
	defer closeLogOutput(logOutput)

	applicationLogger.Info("application logger initialized")
	demoStore, err := openDemoStore(ctx)
	if err != nil {
		applicationLogger.Error("open database", "error", err)
		return err
	}
	defer func() {
		if err := demoStore.Close(); err != nil {
			applicationLogger.Error("close database", "error", err)
		}
	}()
	if err := initializeMigrations(ctx, demoStore, applicationLogger); err != nil {
		applicationLogger.Error("initialize database migrations", "error", err)
		return err
	}

	statusReader, err := newStatusReader(applicationEnvironment, demoStore)
	if err != nil {
		applicationLogger.Error("initialize demo status service", "error", err)
		return err
	}
	server := newHTTPServer(applicationLogger, httpAddress, statusReader)
	applicationLogger.Info("HTTP server starting", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		applicationLogger.Error("HTTP server stopped", "error", err)
		return err
	}
	return nil
}

func newApplicationLogger() (*slog.Logger, io.WriteCloser, error) {
	serviceName := config.LoadServiceName()
	loggingConfig := config.LoadLogging()
	logOutput, err := logger.OpenOutput(logger.OutputConfig{
		Destination: loggingConfig.Output,
		Directory:   loggingConfig.Directory,
		Service:     serviceName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("initialize log output: %w", err)
	}
	applicationLogger, err := logger.New(logger.Config{
		Service:     serviceName,
		Environment: loggingConfig.Environment,
		Level:       loggingConfig.Level,
		Format:      loggingConfig.Format,
		Output:      logOutput,
	})
	if err != nil {
		if closeErr := logOutput.Close(); closeErr != nil {
			return nil, nil, fmt.Errorf("initialize application logger: %w; close log output: %v", err, closeErr)
		}
		return nil, nil, fmt.Errorf("initialize application logger: %w", err)
	}
	return applicationLogger, logOutput, nil
}

func closeLogOutput(logOutput io.WriteCloser) {
	if err := logOutput.Close(); err != nil {
		log.Printf("close log output: %v", err)
	}
}

func openDemoStore(ctx context.Context) (*infrastructure.Store, error) {
	database, err := infrastructure.Open(ctx, config.LoadDatabase())
	if err != nil {
		return nil, err
	}
	demoStore, err := infrastructure.NewStore(database)
	if err != nil {
		if closeErr := database.Close(); closeErr != nil {
			return nil, fmt.Errorf("initialize demo store: %w; close database: %v", err, closeErr)
		}
		return nil, fmt.Errorf("initialize demo store: %w", err)
	}
	return demoStore, nil
}

func newStatusReader(environment string, store demo.StatusStore) (demo.StatusReader, error) {
	if environment != "development" {
		return nil, nil
	}
	return demo.NewService(store, config.LoadDataProvider())
}
