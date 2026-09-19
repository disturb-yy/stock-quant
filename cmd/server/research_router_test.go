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

	"github.com/disturb-yy/stock-quant/internal/research"
	"github.com/disturb-yy/stock-quant/internal/research/domain"
	"github.com/gin-gonic/gin"
)

type fakeResearchReader struct{}

func (fakeResearchReader) Create(context.Context, domain.ResearchProjectInput) (domain.ResearchProject, error) {
	return testResearchProject(), nil
}

func (fakeResearchReader) List(context.Context, research.ResearchListRequest) (research.ResearchListResponse, error) {
	return research.ResearchListResponse{Data: []domain.ResearchProject{testResearchProject()}}, nil
}

func (fakeResearchReader) Get(context.Context, int64) (domain.ResearchProject, error) {
	return testResearchProject(), nil
}

func TestServerRouterRegistersResearchRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	applicationLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouterWithMarketSignalsAndRankingsAndStocksAndBarsAndFinancialsAndValuationAndScreenerAndStockPoolsAndResearch(applicationLogger, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, fakeResearchReader{})
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Post(server.URL+"/api/v1/research", "application/json", strings.NewReader(`{"name":"路由测试研究"}`))
	if err != nil {
		t.Fatalf("POST research: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("POST research status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	for _, path := range []string{"/api/v1/research", "/api/v1/research/1", "/api/v1/openapi.json"} {
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

func testResearchProject() domain.ResearchProject {
	return domain.ResearchProject{ID: 1, Name: "路由测试研究", CreatedAt: time.Unix(0, 0).UTC(), UpdatedAt: time.Unix(0, 0).UTC()}
}
