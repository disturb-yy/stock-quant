package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/disturb-yy/stock-quant/internal/data/application"
	"github.com/disturb-yy/stock-quant/internal/data/domain"
	"github.com/gin-gonic/gin"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
)

type Service interface {
	CreateTask(context.Context, application.CreateTaskInput) (domain.SyncTask, error)
	GetTask(context.Context, string) (domain.SyncTask, error)
	ListTasks(context.Context, int, int) (domain.TaskPage, error)
	RetryFailedTask(context.Context, string) (domain.SyncTask, error)
}

type Handler struct {
	service Service
}

func RegisterRoutes(router *gin.RouterGroup, service Service) {
	if router == nil || service == nil {
		return
	}
	handler := &Handler{service: service}
	routes := router.Group("/stock/data/sync-tasks")
	routes.POST("", handler.create)
	routes.GET("", handler.list)
	routes.GET("/:task_id", handler.detail)
	routes.POST("/:task_id/retry", handler.retry)
}

type createTaskRequest struct {
	Target    string  `json:"target"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
}

type taskResponse struct {
	TaskID        string          `json:"task_id"`
	Target        string          `json:"target"`
	Trigger       string          `json:"trigger"`
	Status        string          `json:"status"`
	Source        sourceResponse  `json:"source"`
	StartDate     *string         `json:"start_date"`
	EndDate       *string         `json:"end_date"`
	DataAsOf      *string         `json:"data_as_of"`
	UpdatedAt     string          `json:"updated_at"`
	CreatedAt     string          `json:"created_at"`
	StartedAt     *string         `json:"started_at"`
	FinishedAt    *string         `json:"finished_at"`
	RetryCount    int             `json:"retry_count"`
	MaxRetries    int             `json:"max_retries"`
	FailureReason *string         `json:"failure_reason"`
	Result        *resultResponse `json:"result"`
	RetryOfTaskID *string         `json:"retry_of_task_id"`
}

type sourceResponse struct {
	Provider string `json:"provider"`
	Mode     string `json:"mode"`
}

type resultResponse struct {
	ProcessedCount int `json:"processed_count"`
	CreatedCount   int `json:"created_count"`
	UpdatedCount   int `json:"updated_count"`
	FailedCount    int `json:"failed_count"`
}

type listResponse struct {
	Items      []taskResponse     `json:"items"`
	Pagination paginationResponse `json:"pagination"`
}

type paginationResponse struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (handler *Handler) create(ctx *gin.Context) {
	var request createTaskRequest
	// DTO 只允许已确认字段，避免请求覆盖运行环境决定的数据源和重试策略。
	if err := decodeJSON(ctx, &request); err != nil {
		writeError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "请求参数无效", nil)
		return
	}
	input, err := buildCreateInput(request)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	task, err := handler.service.CreateTask(ctx.Request.Context(), input)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, toTaskResponse(task))
}

func (handler *Handler) list(ctx *gin.Context) {
	page, pageSize, err := parsePagination(ctx)
	if err != nil {
		writeError(ctx, http.StatusBadRequest, "INVALID_PAGINATION", "分页参数无效", nil)
		return
	}
	result, err := handler.service.ListTasks(ctx.Request.Context(), page, pageSize)
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	items := make([]taskResponse, 0, len(result.Items))
	for _, task := range result.Items {
		items = append(items, toTaskResponse(task))
	}
	ctx.JSON(http.StatusOK, listResponse{Items: items, Pagination: paginationResponse{
		Page: result.Page, PageSize: result.PageSize, Total: result.Total,
	}})
}

func (handler *Handler) detail(ctx *gin.Context) {
	task, err := handler.service.GetTask(ctx.Request.Context(), ctx.Param("task_id"))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, toTaskResponse(task))
}

func (handler *Handler) retry(ctx *gin.Context) {
	task, err := handler.service.RetryFailedTask(ctx.Request.Context(), ctx.Param("task_id"))
	if err != nil {
		writeServiceError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, toTaskResponse(task))
}

func decodeJSON(ctx *gin.Context, target any) error {
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request contains more than one JSON value")
	}
	return nil
}

func buildCreateInput(request createTaskRequest) (application.CreateTaskInput, error) {
	target := domain.SyncTarget(request.Target)
	if !target.Valid() {
		return application.CreateTaskInput{}, domain.ErrInvalidTarget
	}
	if !target.IncludesDailyBars() {
		if request.StartDate != nil || request.EndDate != nil {
			return application.CreateTaskInput{}, domain.ErrInvalidDateRange
		}
		return application.CreateTaskInput{Target: target, Trigger: domain.TriggerManual}, nil
	}
	if request.StartDate == nil || request.EndDate == nil {
		return application.CreateTaskInput{}, domain.ErrInvalidDateRange
	}
	dateRange, err := domain.ParseDateRange(*request.StartDate, *request.EndDate)
	if err != nil {
		return application.CreateTaskInput{}, err
	}
	return application.CreateTaskInput{Target: target, DateRange: dateRange, Trigger: domain.TriggerManual}, nil
}

func parsePagination(ctx *gin.Context) (int, int, error) {
	page, err := queryInt(ctx, "page", defaultPage)
	if err != nil || page < 1 {
		return 0, 0, domain.ErrInvalidPagination
	}
	pageSize, err := queryInt(ctx, "page_size", defaultPageSize)
	if err != nil || pageSize < 1 || pageSize > domain.MaxPageSize {
		return 0, 0, domain.ErrInvalidPagination
	}
	return page, pageSize, nil
}

func queryInt(ctx *gin.Context, name string, fallback int) (int, error) {
	value, exists := ctx.GetQuery(name)
	if !exists {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func toTaskResponse(task domain.SyncTask) taskResponse {
	response := taskResponse{
		TaskID: task.ID, Target: string(task.Target), Trigger: string(task.Trigger), Status: string(task.Status),
		Source:    sourceResponse{Provider: task.Source.Provider, Mode: task.Source.Mode},
		StartDate: dateString(task.DateRange.Start), EndDate: dateString(task.DateRange.End),
		DataAsOf: dateString(task.DataAsOf), UpdatedAt: task.UpdatedAt.UTC().Format(time.RFC3339),
		CreatedAt: task.CreatedAt.UTC().Format(time.RFC3339), StartedAt: timeString(task.StartedAt),
		FinishedAt: timeString(task.FinishedAt), RetryCount: task.RetryCount, MaxRetries: task.MaxRetries,
		FailureReason: optionalString(task.FailureReason), Result: resultResponseFor(task.Result),
		RetryOfTaskID: task.RetryOfTaskID,
	}
	return response
}

func resultResponseFor(result *domain.SyncResult) *resultResponse {
	if result == nil {
		return nil
	}
	return &resultResponse{ProcessedCount: result.ProcessedCount, CreatedCount: result.CreatedCount,
		UpdatedCount: result.UpdatedCount, FailedCount: result.FailedCount}
}

func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(domain.DateLayout)
	return &formatted
}

func timeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func writeServiceError(ctx *gin.Context, err error) {
	descriptor := describeError(err)
	writeError(ctx, descriptor.status, descriptor.code, descriptor.message, nil)
}

type errorDescriptor struct {
	status  int
	code    string
	message string
}

func describeError(err error) errorDescriptor {
	switch {
	case errors.Is(err, domain.ErrInvalidPagination):
		return errorDescriptor{http.StatusBadRequest, "INVALID_PAGINATION", "分页参数无效"}
	case errors.Is(err, domain.ErrInvalidTarget), errors.Is(err, domain.ErrInvalidDateRange),
		errors.Is(err, domain.ErrInvalidTask), errors.Is(err, domain.ErrInvalidStockSymbol):
		return errorDescriptor{http.StatusBadRequest, "INVALID_REQUEST", "请求参数无效"}
	case errors.Is(err, domain.ErrStockNotFound):
		return errorDescriptor{http.StatusNotFound, "STOCK_NOT_FOUND", "股票标识不存在"}
	case errors.Is(err, domain.ErrTaskNotFound):
		return errorDescriptor{http.StatusNotFound, "SYNC_TASK_NOT_FOUND", "同步任务不存在"}
	case errors.Is(err, domain.ErrTaskConflict):
		return errorDescriptor{http.StatusConflict, "SYNC_TASK_CONFLICT", "同步任务已存在运行中的任务"}
	case errors.Is(err, domain.ErrTaskNotRetryable):
		return errorDescriptor{http.StatusConflict, "SYNC_TASK_NOT_RETRYABLE", "当前同步任务不可重试"}
	case errors.Is(err, domain.ErrDataSourceUnavailable):
		return errorDescriptor{http.StatusServiceUnavailable, "DATA_SOURCE_UNAVAILABLE", "数据源暂不可用"}
	default:
		return errorDescriptor{http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "内部服务错误"}
	}
}

func writeError(ctx *gin.Context, status int, code, message string, details any) {
	ctx.JSON(status, errorResponse{Code: code, Message: message, Details: details})
}
