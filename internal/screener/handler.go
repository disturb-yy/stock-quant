package screener

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const runPath = "/screeners/run"

// RegisterRoutes 注册临时选股执行 API。
func RegisterRoutes(router *gin.RouterGroup, query interface {
	Run(context.Context, ScreenerRunRequest) (ScreenerRunResponse, error)
}) {
	if query == nil {
		return
	}
	router.POST(runPath, func(context *gin.Context) {
		request, err := decodeRunRequest(context)
		if err != nil {
			writeRunError(context, err)
			return
		}
		response, err := query.Run(context.Request.Context(), request)
		if err != nil {
			writeRunError(context, err)
			return
		}
		context.JSON(http.StatusOK, response)
	})
}

func decodeRunRequest(context *gin.Context) (ScreenerRunRequest, error) {
	decoder := json.NewDecoder(context.Request.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var request ScreenerRunRequest
	if err := decoder.Decode(&request); err != nil {
		return ScreenerRunRequest{}, &domain.ValidationError{Fields: map[string]string{"body": "请求体必须是合法 JSON 且只包含已发布字段"}}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return ScreenerRunRequest{}, &domain.ValidationError{Fields: map[string]string{"body": "请求体只能包含一个 JSON 对象"}}
	}
	return request, nil
}

func writeRunError(context *gin.Context, err error) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "选股条件无效", validationError.Details()))
		return
	}
	if errors.Is(err, domain.ErrUniverseNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "选股 Universe 不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "选股数据暂不可用", nil))
}
