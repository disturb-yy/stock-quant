package screener

import (
	"context"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/screener/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
)

// SavedScreenerStore 是保存方案的持久化边界。
type SavedScreenerStore interface {
	Create(context.Context, domain.SavedScreenerInput) (domain.SavedScreener, error)
	List(context.Context, int, int) ([]domain.SavedScreener, int64, error)
	Get(context.Context, int64) (domain.SavedScreener, error)
	Update(context.Context, int64, int64, domain.SavedScreenerInput) (domain.SavedScreener, error)
}

// SavedScreenerQuery 是保存方案 HTTP 适配器需要的最小应用接口。
type SavedScreenerQuery interface {
	Create(context.Context, domain.SavedScreenerInput) (domain.SavedScreener, error)
	List(context.Context, SavedScreenerListRequest) (SavedScreenerListResponse, error)
	Get(context.Context, int64) (domain.SavedScreener, error)
	Update(context.Context, int64, int64, domain.SavedScreenerInput) (domain.SavedScreener, error)
}

// SavedScreenerListRequest 是方案列表分页请求。
type SavedScreenerListRequest struct {
	Page     int
	PageSize int
}

// SavedScreenerListResponse 是方案列表响应。
type SavedScreenerListResponse struct {
	Data       []domain.SavedScreener `json:"data"`
	Pagination api.PaginationMeta     `json:"pagination"`
}

// SavedScreenerService 编排方案字段规范化和持久化。
type SavedScreenerService struct {
	store SavedScreenerStore
}

func NewSavedScreenerService(store SavedScreenerStore) (*SavedScreenerService, error) {
	if store == nil {
		return nil, errors.New("saved screener store is required")
	}
	return &SavedScreenerService{store: store}, nil
}

func (service *SavedScreenerService) Create(ctx context.Context, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	normalized, err := domain.NormalizeSavedScreenerInput(input)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	saved, err := service.store.Create(ctx, normalized)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("create saved screener: %w", err)
	}
	return saved, nil
}

func (service *SavedScreenerService) List(ctx context.Context, request SavedScreenerListRequest) (SavedScreenerListResponse, error) {
	page, pageSize, err := normalizeSavedScreenerPagination(request)
	if err != nil {
		return SavedScreenerListResponse{}, err
	}
	items, total, err := service.store.List(ctx, page, pageSize)
	if err != nil {
		return SavedScreenerListResponse{}, fmt.Errorf("list saved screeners: %w", err)
	}
	return SavedScreenerListResponse{
		Data: items,
		Pagination: api.PaginationMeta{
			Page: page, PageSize: pageSize, Total: total,
			TotalPages: savedScreenerTotalPages(total, int64(pageSize)),
		},
	}, nil
}

func (service *SavedScreenerService) Get(ctx context.Context, id int64) (domain.SavedScreener, error) {
	if err := validateSavedScreenerID(id); err != nil {
		return domain.SavedScreener{}, err
	}
	saved, err := service.store.Get(ctx, id)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("get saved screener: %w", err)
	}
	return saved, nil
}

func (service *SavedScreenerService) Update(ctx context.Context, id, version int64, input domain.SavedScreenerInput) (domain.SavedScreener, error) {
	if err := validateSavedScreenerID(id); err != nil {
		return domain.SavedScreener{}, err
	}
	if version < 1 {
		return domain.SavedScreener{}, &domain.ValidationError{Fields: map[string]string{"version": "必须是大于等于 1 的整数"}}
	}
	normalized, err := domain.NormalizeSavedScreenerInput(input)
	if err != nil {
		return domain.SavedScreener{}, err
	}
	saved, err := service.store.Update(ctx, id, version, normalized)
	if err != nil {
		return domain.SavedScreener{}, fmt.Errorf("update saved screener: %w", err)
	}
	return saved, nil
}

func normalizeSavedScreenerPagination(request SavedScreenerListRequest) (int, int, error) {
	page, pageSize := request.Page, request.PageSize
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

func validateSavedScreenerID(id int64) error {
	if id < 1 {
		return &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	return nil
}

func savedScreenerTotalPages(total, pageSize int64) int64 {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

var _ SavedScreenerQuery = (*SavedScreenerService)(nil)
