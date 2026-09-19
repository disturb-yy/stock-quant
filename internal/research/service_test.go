package research

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/research/domain"
)

type fakeResearchStore struct {
	created   domain.ResearchProject
	createArg domain.ResearchProjectInput
	items     []domain.ResearchProject
	total     int64
	page      int
	pageSize  int
	getErr    error
	listErr   error
}

func (store *fakeResearchStore) Create(_ context.Context, input domain.ResearchProjectInput) (domain.ResearchProject, error) {
	store.createArg = input
	return store.created, nil
}

func (store *fakeResearchStore) List(_ context.Context, page, pageSize int) ([]domain.ResearchProject, int64, error) {
	store.page, store.pageSize = page, pageSize
	return store.items, store.total, store.listErr
}

func (store *fakeResearchStore) Get(_ context.Context, _ int64) (domain.ResearchProject, error) {
	return domain.ResearchProject{}, store.getErr
}

func TestServiceCreateNormalizesMetadata(t *testing.T) {
	store := &fakeResearchStore{created: domain.ResearchProject{ID: 7}}
	service, err := NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	description := "  仅记录估值判断  "
	if _, err := service.Create(context.Background(), domain.ResearchProjectInput{Name: "  银行研究  ", Description: &description}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.createArg.Name != "银行研究" || store.createArg.Description == nil || *store.createArg.Description != "仅记录估值判断" {
		t.Fatalf("create argument = %#v, want normalized metadata", store.createArg)
	}
}

func TestServiceListUsesDefaultPaginationAndReturnsEmptyArray(t *testing.T) {
	store := &fakeResearchStore{items: []domain.ResearchProject{}, total: 0}
	service, err := NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	result, err := service.List(context.Background(), ResearchListRequest{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if store.page != 1 || store.pageSize != 20 || result.Data == nil || len(result.Data) != 0 || result.Pagination.TotalPages != 0 {
		t.Fatalf("request/response = %d/%d/%#v, want default pagination and empty array", store.page, store.pageSize, result)
	}
}

func TestServiceRejectsInvalidPaginationAndID(t *testing.T) {
	service, err := NewService(&fakeResearchStore{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.List(context.Background(), ResearchListRequest{PageSize: 101}); err == nil {
		t.Fatal("List() error = nil, want invalid pagination")
	}
	if _, err := service.Get(context.Background(), 0); err == nil {
		t.Fatal("Get() error = nil, want invalid id")
	}
}

func TestServicePropagatesNotFoundAndDependencyErrors(t *testing.T) {
	store := &fakeResearchStore{getErr: domain.ErrResearchProjectNotFound, listErr: errors.New("database unavailable")}
	service, err := NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.Get(context.Background(), 1); !errors.Is(err, domain.ErrResearchProjectNotFound) {
		t.Fatalf("Get() error = %v, want not found", err)
	}
	if _, err := service.List(context.Background(), ResearchListRequest{}); err == nil {
		t.Fatal("List() error = nil, want dependency error")
	}
}

func TestServiceReturnsProjectTimestampsUnchanged(t *testing.T) {
	createdAt := time.Unix(100, 0).UTC()
	store := &fakeResearchStore{created: domain.ResearchProject{ID: 1, CreatedAt: createdAt, UpdatedAt: createdAt}}
	service, err := NewService(store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	project, err := service.Create(context.Background(), domain.ResearchProjectInput{Name: "项目"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !project.CreatedAt.Equal(createdAt) || !project.UpdatedAt.Equal(createdAt) {
		t.Fatalf("timestamps = %#v, want unchanged store values", project)
	}
}
