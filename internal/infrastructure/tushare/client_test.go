package tushare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"stock-quant/internal/shared/apperror"
)

func TestTushareQueryPostsRequestAndMapsReturnedFieldOrder(t *testing.T) {
	body := fixture(t, "reordered_response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Errorf("Content-Type = %q", got)
		}
		var request map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		var apiName, token, fields string
		_ = json.Unmarshal(request["api_name"], &apiName)
		_ = json.Unmarshal(request["token"], &token)
		_ = json.Unmarshal(request["fields"], &fields)
		if apiName != "daily" || token != "token-secret" || fields != "ts_code,trade_date,close" {
			t.Errorf("request identity = %q/%q/%q", apiName, token, fields)
		}
		var params map[string]string
		if err := json.Unmarshal(request["params"], &params); err != nil || params["trade_date"] != "20261008" {
			t.Errorf("params = %#v, %v", params, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, "token-secret", time.Second)
	rows, err := client.Query(context.Background(), QueryRequest{APIName: "daily", Params: map[string]any{"trade_date": "20261008"}, Fields: "ts_code,trade_date,close", RequiredFields: []string{"ts_code", "trade_date", "close"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || string(rows[0]["ts_code"]) != `"000001.SZ"` || string(rows[0]["trade_date"]) != `"20261008"` || string(rows[0]["close"]) != "12.3000000000001" || string(rows[0]["vendor_extra"]) != "null" || string(rows[0]["turnover"]) != "9000000000000000.25" {
		t.Fatalf("mapped row = %#v", rows)
	}
}

func TestTushareQueryRejectsMissingFieldsAndWrongRowWidth(t *testing.T) {
	tests := []struct {
		name, body string
		required   []string
	}{
		{name: "required field absent", body: `{"code":0,"msg":"","data":{"fields":["ts_code"],"items":[["000001.SZ"]]}}`, required: []string{"ts_code", "trade_date"}},
		{name: "row has fewer cells", body: `{"code":0,"msg":"","data":{"fields":["ts_code","trade_date"],"items":[["000001.SZ"]]}}`, required: []string{"ts_code"}},
		{name: "row has extra cells", body: `{"code":0,"msg":"","data":{"fields":["ts_code"],"items":[["000001.SZ","unexpected"]]}}`, required: []string{"ts_code"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := clientForBody(t, tt.body)
			_, err := client.Query(context.Background(), QueryRequest{APIName: "daily", Fields: "ts_code,trade_date", RequiredFields: tt.required})
			if err == nil {
				t.Fatal("Query() unexpectedly succeeded")
			}
			if got := apperror.CodeOf(err); got != apperror.CodeDataIncomplete {
				t.Fatalf("error code = %q, want DATA_INCOMPLETE: %v", got, err)
			}
		})
	}
}

func TestTushareQueryClassifiesPermissionAndRedactsToken(t *testing.T) {
	client := clientForFixture(t, "permission_response.json")
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily", Fields: "ts_code"})
	if err == nil || apperror.CodeOf(err) != apperror.CodePermissionDenied {
		t.Fatalf("Query() error = %v, code %q", err, apperror.CodeOf(err))
	}
	if strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("error exposed token: %v", err)
	}
}

func TestTushareQueryRedactsTokenFromRequestAndAPIContextErrors(t *testing.T) {
	client := clientForBody(t, `{"code":17,"msg":"rejected","data":null}`)
	client.token = "daily"
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || strings.Contains(err.Error(), "daily") {
		t.Fatalf("error contains token used as API name: %v", err)
	}

	client.token = "token-secret"
	_, err = client.Query(context.Background(), QueryRequest{APIName: "daily", Params: map[string]any{"custom": failingJSON{}}})
	if err == nil || strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("request encoding error exposed token: %v", err)
	}
}

type failingJSON struct{}

func (failingJSON) MarshalJSON() ([]byte, error) { return nil, errors.New("token-secret") }

func TestTushareQueryClassifiesHTTP500WithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "provider echoed token-secret")
	}))
	defer server.Close()
	client := newTestClient(t, server.URL, "token-secret", time.Second)
	client.retry.MaxAttempts = 1
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeUpstreamUnavailable {
		t.Fatalf("Query() error = %v, code %q", err, apperror.CodeOf(err))
	}
	if strings.Contains(err.Error(), "token-secret") {
		t.Fatalf("error exposed token: %v", err)
	}
}

