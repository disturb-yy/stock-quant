package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	runtimeconfig "github.com/disturb-yy/stock-quant/internal/config"
	datamysql "github.com/disturb-yy/stock-quant/internal/data/adapter/mysql"
	"github.com/disturb-yy/stock-quant/internal/data/adapter/source"
	"github.com/disturb-yy/stock-quant/internal/data/application"
	"github.com/disturb-yy/stock-quant/internal/data/config"
	dbinfra "github.com/disturb-yy/stock-quant/internal/infrastructure/database/mysql"
)

const defaultHTTPAddress = ":8357"

func main() {
	if err := run(); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func run() error {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return fmt.Errorf("load server config: %w", err)
	}
	dsn, err := runtimeconfig.LoadDatabaseDSNFromEnv()
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}
	ctx, cancelConnection := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelConnection()
	db, err := dbinfra.Open(ctx, dsn)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get mysql sql database: %w", err)
	}
	defer sqlDB.Close()
	dataSource, err := source.NewFromConfig(cfg, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return fmt.Errorf("create data source: %w", err)
	}
	repository := datamysql.NewRepository(db)
	service, err := application.NewService(repository, dataSource, cfg.MaxRetries, application.Options{})
	if err != nil {
		return fmt.Errorf("create data sync service: %w", err)
	}
	worker := application.NewWorker(service, time.Second)
	server := &http.Server{
		Addr:              httpAddress(),
		Handler:           newRouter(service),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return runRuntime(server, worker)
}

func runRuntime(server *http.Server, worker *application.Worker) error {
	ctx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	workerDone := make(chan error, 1)
	go func() {
		workerDone <- worker.Run(ctx)
	}()
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	signals := shutdownSignal()
	defer signal.Stop(signals)
	select {
	case err := <-serverErrors:
		cancelWorker()
		workerErr := <-workerDone
		if workerError := nonContextError(workerErr); workerError != nil {
			return fmt.Errorf("worker stopped: %w", workerError)
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen on %s: %w", server.Addr, err)
	case workerErr := <-workerDone:
		cancelWorker()
		if err := shutdownServer(server); err != nil {
			return err
		}
		if workerError := nonContextError(workerErr); workerError != nil {
			return fmt.Errorf("worker stopped: %w", workerError)
		}
		return nil
	case <-signals:
		shutdownErr := shutdownServer(server)
		// 先停止接收新请求，再取消并等待 Worker，避免进程退出时遗留后台任务。
		cancelWorker()
		workerErr := <-workerDone
		if shutdownErr != nil {
			return shutdownErr
		}
		if workerError := nonContextError(workerErr); workerError != nil {
			return fmt.Errorf("worker stopped during shutdown: %w", workerError)
		}
		return nil
	}
}

func shutdownServer(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}
	return nil
}

func nonContextError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func httpAddress() string {
	if address := os.Getenv("HTTP_ADDRESS"); address != "" {
		return address
	}
	return defaultHTTPAddress
}

func shutdownSignal() chan os.Signal {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	return signals
}
