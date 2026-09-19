// Package pool 编排股票池用例并提供 HTTP 所需的最小接口。
package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
)

// StockPoolStore 是股票池的持久化边界。
type StockPoolStore interface {
	Create(context.Context, domain.StockPool) (domain.StockPool, error)
	List(context.Context, string, int, int) ([]domain.StockPool, int64, error)
	Get(context.Context, int64) (domain.StockPool, error)
	Summary(context.Context, int64) (domain.StockPoolSummary, error)
	ListMembers(context.Context, int64, int, int) ([]domain.StockPoolMember, int64, error)
	AddMember(context.Context, int64, string) (domain.StockPoolMember, int64, error)
	DeleteMember(context.Context, int64, string) (int64, error)
}

// StockIdentityReader 是复用 STK-001 股票身份表的最小查询边界。
type StockIdentityReader interface {
	Exists(context.Context, string) (bool, error)
}

// StockPoolQuery 是股票池 HTTP 适配器需要的最小应用接口。
type StockPoolQuery interface {
	Create(context.Context, domain.StockPoolInput) (domain.StockPool, error)
	List(context.Context, StockPoolListRequest) (StockPoolListResponse, error)
	Get(context.Context, int64) (domain.StockPool, error)
	Summary(context.Context, int64) (domain.StockPoolSummary, error)
	ListMembers(context.Context, int64, StockPoolMemberListRequest) (StockPoolMemberListResponse, error)
	AddMember(context.Context, int64, string) (StockPoolMemberAddResponse, error)
	DeleteMember(context.Context, int64, string) (StockPoolMemberDeleteResponse, error)
}

// StockPoolListRequest 是股票池列表的搜索与分页请求。
type StockPoolListRequest struct {
	Search   string
	Page     int
	PageSize int
}

// StockPoolListResponse 是股票池列表响应。
type StockPoolListResponse struct {
	Data       []domain.StockPool `json:"data"`
	Pagination api.PaginationMeta `json:"pagination"`
}

// StockPoolMemberListRequest 是股票池成员列表的分页请求。
type StockPoolMemberListRequest struct {
	Page     int
	PageSize int
}

// StockPoolMemberListResponse 是股票池成员列表响应。
type StockPoolMemberListResponse struct {
	Data       []domain.StockPoolMember `json:"data"`
	Pagination api.PaginationMeta       `json:"pagination"`
}

// StockPoolMemberAddResponse 是添加成员后的真实结果。
type StockPoolMemberAddResponse struct {
	Member      domain.StockPoolMember `json:"member"`
	MemberCount int64                  `json:"member_count"`
}

// StockPoolMemberDeleteResponse 是删除成员后的真实结果。
type StockPoolMemberDeleteResponse struct {
	Symbol      string `json:"symbol"`
	MemberCount int64  `json:"member_count"`
}

// Service 编排股票池的领域校验与持久化调用。
type Service struct {
	store           StockPoolStore
	stockIdentities StockIdentityReader
}

// NewService 创建股票池应用服务。
func NewService(store StockPoolStore, stockIdentities StockIdentityReader) (*Service, error) {
	if store == nil {
		return nil, errors.New("stock pool store is required")
	}
	if stockIdentities == nil {
		return nil, errors.New("stock identity reader is required")
	}
	return &Service{store: store, stockIdentities: stockIdentities}, nil
}

// Create 创建来源固定为 manual 的股票池。
func (service *Service) Create(ctx context.Context, input domain.StockPoolInput) (domain.StockPool, error) {
	pool, err := domain.NewManualStockPool(input)
	if err != nil {
		return domain.StockPool{}, err
	}
	created, err := service.store.Create(ctx, pool)
	if err != nil {
		return domain.StockPool{}, fmt.Errorf("create stock pool: %w", err)
	}
	return created, nil
}

// List 按名称搜索并以稳定顺序分页读取股票池。
func (service *Service) List(ctx context.Context, request StockPoolListRequest) (StockPoolListResponse, error) {
	normalized, err := normalizeListRequest(request)
	if err != nil {
		return StockPoolListResponse{}, err
	}
	items, total, err := service.store.List(ctx, normalized.Search, normalized.Page, normalized.PageSize)
	if err != nil {
		return StockPoolListResponse{}, fmt.Errorf("list stock pools: %w", err)
	}
	return newListResponse(items, total, normalized.Page, normalized.PageSize), nil
}

