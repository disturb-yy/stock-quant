package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/pool"
	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	"github.com/gin-gonic/gin"
)

type fakeStockPoolReader struct{}

func (fakeStockPoolReader) Create(context.Context, domain.StockPoolInput) (domain.StockPool, error) {
	return routerStockPool(), nil
}

func (fakeStockPoolReader) List(context.Context, pool.StockPoolListRequest) (pool.StockPoolListResponse, error) {
	return pool.StockPoolListResponse{Data: []domain.StockPool{routerStockPool()}}, nil
}

func (fakeStockPoolReader) Get(context.Context, int64) (domain.StockPool, error) {
	return routerStockPool(), nil
}

func TestServerRouterRegistersStockPoolRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocksAndBarsAndFinancialsAndValuationAndScreenerAndStockPools(applicationLogger, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fakeStockPoolReader{})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Post(server.URL+"/api/v1/stock-pools", "application/json", strings.NewReader(`{"name":"路由测试池"}`))
	if err != nil {
		t.Fatalf("POST stock pool: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read POST body: %v", err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"source":"manual"`) {
		t.Fatalf("POST status/body = %d/%s", response.StatusCode, body)
	}

	for _, path := range []string{"/api/v1/stock-pools?q=%E6%B5%8B%E8%AF%95", "/api/v1/stock-pools/1", "/api/v1/openapi.json"} {
		response, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s status = %d, want %d", path, response.StatusCode, http.StatusOK)
		}
	}
}

func routerStockPool() domain.StockPool {
	return domain.StockPool{ID: 1, Name: "路由测试池", Source: domain.SourceManual, MemberCount: 0, CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()}
}
