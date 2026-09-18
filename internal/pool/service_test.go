package pool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/disturb-yy/stock-quant/internal/pool/domain"
)

type fakeStockPoolStore struct {
	created      domain.StockPool
	createArg    domain.StockPool
	listItems    []domain.StockPool
	listTotal    int64
	search       string
	page         int
	pageSize     int
	members      []domain.StockPoolMember
	memberTotal  int64
	member       domain.StockPoolMember
	memberCount  int64
	memberErr    error
	memberID     int64
	memberSymbol string
}

type fakeStockIdentityReader struct {
	exists bool
	err    error
	symbol string
}

func (store *fakeStockPoolStore) Create(_ context.Context, pool domain.StockPool) (domain.StockPool, error) {
	store.createArg = pool
	return store.created, nil
}

func (store *fakeStockPoolStore) List(_ context.Context, search string, page, pageSize int) ([]domain.StockPool, int64, error) {
	store.search, store.page, store.pageSize = search, page, pageSize
	return store.listItems, store.listTotal, nil
}

func (store *fakeStockPoolStore) Get(_ context.Context, id int64) (domain.StockPool, error) {
	return domain.StockPool{ID: id}, nil
}

func (store *fakeStockPoolStore) ListMembers(_ context.Context, id int64, page, pageSize int) ([]domain.StockPoolMember, int64, error) {
	store.memberID, store.page, store.pageSize = id, page, pageSize
	return store.members, store.memberTotal, store.memberErr
}

func (store *fakeStockPoolStore) AddMember(_ context.Context, id int64, symbol string) (domain.StockPoolMember, int64, error) {
	store.memberID, store.memberSymbol = id, symbol
	return store.member, store.memberCount, store.memberErr
}

func (store *fakeStockPoolStore) DeleteMember(_ context.Context, id int64, symbol string) (int64, error) {
	store.memberID, store.memberSymbol = id, symbol
	return store.memberCount, store.memberErr
}

func (reader *fakeStockIdentityReader) Exists(_ context.Context, symbol string) (bool, error) {
	reader.symbol = symbol
	return reader.exists, reader.err
}

func TestServiceCreatesManualPoolFromNormalizedMetadata(t *testing.T) {
	store := &fakeStockPoolStore{created: domain.StockPool{ID: 7, Source: domain.SourceManual, MemberCount: 0, CreatedAt: time.Unix(0, 0).UTC()}}
	service, err := NewService(store, &fakeStockIdentityReader{exists: true})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	description := "  仅供长期观察  "
	if _, err := service.Create(context.Background(), domain.StockPoolInput{Name: "  红利观察  ", Description: &description}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if store.createArg.Name != "红利观察" || *store.createArg.Description != "仅供长期观察" {
		t.Fatalf("create argument = %#v, want normalized metadata", store.createArg)
	}
	if store.createArg.Source != domain.SourceManual || store.createArg.MemberCount != 0 {
		t.Fatalf("create argument = %#v, want manual source and zero members", store.createArg)
	}
}

func TestServiceListNormalizesSearchAndUsesDefaultPagination(t *testing.T) {
	store := &fakeStockPoolStore{listItems: []domain.StockPool{}, listTotal: 0}
	service, err := NewService(store, &fakeStockIdentityReader{exists: true})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	result, err := service.List(context.Background(), StockPoolListRequest{Search: "  红利  "})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if store.search != "红利" || store.page != 1 || store.pageSize != 20 || result.Pagination.TotalPages != 0 {
		t.Fatalf("list request = %q/%d/%d, response = %#v", store.search, store.page, store.pageSize, result)
	}
}

func TestServiceListRejectsInvalidSearchAndPagination(t *testing.T) {
	service, err := NewService(&fakeStockPoolStore{}, &fakeStockIdentityReader{exists: true})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	_, err = service.List(context.Background(), StockPoolListRequest{Search: string(make([]rune, domain.MaxStockPoolSearchLength+1)), Page: 0})
	if err == nil {
		t.Fatal("List() error = nil, want invalid search")
	}
	_, err = service.List(context.Background(), StockPoolListRequest{Page: 0, PageSize: 101})
	if err == nil {
		t.Fatal("List() error = nil, want invalid pagination")
	}
}

func TestServiceManagesMembersWithRealIdentityChecks(t *testing.T) {
	store := &fakeStockPoolStore{
		member:      domain.StockPoolMember{Symbol: "000001.SZ", Name: "平安银行"},
		memberCount: 1,
		members:     []domain.StockPoolMember{{Symbol: "000001.SZ", Name: "平安银行"}},
		memberTotal: 1,
	}
	identity := &fakeStockIdentityReader{exists: true}
	service, err := NewService(store, identity)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	added, err := service.AddMember(context.Background(), 7, "000001.SZ")
	if err != nil || added.Member.Symbol != "000001.SZ" || added.MemberCount != 1 || identity.symbol != "000001.SZ" {
		t.Fatalf("AddMember() = %#v/%v, identity = %q", added, err, identity.symbol)
	}
	listed, err := service.ListMembers(context.Background(), 7, StockPoolMemberListRequest{Page: 1, PageSize: 20})
	if err != nil || listed.Pagination.Total != 1 || listed.Data[0].Symbol != "000001.SZ" {
		t.Fatalf("ListMembers() = %#v/%v", listed, err)
	}
	deleted, err := service.DeleteMember(context.Background(), 7, "000001.SZ")
	if err != nil || deleted.Symbol != "000001.SZ" || deleted.MemberCount != 1 {
		t.Fatalf("DeleteMember() = %#v/%v", deleted, err)
	}
}

func TestServiceRejectsUnknownMemberIdentityAndPropagatesConflict(t *testing.T) {
	store := &fakeStockPoolStore{memberErr: domain.ErrStockPoolMemberConflict}
	identity := &fakeStockIdentityReader{}
	service, err := NewService(store, identity)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.AddMember(context.Background(), 7, "000001.SZ"); !errors.Is(err, domain.ErrStockPoolInstrumentNotFound) {
		t.Fatalf("unknown identity error = %v, want not found", err)
	}
	identity.exists = true
	_, err = service.AddMember(context.Background(), 7, "000001.SZ")
	if !errors.Is(err, domain.ErrStockPoolMemberConflict) {
		t.Fatalf("duplicate identity error = %v, want conflict", err)
	}
}