func TestTushareQueryClassifiesHTTP429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTooManyRequests) }))
	defer server.Close()
	client := newTestClient(t, server.URL, "token-secret", time.Second)
	client.retry.MaxAttempts = 1
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeRateLimited {
		t.Fatalf("Query() error = %v, code %q", err, apperror.CodeOf(err))
	}
}

func TestNewTushareClientRejectsCredentialBearingEndpoints(t *testing.T) {
	for _, endpoint := range []string{"http://example.com/api", "https://user:pass@example.com", "https://example.com/api?token=secret"} {
		if _, err := NewClient(ClientConfig{Endpoint: endpoint, Token: "token-secret"}); err == nil {
			t.Errorf("NewClient(%q) unexpectedly succeeded", endpoint)
		}
	}
}

func TestTushareQueryDoesNotForwardTokenAcrossRedirect(t *testing.T) {
	forwarded := make(chan bool, 1)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		forwarded <- strings.Contains(string(body), "token-secret")
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	client := newTestClient(t, origin.URL, "token-secret", time.Second)
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeUpstreamUnavailable {
		t.Fatalf("redirect response error = %v, code %q", err, apperror.CodeOf(err))
	}
	select {
	case leaked := <-forwarded:
		t.Fatalf("redirect target received a request (token present: %t)", leaked)
	default:
	}
}

func TestTushareQueryAcceptsEmptyItems(t *testing.T) {
	client := clientForFixture(t, "empty_response.json")
	rows, err := client.Query(context.Background(), QueryRequest{APIName: "daily", Fields: "ts_code,trade_date", RequiredFields: []string{"ts_code", "trade_date"}})
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("Query() = %#v, %v; want empty non-nil rows", rows, err)
	}
}

func TestTushareQueryRejectsMalformedAndIncompleteEnvelope(t *testing.T) {
	tests := []struct{ name, body string }{
		{name: "malformed json", body: `{"code":0`},
		{name: "missing data", body: `{"code":0,"msg":"ok"}`},
		{name: "missing fields", body: `{"code":0,"msg":"ok","data":{"items":[]}}`},
		{name: "missing items", body: `{"code":0,"msg":"ok","data":{"fields":["ts_code"]}}`},
		{name: "duplicate fields", body: `{"code":0,"msg":"ok","data":{"fields":["ts_code","ts_code"],"items":[]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := clientForBody(t, tt.body)
			_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
			if err == nil {
				t.Fatal("Query() unexpectedly succeeded")
			}
			if apperror.CodeOf(err) != apperror.CodeUpstreamUnavailable && apperror.CodeOf(err) != apperror.CodeDataIncomplete {
				t.Fatalf("unexpected error classification %q: %v", apperror.CodeOf(err), err)
			}
		})
	}
}

func TestTushareQueryAppliesContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer closeTestServer(server)
	client := newTestClient(t, server.URL, "token-secret", 20*time.Millisecond)
	_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeTimeout || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Query() error = %v, code %q", err, apperror.CodeOf(err))
	}
}

func TestTushareQueryMapsCanceledContextAndResourceFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	client := newTestClient(t, server.URL, "token-secret", time.Second)
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Query(ctx, QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeCancelled || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel error = %v, code %q", err, apperror.CodeOf(err))
	}
	server.Close()
	_, err = client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeUpstreamUnavailable {
		t.Fatalf("closed server error = %v, code %q", err, apperror.CodeOf(err))
	}
}

func closeTestServer(server *httptest.Server) {
	server.CloseClientConnections()
	server.Close()
}

func newTestClient(t *testing.T, endpoint, token string, timeout time.Duration) *Client {
	t.Helper()
	client, err := NewClient(ClientConfig{Endpoint: endpoint, Token: token, HTTPClient: http.DefaultClient, Timeout: timeout})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
func clientForFixture(t *testing.T, name string) *Client {
	t.Helper()
	return clientForBody(t, string(fixture(t, name)))
}
func clientForBody(t *testing.T, body string) *Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return newTestClient(t, server.URL, "token-secret", time.Second)
}
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