// Get 读取一个股票池概览。
func (service *Service) Get(ctx context.Context, id int64) (domain.StockPool, error) {
	if id < 1 {
		return domain.StockPool{}, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	pool, err := service.store.Get(ctx, id)
	if err != nil {
		return domain.StockPool{}, fmt.Errorf("get stock pool: %w", err)
	}
	return pool, nil
}

// Summary 读取 Pool 元数据、来源和基础画像的一致性快照。
func (service *Service) Summary(ctx context.Context, id int64) (domain.StockPoolSummary, error) {
	if id < 1 {
		return domain.StockPoolSummary{}, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	summary, err := service.store.Summary(ctx, id)
	if err != nil {
		return domain.StockPoolSummary{}, fmt.Errorf("get stock pool summary: %w", err)
	}
	return summary, nil
}

// ListMembers 读取股票池中的真实成员并按 symbol 稳定分页。
func (service *Service) ListMembers(ctx context.Context, id int64, request StockPoolMemberListRequest) (StockPoolMemberListResponse, error) {
	if id < 1 {
		return StockPoolMemberListResponse{}, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	page, pageSize, err := normalizePagination(request.Page, request.PageSize)
	if err != nil {
		return StockPoolMemberListResponse{}, err
	}
	items, total, err := service.store.ListMembers(ctx, id, page, pageSize)
	if err != nil {
		return StockPoolMemberListResponse{}, fmt.Errorf("list stock pool members: %w", err)
	}
	return StockPoolMemberListResponse{
		Data: items,
		Pagination: api.PaginationMeta{
			Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages(total, int64(pageSize)),
		},
	}, nil
}

// AddMember 验证 Pool 和股票身份后添加一个成员。
func (service *Service) AddMember(ctx context.Context, id int64, symbol string) (StockPoolMemberAddResponse, error) {
	if id < 1 {
		return StockPoolMemberAddResponse{}, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	member, err := domain.NewStockPoolMember(symbol)
	if err != nil {
		return StockPoolMemberAddResponse{}, err
	}
	if _, err := service.store.Get(ctx, id); err != nil {
		return StockPoolMemberAddResponse{}, fmt.Errorf("get stock pool before adding member: %w", err)
	}
	exists, err := service.stockIdentities.Exists(ctx, member.Symbol)
	if err != nil {
		return StockPoolMemberAddResponse{}, fmt.Errorf("check stock identity: %w", err)
	}
	if !exists {
		return StockPoolMemberAddResponse{}, domain.ErrStockPoolInstrumentNotFound
	}
	created, count, err := service.store.AddMember(ctx, id, member.Symbol)
	if err != nil {
		return StockPoolMemberAddResponse{}, fmt.Errorf("add stock pool member: %w", err)
	}
	return StockPoolMemberAddResponse{Member: created, MemberCount: count}, nil
}

// DeleteMember 删除一个真实存在的股票池成员。
func (service *Service) DeleteMember(ctx context.Context, id int64, symbol string) (StockPoolMemberDeleteResponse, error) {
	if id < 1 {
		return StockPoolMemberDeleteResponse{}, &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	member, err := domain.NewStockPoolMember(symbol)
	if err != nil {
		return StockPoolMemberDeleteResponse{}, err
	}
	count, err := service.store.DeleteMember(ctx, id, member.Symbol)
	if err != nil {
		return StockPoolMemberDeleteResponse{}, fmt.Errorf("delete stock pool member: %w", err)
	}
	return StockPoolMemberDeleteResponse{Symbol: member.Symbol, MemberCount: count}, nil
}

func normalizeListRequest(request StockPoolListRequest) (StockPoolListRequest, error) {
	normalized := request
	normalized.Search = strings.TrimSpace(request.Search)
	if len([]rune(normalized.Search)) > domain.MaxStockPoolSearchLength {
		return StockPoolListRequest{}, &domain.ValidationError{Fields: map[string]string{"q": fmt.Sprintf("长度不能超过 %d 个字符", domain.MaxStockPoolSearchLength)}}
	}
	page, pageSize, err := normalizePagination(request.Page, request.PageSize)
	if err != nil {
		return StockPoolListRequest{}, err
	}
	normalized.Page, normalized.PageSize = page, pageSize
	return normalized, nil
}

func normalizePagination(page, pageSize int) (int, int, error) {
	if page == 0 {
		page = api.DefaultPage
	}
	if pageSize == 0 {
		pageSize = api.DefaultPageSize
	}
	fields := make(map[string]string)
	if page < api.DefaultPage {
		fields["page"] = "必须是大于等于 1 的整数"
	}
	if pageSize < 1 || pageSize > api.MaxPageSize {
		fields["page_size"] = fmt.Sprintf("必须是 1 到 %d 之间的整数", api.MaxPageSize)
	}
	if len(fields) > 0 {
		return 0, 0, &api.PaginationValidationError{Fields: fields}
	}
	return page, pageSize, nil
}

func newListResponse(items []domain.StockPool, total int64, page, pageSize int) StockPoolListResponse {
	return StockPoolListResponse{
		Data: items,
		Pagination: api.PaginationMeta{
			Page: page, PageSize: pageSize, Total: total,
			TotalPages: totalPages(total, int64(pageSize)),
		},
	}
}

func totalPages(total, pageSize int64) int64 {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

var _ StockPoolQuery = (*Service)(nil)
