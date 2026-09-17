package stock

import (
	"context"
	"errors"
	"net/http"
	"strings"

	stockdomain "github.com/disturb-yy/stock-quant/internal/stock/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const valuationPath = "/stocks/:symbol/valuation"

// RegisterValuationRoutes 注册股票估值、历史分位和行业中位数查询路由。
func RegisterValuationRoutes(router *gin.RouterGroup, query interface {
	Valuation(context.Context, ValuationRequest) (StockValuation, error)
}) {
	if query == nil {
		return
	}
	router.GET(valuationPath, func(context *gin.Context) {
		request, err := valuationRequestFromQuery(context)
		if err != nil {
			writeValuationError(context, err)
			return
		}
		request.Symbol = context.Param("symbol")
		result, err := query.Valuation(context.Request.Context(), request)
		if err != nil {
			writeValuationError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

func valuationRequestFromQuery(context *gin.Context) (ValuationRequest, error) {
	values := context.Request.URL.Query()
	request := ValuationRequest{}
	queryValues, ok := values["range"]
	if !ok {
		return request, nil
	}
	if len(queryValues) != 1 || strings.TrimSpace(queryValues[0]) == "" {
		return ValuationRequest{}, &ValuationValidationError{Fields: map[string]string{"range": "必须提供且只能提供一次"}}
	}
	request.Range = stockdomain.ValuationRange(queryValues[0])
	return request, nil
}

func writeValuationError(context *gin.Context, err error) {
	var validationError *ValuationValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "股票估值参数无效", validationError.Details()))
		return
	}
	if errors.Is(err, ErrValuationInstrumentNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票估值数据暂不可用", nil))
}
