package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"

	"github.com/disturb-yy/stock-quant/internal/demo"
	"github.com/disturb-yy/stock-quant/internal/demo/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/market"
	marketinfrastructure "github.com/disturb-yy/stock-quant/internal/market/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/pool"
	poolinfrastructure "github.com/disturb-yy/stock-quant/internal/pool/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/research"
	researchinfrastructure "github.com/disturb-yy/stock-quant/internal/research/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/screener"
	screenerinfrastructure "github.com/disturb-yy/stock-quant/internal/screener/infrastructure"
	"github.com/disturb-yy/stock-quant/internal/stock"
	stockinfrastructure "github.com/disturb-yy/stock-quant/internal/stock/infrastructure"
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
	database, err := openDemoDatabase(ctx)
	if err != nil {
		applicationLogger.Error("open database", "error", err)
		return err
	}
	defer func() {
		if err := database.Close(); err != nil {
			applicationLogger.Error("close database", "error", err)
		}
	}()
	demoStore, err := infrastructure.NewStore(database)
	if err != nil {
		applicationLogger.Error("initialize demo store", "error", err)
		return err
	}
	savedScreenerStore, err := screenerinfrastructure.NewMySQLSavedScreenerStore(database)
	if err != nil {
		applicationLogger.Error("initialize saved screener store", "error", err)
		return err
	}
	stockPoolStore, err := poolinfrastructure.NewMySQLStockPoolStore(database)
	if err != nil {
		applicationLogger.Error("initialize stock pool store", "error", err)
		return err
	}
	researchStore, err := researchinfrastructure.NewMySQLResearchStore(database)
	if err != nil {
		applicationLogger.Error("initialize research store", "error", err)
		return err
	}
	if err := initializeMigrations(ctx, demoStore, applicationLogger, savedScreenerStore, stockPoolStore, researchStore); err != nil {
		applicationLogger.Error("initialize database migrations", "error", err)
		return err
	}

	requestedProvider := config.LoadDataProvider()
	providerSelection := market.SelectProvider(requestedProvider, true)
	if market.IsRealProviderRequested(requestedProvider) {
		if err := syncTushare(ctx, database, config.LoadTushare(), applicationLogger); err != nil {
			applicationLogger.Error("synchronize Tushare data", "error", err)
			return err
		}
		providerSelection = market.SelectProviderWithAvailability(requestedProvider, true, true)
	}
	statusReader, err := newStatusReader(applicationEnvironment, demoStore, requestedProvider)
	if err != nil {
		applicationLogger.Error("initialize demo status service", "error", err)
		return err
	}
	overviewReader, err := marketinfrastructure.NewMySQLOverviewReader(database, providerSelection.MetadataName)
	if err != nil {
		applicationLogger.Error("initialize market overview reader", "error", err)
		return err
	}
	overviewService, err := market.NewOverviewService(
		overviewReader,
		providerSelection,
	)
	if err != nil {
		applicationLogger.Error("initialize market overview service", "error", err)
		return err
	}
	sectorService, err := market.NewSectorService(overviewReader, providerSelection)
	if err != nil {
		applicationLogger.Error("initialize market sector service", "error", err)
		return err
	}
	signalService, err := market.NewSignalService(overviewReader, providerSelection)
	if err != nil {
		applicationLogger.Error("initialize market signal service", "error", err)
		return err
	}
	rankingService, err := market.NewRankingService(overviewReader, providerSelection)
	if err != nil {
		applicationLogger.Error("initialize market ranking service", "error", err)
		return err
	}
	stockReader, err := stockinfrastructure.NewMySQLOverviewReader(database)
	if err != nil {
		applicationLogger.Error("initialize stock overview reader", "error", err)
		return err
	}
	stockService, err := stock.NewOverviewService(stockReader)
	if err != nil {
		applicationLogger.Error("initialize stock overview service", "error", err)
		return err
	}
	financialsReader, err := stockinfrastructure.NewMySQLFinancialsReader(database, providerSelection.MetadataName)
	if err != nil {
		applicationLogger.Error("initialize stock financials reader", "error", err)
		return err
	}
	financialsService, err := stock.NewFinancialsService(financialsReader, stock.FinancialSource{Mode: string(providerSelection.Mode), Provider: providerSelection.Provider})
	if err != nil {
		applicationLogger.Error("initialize stock financials service", "error", err)
		return err
	}
	valuationReader, err := stockinfrastructure.NewMySQLValuationReader(database, providerSelection.MetadataName)
	if err != nil {
		applicationLogger.Error("initialize stock valuation reader", "error", err)
		return err
	}
	valuationService, err := stock.NewValuationService(valuationReader, stock.ValuationSource{Mode: string(providerSelection.Mode), Provider: providerSelection.Provider})
	if err != nil {
		applicationLogger.Error("initialize stock valuation service", "error", err)
		return err
	}
	screenerReader, err := screenerinfrastructure.NewMySQLReader(database, screener.Source{Mode: string(providerSelection.Mode), Provider: providerSelection.Provider}, providerSelection.MetadataName)
	if err != nil {
		applicationLogger.Error("initialize screener reader", "error", err)
		return err
	}
	screenerService, err := screener.NewService(screenerReader)
	if err != nil {
		applicationLogger.Error("initialize screener service", "error", err)
		return err
	}
	savedScreenerService, err := screener.NewSavedScreenerService(savedScreenerStore)
	if err != nil {
		applicationLogger.Error("initialize saved screener service", "error", err)
		return err
	}
	stockPoolService, err := pool.NewService(stockPoolStore, stockReader)
	if err != nil {
		applicationLogger.Error("initialize stock pool service", "error", err)
		return err
	}
	researchService, err := research.NewService(researchStore)
	if err != nil {
		applicationLogger.Error("initialize research service", "error", err)
		return err
	}
	barsService, err := market.NewBarsService(overviewReader, providerSelection)
	if err != nil {
		applicationLogger.Error("initialize stock bars service", "error", err)
		return err
	}
	server := newHTTPServerWithMarketSignalsAndRankingsAndStocksAndBarsAndFinancialsAndValuationAndScreenerAndStockPoolsAndResearch(applicationLogger, httpAddress, overviewService, sectorService, signalService, rankingService, stockService, barsService, financialsService, valuationService, screenerService, savedScreenerService, stockPoolService, researchService, statusReader)
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

