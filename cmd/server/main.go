package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/stock-ddd/pkg/config"
	"example.com/stock-ddd/pkg/logger"
)

func main() {
	serviceName := config.LoadServiceName()
	loggingConfig := config.LoadLogging()
	logOutput, err := logger.OpenOutput(logger.OutputConfig{
		Destination: loggingConfig.Output,
		Directory:   loggingConfig.Directory,
		Service:     serviceName,
	})
	if err != nil {
		log.Fatalf("initialize log output: %v", err)
	}
	defer func() {
		if err := logOutput.Close(); err != nil {
			log.Printf("close log output: %v", err)
		}
	}()

	applicationLogger, err := logger.New(logger.Config{
		Service:     serviceName,
		Environment: loggingConfig.Environment,
		Level:       loggingConfig.Level,
		Format:      loggingConfig.Format,
		Output:      logOutput,
	})
	if err != nil {
		if closeErr := logOutput.Close(); closeErr != nil {
			log.Printf("close log output: %v", closeErr)
		}
		log.Fatalf("initialize application logger: %v", err)
	}

	applicationLogger.Info("application logger initialized")
	server := newHTTPServer(applicationLogger)
	applicationLogger.Info("HTTP server starting", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		applicationLogger.Error("HTTP server stopped", "error", err)
		if closeErr := logOutput.Close(); closeErr != nil {
			applicationLogger.Error("close log output", "error", closeErr)
		}
		os.Exit(1)
	}
}
