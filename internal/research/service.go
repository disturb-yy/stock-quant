package research

import (
	"context"
	"errors"
	"fmt"

	"github.com/disturb-yy/stock-quant/internal/research/domain"
	"github.com/disturb-yy/stock-quant/pkg/api"
)

// ResearchStore 是 Research 项目的持久化边界。
type ResearchStore interface {
	Create(context.Context, domain.ResearchProjectInput) (domain.ResearchProject, error)
	List(context.Context, int, int) ([]domain.ResearchProject, int64, error)
	Get(context.Context, int64) (domain.ResearchProject, error)
}

// ResearchQuery 是 Research HTTP 适配器需要的最小应用接口。
type ResearchQuery interface {
	Create(context.Context, domain.ResearchProjectInput) (domain.ResearchProject, error)
	List(context.Context, ResearchListRequest) (ResearchListResponse, error)
	Get(context.Context, int64) (domain.ResearchProject, error)
}

// ResearchListRequest 是最近研究列表的分页请求。
type ResearchListRequest struct {
	Page     int
	PageSize int
}

// ResearchListResponse 是最近研究列表响应。
type ResearchListResponse struct {
	Data       []domain.ResearchProject `json:"data"`
	Pagination api.PaginationMeta       `json:"pagination"`
}

// Service 编排 Research 项目输入校验与持久化调用。
type Service struct {
	store ResearchStore
}

// NewService 创建 Research 应用服务。
func NewService(store ResearchStore) (*Service, error) {
	if store == nil {
		return nil, errors.New("research store is required")
	}
	return &Service{store: store}, nil
}

// Create 创建一个真实 Research 项目。
func (service *Service) Create(ctx context.Context, input domain.ResearchProjectInput) (domain.ResearchProject, error) {
	normalized, err := domain.NormalizeResearchProjectInput(input)
	if err != nil {
		return domain.ResearchProject{}, err
	}
	project, err := service.store.Create(ctx, normalized)
	if err != nil {
		return domain.ResearchProject{}, fmt.Errorf("create research project: %w", err)
	}
	return project, nil
}

// List 按持久化更新时间读取最近 Research 项目。
func (service *Service) List(ctx context.Context, request ResearchListRequest) (ResearchListResponse, error) {
	page, pageSize, err := normalizePagination(request)
	if err != nil {
		return ResearchListResponse{}, err
	}
	items, total, err := service.store.List(ctx, page, pageSize)
	if err != nil {
		return ResearchListResponse{}, fmt.Errorf("list research projects: %w", err)
	}
	if items == nil {
		items = []domain.ResearchProject{}
	}
	return ResearchListResponse{
		Data: items,
		Pagination: api.PaginationMeta{
			Page: page, PageSize: pageSize, Total: total,
			TotalPages: totalPages(total, int64(pageSize)),
		},
	}, nil
}

// Get 读取一个 Research 项目的真实基础元数据。
func (service *Service) Get(ctx context.Context, id int64) (domain.ResearchProject, error) {
	if err := validateID(id); err != nil {
		return domain.ResearchProject{}, err
	}
	project, err := service.store.Get(ctx, id)
	if err != nil {
		return domain.ResearchProject{}, fmt.Errorf("get research project: %w", err)
	}
	return project, nil
}

func normalizePagination(request ResearchListRequest) (int, int, error) {
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

func validateID(id int64) error {
	if id < 1 {
		return &domain.ValidationError{Fields: map[string]string{"id": "必须是大于等于 1 的整数"}}
	}
	return nil
}

func totalPages(total, pageSize int64) int64 {
	if total == 0 {
		return 0
	}
	return (total + pageSize - 1) / pageSize
}

var _ ResearchQuery = (*Service)(nil)