func openDemoDatabase(ctx context.Context) (*sql.DB, error) {
	database, err := infrastructure.Open(ctx, config.LoadDatabase())
	if err != nil {
		return nil, err
	}
	return database, nil
}

func syncTushare(ctx context.Context, database *sql.DB, settings config.Tushare, applicationLogger *slog.Logger) error {
	client, err := marketinfrastructure.NewTushareClient(settings)
	if err != nil {
		return fmt.Errorf("initialize Tushare client: %w", err)
	}
	syncer, err := marketinfrastructure.NewTushareSyncer(database, client)
	if err != nil {
		return fmt.Errorf("initialize Tushare syncer: %w", err)
	}
	summary, err := syncer.Sync(ctx, marketinfrastructure.TushareSyncOptions{
		StartDate: settings.StartDate, EndDate: settings.EndDate, LookbackDays: settings.LookbackDays, MetadataName: market.TushareMetadataName,
	})
	if err != nil {
		return fmt.Errorf("run Tushare sync: %w", err)
	}
	applicationLogger.Info("Tushare data synchronized", "provider", summary.Provider, "as_of", summary.AsOf, "daily_bars", summary.DailyBars, "daily_basics", summary.DailyBasics, "factors", summary.Factors, "indexes", summary.Indexes)
	return nil
}

func newStatusReader(environment string, store demo.StatusStore, requestedProvider string) (demo.StatusReader, error) {
	if environment != "development" {
		return nil, nil
	}
	if market.IsRealProviderRequested(requestedProvider) {
		return nil, nil
	}
	return demo.NewService(store, requestedProvider)
}
