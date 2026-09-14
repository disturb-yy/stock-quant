package api

// Response 是统一成功响应的可复用 DTO。
// 现有 health 接口为保持兼容仍使用自己的非 envelope 响应。
type Response[T any] struct {
	Data T `json:"data"`
}

// PaginatedResponse 是列表接口可复用的分页成功响应 DTO。
type PaginatedResponse[T any] struct {
	Data       []T            `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}
