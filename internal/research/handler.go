package research

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/research/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
	"github.com/gin-gonic/gin"
)

const (
	researchPath     = "/research"
	researchByIDPath = "/research/:id"
)

type createResearchRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// RegisterRoutes 注册 Research 项目创建、列表和详情 API。
func RegisterRoutes(router *gin.RouterGroup, query ResearchQuery) {
	if query == nil {
		return
	}
	router.POST(researchPath, createResearchHandler(query))
	router.GET(researchPath, listResearchHandler(query))
	router.GET(researchByIDPath, getResearchHandler(query))
}

func createResearchHandler(query ResearchQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		request, err := decodeCreateResearchRequest(context)
		if err != nil {
			writeResearchError(context, err)
			return
		}
		result, err := query.Create(context.Request.Context(), domain.ResearchProjectInput{
			Name: request.Name, Description: request.Description,
		})
		if err != nil {
			writeResearchError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func listResearchHandler(query ResearchQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		pagination, err := api.ParsePagination(context.Request.URL.Query())
		if err != nil {
			writeResearchError(context, err)
			return
		}
		result, err := query.List(context.Request.Context(), ResearchListRequest{Page: pagination.Page, PageSize: pagination.PageSize})
		if err != nil {
			writeResearchError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func getResearchHandler(query ResearchQuery) gin.HandlerFunc {
	return func(context *gin.Context) {
		id, err := parseResearchID(context.Param("id"))
		if err != nil {
			writeResearchError(context, err)
			return
		}
		result, err := query.Get(context.Request.Context(), id)
		if err != nil {
			writeResearchError(context, err)
			return
		}
		context.JSON(http.StatusOK, result)
	}
}

func decodeCreateResearchRequest(context *gin.Context) (createResearchRequest, error) {
	decoder := json.NewDecoder(context.Request.Body)
	decoder.DisallowUnknownFields()
	var request createResearchRequest
	if err := decoder.Decode(&request); err != nil {
		return createResearchRequest{}, invalidResearchBodyError()
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return createResearchRequest{}, invalidResearchBodyError()
	}
	return request, nil
}

func invalidResearchBodyError() error {
	return &domain.ValidationError{Fields: map[string]string{"body": "请求体必须是合法 JSON 且只包含已发布字段"}}
}

func parseResearchID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id < 1 {
		return 0, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	return id, nil
}

func writeResearchError(context *gin.Context, err error) {
	var validationError *domain.ValidationError
	if errors.As(err, &validationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeValidation, "Research 项目参数无效", validationError.Details()))
		return
	}
	var paginationError *api.PaginationValidationError
	if errors.As(err, &paginationError) {
		context.AbortWithStatusJSON(http.StatusBadRequest, api.NewErrorResponse(api.CodeInvalidPagination, "分页参数无效", paginationError.Details()))
		return
	}
	if errors.Is(err, domain.ErrResearchProjectNotFound) {
		context.AbortWithStatusJSON(http.StatusNotFound, api.NewErrorResponse(api.CodeNotFound, "Research 项目不存在", nil))
		return
	}
	context.AbortWithStatusJSON(http.StatusServiceUnavailable, api.NewErrorResponse(api.CodeDependencyUnavailable, "Research 数据暂不可用", nil))
}

var _ interface {
	Create(context.Context, domain.ResearchProjectInput) (domain.ResearchProject, error)
} = (*Service)(nil)
