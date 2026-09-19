package pool

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	stockPoolsPath              = "/stock-pools"
	stockPoolIDPath             = "/stock-pools/:id"
	stockPoolSummaryPath        = "/stock-pools/:id/summary"
	stockPoolMembersPath        = "/stock-pools/:id/members"
	stockPoolMemberBySymbolPath = "/stock-pools/:id/members/:symbol"
)

type stockPoolCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type stockPoolMemberAddRequest struct {
	Symbol string `json:"symbol"`
}

// RegisterRoutes 注册股票池及其成员 API。
func RegisterRoutes(router *gin.RouterGroup, query StockPoolQuery) {
	if query == nil {
		return
	}
	router.POST(stockPoolsPath, createStockPoolHandler(query))
	router.GET(stockPoolsPath, listStockPoolsHandler(query))
	router.GET(stockPoolIDPath, getStockPoolHandler(query))
	router.GET(stockPoolSummaryPath, getStockPoolSummaryHandler(query))
	router.GET(stockPoolMembersPath, listStockPoolMembersHandler(query))
	router.POST(stockPoolMembersPath, addStockPoolMemberHandler(query))
	router.DELETE(stockPoolMemberBySymbolPath, deleteStockPoolMemberHandler(query))
}

func createStockPoolHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		request, err := decodeStockPoolCreateRequest(context)
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.Create(context.Request.Context(), domain.StockPoolInput{Name: request.Name, Description: request.Description})
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func listStockPoolsHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		pagination, err := api.ParsePagination(context.Request.URL.Query())
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.List(context.Request.Context(), StockPoolListRequest{
			Search: context.Query("q"), Page: pagination.Page, PageSize: pagination.PageSize,
		})
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func getStockPoolHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseStockPoolID(context.Param("id"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.Get(context.Request.Context(), id)
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func getStockPoolSummaryHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseStockPoolID(context.Param("id"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.Summary(context.Request.Context(), id)
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func listStockPoolMembersHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseStockPoolID(context.Param("id"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		pagination, err := api.ParsePagination(context.Request.URL.Query())
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.ListMembers(context.Request.Context(), id, StockPoolMemberListRequest{
			Page: pagination.Page, PageSize: pagination.PageSize,
		})
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func addStockPoolMemberHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseStockPoolID(context.Param("id"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		request, err := decodeStockPoolMemberAddRequest(context)
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.AddMember(context.Request.Context(), id, request.Symbol)
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func deleteStockPoolMemberHandler(query StockPoolQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseStockPoolID(context.Param("id"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		result, err := query.DeleteMember(context.Request.Context(), id, context.Param("symbol"))
		if err != nil {
			writeStockPoolError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func decodeStockPoolCreateRequest(context *gin.Context) (stockPoolCreateRequest, error) {
	decoder := json.NewDecoder(context.Request.Body)
	decoder.DisallowUnknownFields()
	var request stockPoolCreateRequest
	if err := decoder.Decode(&request); err != nil {
		return stockPoolCreateRequest{}, invalidStockPoolBodyError()
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return stockPoolCreateRequest{}, invalidStockPoolBodyError()
	}
	return request, nil
}

func decodeStockPoolMemberAddRequest(context *gin.Context) (stockPoolMemberAddRequest, error) {
	decoder := json.NewDecoder(context.Request.Body)
	decoder.DisallowUnknownFields()
	var request stockPoolMemberAddRequest
	if err := decoder.Decode(&request); err != nil {
		return stockPoolMemberAddRequest{}, invalidStockPoolBodyError()
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return stockPoolMemberAddRequest{}, invalidStockPoolBodyError()
	}
	return request, nil
}

func invalidStockPoolBodyError() error {
	return &domain.ValidationError{Fields: map[string]string{"body": "请求体必须是合法 JSON 且只包含已发布字段"}}
}

func parseStockPoolID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id < 1 {
		return 0, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	return id, nil
}

func writeStockPoolError(context *gin.Context, err error) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "股票池参数无效", validationError.Details()))
		return
	}
	var paginationError *api.PaginationValidationError
	if errors.As(err, &paginationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeInvalidPagination, "分页参数无效", paginationError.Details()))
		return
	}
	if errors.Is(err, domain.ErrStockPoolNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票池不存在", nil))
		return
	}
	if errors.Is(err, domain.ErrStockPoolInstrumentNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票不存在", nil))
		return
	}
	if errors.Is(err, domain.ErrStockPoolMemberNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "股票池成员不存在", nil))
		return
	}
	if errors.Is(err, domain.ErrStockPoolMemberConflict) {
		context.AbortWithStatusJSON(http.StatusConflict, api.NewErrorResponse(api.CodeConflict, "股票已在股票池中", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "股票池数据暂不可用", nil))
}

var _ interface {
	Create(context.Context, domain.StockPoolInput) (domain.StockPool, error)
} = (*Service)(nil)
