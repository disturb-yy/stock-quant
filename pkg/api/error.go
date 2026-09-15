package api

// ErrorCode 是客户端可依赖的机器可读错误码。
type ErrorCode string

const (
	CodeNotFound              ErrorCode = "NOT_FOUND"
	CodeMethodNotAllowed      ErrorCode = "METHOD_NOT_ALLOWED"
	CodeValidation            ErrorCode = "VALIDATION_ERROR"
	CodeInvalidPagination     ErrorCode = "INVALID_PAGINATION"
	CodeInsufficientHistory   ErrorCode = "INSUFFICIENT_HISTORY"
	CodeInternal              ErrorCode = "INTERNAL_ERROR"
	CodeDependencyUnavailable ErrorCode = "DEPENDENCY_UNAVAILABLE"
)

// ErrorResponse 是 API 错误响应的统一 DTO。
// Details 只承载可安全暴露的结构化校验信息，不应放入内部错误文本。
type ErrorResponse struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// NewErrorResponse 创建一个统一错误响应。
func NewErrorResponse(code ErrorCode, message string, details map[string]any) ErrorResponse {
	return ErrorResponse{Code: code, Message: message, Details: details}
}
