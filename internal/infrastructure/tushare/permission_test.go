package tushare

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"stock-quant/internal/shared/apperror"
)

func TestPermissionRequirementDocumentsMinimumTier(t *testing.T) {
	tests := []struct {
		apiName       string
		group         APIGroup
		minimumPoints int
		pointsKnown   bool
	}{
		{apiName: "stock_basic", group: APIGroupCore, minimumPoints: 2000, pointsKnown: true},
		{apiName: "trade_cal", group: APIGroupCore, minimumPoints: 2000, pointsKnown: true},
		{apiName: "daily", group: APIGroupCore, minimumPoints: 120, pointsKnown: true},
		{apiName: "adj_factor", group: APIGroupCore, minimumPoints: 2000, pointsKnown: true},
		{apiName: "suspend_d", group: APIGroupOptional, minimumPoints: 2000, pointsKnown: true},
		{apiName: "stk_limit", group: APIGroupOptional, minimumPoints: 2000, pointsKnown: true},
		{apiName: "namechange", group: APIGroupOptional, pointsKnown: false},
		{apiName: "stock_st", group: APIGroupRestricted, minimumPoints: 3000, pointsKnown: true},
	}
	for _, test := range tests {
		t.Run(test.apiName, func(t *testing.T) {
			got, ok := LookupAPIRequirement(test.apiName)
			if !ok {
				t.Fatal("API requirement was not found")
			}
			if got.APIName != test.apiName || got.Group != test.group || got.MinimumPoints != test.minimumPoints || got.MinimumPointsKnown != test.pointsKnown {
				t.Fatalf("requirement = %#v; want group=%q minimum=%d known=%t", got, test.group, test.minimumPoints, test.pointsKnown)
			}
			if got.PermissionStatus != PermissionNotTested {
				t.Fatalf("permission status = %q, want %q", got.PermissionStatus, PermissionNotTested)
			}
		})
	}
}

func TestPermissionRequirementDoesNotInventUnknownAPIAccess(t *testing.T) {
	if _, ok := LookupAPIRequirement("unknown_api"); ok {
		t.Fatal("unknown API received an invented permission requirement")
	}
}

func TestPermissionProbeReportsUnavailableProviderWithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(nil)
	endpoint := server.URL
	server.Close()

	const token = "test-only-permission-token"
	client, err := NewClient(ClientConfig{Endpoint: endpoint, Token: token, Retry: RetryConfig{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(context.Background(), QueryRequest{APIName: "stock_basic"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeUpstreamUnavailable {
		t.Fatalf("permission probe error = %v, want UPSTREAM_UNAVAILABLE", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Fatal("permission probe exposed token in provider resource failure")
	}
}
