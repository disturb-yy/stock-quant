package stock

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const financialsPath = "/stocks/:symbol/financials"

// RegisterFinancialsRoutes 注册股票财务摘要、趋势和简化报表 HTTP 路由。
func RegisterFinancialsRoutes(router *gin.RouterGroup, query interface {
	Financials(context.Context, FinancialsRequest) (StockFinancials, error)
}) {
	if query == nil {
		return
	}
	router.GET(financialsPath, func(context *gin.Context) {
		request, err := financialsRequestFromQuery(context)
		if err != nil {
			writeFinancialsError(context, err)
			return
		}
		request.Symbol = context.Param("symbol")
		request, err = normalizeFinancialsRequest(request)
		if err != nil {
			writeFinancialsError(context, err)
			return
		}
		result, err := query.Financials(context.Request.Context(), request)
		if err != nil {
			writeFinancialsError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

func financialsRequestFromQuery(context *gin.Context) (FinancialsRequest, error) {
	values := context.Request.URL.Query()
	request := FinancialsRequest{}
	for _, field := range []struct {
		name   string
		target *string
	}{
		{name: "period", target: (*string)(&request.Period)},
		{name: "range", target: (*string)(&request.Range)},
	} {
		queryValues, ok := values[field.name]
		if !ok {
			continue
		}
		if len(queryValues) != 1 || strings.TrimSpace(queryValues[0]) == "" {
			return FinancialsRequest{}, &FinancialsValidationError{Fields: map[string]string{field.name: "必须提供且只能提供一次"}}
		}
		*field.target = queryValues[0]
	}
	return request, nil
}

func writeFinancialsError(context *gin.Context, err error) {
	var validationError *FinancialsValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "股票财务参数无效", validationError.Details()))
		return
	}
	if errors.Is(err, ErrFinancialsInstrumentNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票财务数据暂不可用", nil))
}
