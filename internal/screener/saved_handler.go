package screener

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	savedScreenersPath  = "/screeners"
	savedScreenerIDPath = "/screeners/:id"
)

type savedScreenerWriteRequest struct {
	Name               string              `json:"name"`
	Description        *string             `json:"description"`
	Spec               domain.ScreenerSpec `json:"spec"`
	Version            int64               `json:"version"`
	descriptionPresent bool
}

func (request *savedScreenerWriteRequest) UnmarshalJSON(raw []byte) error {
	type requestAlias savedScreenerWriteRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var decoded requestAlias
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("请求体只能包含一个 JSON 对象")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	decoded.descriptionPresent = false
	if _, ok := fields["description"]; ok {
		decoded.descriptionPresent = true
	}
	*request = savedScreenerWriteRequest(decoded)
	return nil
}

// RegisterSavedRoutes 注册选股方案保存、列表、读取和更新 API。
func RegisterSavedRoutes(router *gin.RouterGroup, query SavedScreenerQuery) {
	if query == nil {
		return
	}
	// 预留 run 静态路径，避免方案的 :id 路由把 GET/PUT /screeners/run 当成非法 ID。
	methodNotAllowed := func(context *gin.Context) {
		context.AbortWithStatusJSON(http.StatusMethodNotAllowed, api.NewErrorResponse(api.CodeMethodNotAllowed, "请求方法不被允许", nil))
	}
	router.GET(runPath, methodNotAllowed)
	router.PUT(runPath, methodNotAllowed)
	router.POST(savedScreenersPath, func(context *gin.Context) {
		request, err := decodeSavedScreenerWriteRequest(context, false)
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		result, err := query.Create(context.Request.Context(), request.input())
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
	router.GET(savedScreenersPath, func(context *gin.Context) {
		pagination, err := api.ParsePagination(context.Request.URL.Query())
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		result, err := query.List(context.Request.Context(), SavedScreenerListRequest{Page: pagination.Page, PageSize: pagination.PageSize})
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
	router.GET(savedScreenerIDPath, func(context *gin.Context) {
		id, err := parseSavedScreenerID(context.Param("id"))
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		result, err := query.Get(context.Request.Context(), id)
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
	router.PUT(savedScreenerIDPath, func(context *gin.Context) {
		id, err := parseSavedScreenerID(context.Param("id"))
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		request, err := decodeSavedScreenerWriteRequest(context, true)
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		result, err := query.Update(context.Request.Context(), id, request.Version, request.input())
		if err != nil {
			writeSavedScreenerError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	})
}

func (request savedScreenerWriteRequest) input() domain.SavedScreenerInput {
	return domain.SavedScreenerInput{Name: request.Name, Description: request.Description, Spec: request.Spec}
}

func decodeSavedScreenerWriteRequest(context *gin.Context, update bool) (savedScreenerWriteRequest, error) {
	decoder := json.NewDecoder(context.Request.Body)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var request savedScreenerWriteRequest
	if err := decoder.Decode(&request); err != nil {
		return savedScreenerWriteRequest{}, &domain.ValidationError{Fields: map[string]string{"body": "请求体必须是合法 JSON 且只包含已发布字段"}}
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return savedScreenerWriteRequest{}, &domain.ValidationError{Fields: map[string]string{"body": "请求体只能包含一个 JSON 对象"}}
	}
	if update {
		fields := make(map[string]string)
		if request.Version < 1 {
			fields["version"] = "必须提供当前版本号"
		}
		if !request.descriptionPresent {
			fields["description"] = "更新时必须明确提供字符串或 null"
		}
		if len(fields) > 0 {
			return savedScreenerWriteRequest{}, &domain.ValidationError{Fields: fields}
		}
	}
	return request, nil
}

func parseSavedScreenerID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id < 1 {
		return 0, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	return id, nil
}

func writeSavedScreenerError(context *gin.Context, err error) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "保存方案参数无效", validationError.Details()))
		return
	}
	var paginationError *api.PaginationValidationError
	if errors.As(err, &paginationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeInvalidPagination, "分页参数无效", paginationError.Details()))
		return
	}
	var conflictError *domain.SavedScreenerVersionConflictError
	if errors.As(err, &conflictError) {
		context.AbortWithStatusJSON(http.StatusConflict, api.NewErrorResponse(api.CodeConflict, "保存方案版本已过期", conflictError.Details()))
		return
	}
	if errors.Is(err, domain.ErrSavedScreenerNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "保存方案不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "保存方案数据暂不可用", nil))
}

var _ interface {
	Create(context.Context, domain.SavedScreenerInput) (domain.SavedScreener, error)
} = (*SavedScreenerService)(nil)
