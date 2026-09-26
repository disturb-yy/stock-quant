package source

import (
	"fmt"
	"net/http"

	"github.com/disturb-yy/stock-quant/internal/data/config"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
)

func NewFromConfig(cfg config.Config, client *http.Client) (domain.StockDataSource, error) {
	switch cfg.Provider {
	case config.ProviderMock:
		return NewMockAdapter(), nil
	case config.ProviderTushare:
		return NewTushareAdapter(client, cfg.TushareEndpoint, cfg.TushareToken)
	default:
		return nil, fmt.Errorf("unsupported data source provider %q", cfg.Provider)
	}
}
