package tushare

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"stock-quant/internal/shared/apperror"
)

func TestTushareRateLimiterSmoothsGlobalAndPerAPILimits(t *testing.T) {
	defaults := newRateLimiter(RateLimitConfig{}, newTestClock(time.Unix(0, 0)))
	if defaults.globalInterval != 600*time.Millisecond || defaults.defaultAPIInterval != 600*time.Millisecond || defaults.apiIntervals["stock_basic"] != 1500*time.Millisecond {
		t.Fatalf("default limiter intervals = global %s, api %s, stock_basic %s", defaults.globalInterval, defaults.defaultAPIInterval, defaults.apiIntervals["stock_basic"])
	}
	clock := newTestClock(time.Unix(0, 0))
	limiter := newRateLimiter(RateLimitConfig{
		GlobalPerMinute:     100,
		DefaultAPIPerMinute: 100,
		APIOverrides:        map[string]int{"stock_basic": 40},
	}, clock)

	if err := limiter.Wait(context.Background(), "daily"); err != nil {
		t.Fatal(err)
	}
	dailyDone := make(chan error, 1)
	go func() { dailyDone <- limiter.Wait(context.Background(), "daily") }()
	if !clock.WaitForSleep(t, 1) {
		t.Fatal("second daily request did not wait for the global 100/minute limit")
	}
	clock.Advance(599 * time.Millisecond)
	select {
	case err := <-dailyDone:
		t.Fatalf("daily limiter released early: %v", err)
	default:
	}
	clock.Advance(time.Millisecond)
	if err := <-dailyDone; err != nil {
		t.Fatal(err)
	}

	stockClock := newTestClock(time.Unix(0, 0))
	stockLimiter := newRateLimiter(RateLimitConfig{
		GlobalPerMinute:     100,
		DefaultAPIPerMinute: 100,
		APIOverrides:        map[string]int{"stock_basic": 40},
	}, stockClock)
	if err := stockLimiter.Wait(context.Background(), "stock_basic"); err != nil {
		t.Fatal(err)
	}
	stockDone := make(chan error, 1)
	go func() { stockDone <- stockLimiter.Wait(context.Background(), "stock_basic") }()
	if !stockClock.WaitForSleep(t, 1) {
		t.Fatal("second stock_basic request did not wait for the 40/minute override")
	}
	stockClock.Advance(1499 * time.Millisecond)
	select {
	case err := <-stockDone:
		t.Fatalf("stock_basic limiter released early: %v", err)
	default:
	}
	stockClock.Advance(time.Millisecond)
	if err := <-stockDone; err != nil {
		t.Fatal(err)
	}
}

func TestTushareRateLimiterSerializesConcurrentReservations(t *testing.T) {
	clock := newTestClock(time.Unix(0, 0))
	limiter := newRateLimiter(RateLimitConfig{GlobalPerMinute: 60, DefaultAPIPerMinute: 60}, clock)
	const requests = 5
	results := make(chan error, requests)
	for range requests {
		go func() { results <- limiter.Wait(context.Background(), "daily") }()
	}
	if !clock.WaitForSleep(t, requests-1) {
		t.Fatal("concurrent requests did not reserve all serialized wait slots")
	}
	clock.Advance(10 * time.Second)
	for range requests {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}

func TestTushareRateLimiterWaitCancellationDoesNotSendRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{"fields":["ts_code"],"items":[]}}`)
	}))
	defer server.Close()
	clock := newTestClock(time.Unix(0, 0))
	client, err := newClientWithClock(ClientConfig{
		Endpoint: server.URL, Token: "token-secret",
		RateLimits: RateLimitConfig{GlobalPerMinute: 1, DefaultAPIPerMinute: 1},
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Query(context.Background(), QueryRequest{APIName: "daily"}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.Query(ctx, QueryRequest{APIName: "daily"})
		done <- err
	}()
	if !clock.WaitForSleep(t, 1) {
		t.Fatal("second query did not enter limiter wait")
	}
	cancel()
	if err := <-done; err == nil || apperror.CodeOf(err) != apperror.CodeCancelled {
		t.Fatalf("cancelled query error = %v, want CANCELLED", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP requests = %d, want only the first request", got)
	}
}

func TestTushareRetryClassificationAndLimit(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		statuses  []int
		wantCalls int32
		wantErr   apperror.Code
	}{
		{name: "429 retries", statuses: []int{http.StatusTooManyRequests, http.StatusOK}, wantCalls: 2},
		{name: "5xx retries", statuses: []int{http.StatusBadGateway, http.StatusOK}, wantCalls: 2},
		{name: "permission never retries", responses: []string{`{"code":2002,"msg":"denied","data":null}`}, statuses: []int{http.StatusOK}, wantCalls: 1, wantErr: apperror.CodePermissionDenied},
		{name: "provider business rejection never retries", responses: []string{`{"code":17,"msg":"token-secret denied","data":null}`}, statuses: []int{http.StatusOK}, wantCalls: 1, wantErr: apperror.CodeUpstreamUnavailable},
		{name: "malformed payload never retries", responses: []string{`{"code":0`}, statuses: []int{http.StatusOK}, wantCalls: 1, wantErr: apperror.CodeUpstreamUnavailable},
		{name: "attempt limit includes first try", statuses: []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusServiceUnavailable}, wantCalls: 3, wantErr: apperror.CodeUpstreamUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				index := int(calls.Add(1) - 1)
				status := tt.statuses[min(index, len(tt.statuses)-1)]
				w.WriteHeader(status)
				if status == http.StatusOK {
					body := `{"code":0,"msg":"ok","data":{"fields":["ts_code"],"items":[]}}`
					if len(tt.responses) > 0 {
						body = tt.responses[min(index, len(tt.responses)-1)]
					}
					_, _ = io.WriteString(w, body)
				}
			}))
			defer server.Close()
			client, err := NewClient(ClientConfig{
				Endpoint: server.URL, Token: "token-secret",
				Retry: RetryConfig{MaxAttempts: 3, BaseBackoff: time.Nanosecond, MaxBackoff: time.Nanosecond, Jitter: func(time.Duration) time.Duration { return 0 }},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Query(context.Background(), QueryRequest{APIName: "daily"})
			if tt.wantErr == "" && err != nil {
				t.Fatalf("Query() error = %v", err)
			}
			if tt.wantErr != "" && apperror.CodeOf(err) != tt.wantErr {
				t.Fatalf("Query() code = %q, want %q; err=%v", apperror.CodeOf(err), tt.wantErr, err)
			}
			if got := calls.Load(); got != tt.wantCalls {
				t.Fatalf("attempts = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestTushareRetryBackoffIsExponentialCappedAndJittered(t *testing.T) {
	config := normalizeRetryConfig(RetryConfig{
		MaxAttempts: 4,
		BaseBackoff: 100 * time.Millisecond,
		MaxBackoff:  250 * time.Millisecond,
		Jitter:      func(max time.Duration) time.Duration { return max / 2 },
	})
	want := []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 125 * time.Millisecond, 125 * time.Millisecond}
	for retryNumber, expected := range want {
		if got := config.backoff(retryNumber + 1); got != expected {
			t.Errorf("backoff(%d) = %s, want %s", retryNumber+1, got, expected)
		}
	}
}

func TestTushareRetryCancellationDuringBackoff(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	clock := newTestClock(time.Unix(0, 0))
	client, err := newClientWithClock(ClientConfig{
		Endpoint: server.URL, Token: "token-secret",
		Retry: RetryConfig{MaxAttempts: 3, BaseBackoff: time.Second, MaxBackoff: time.Second, Jitter: func(time.Duration) time.Duration { return time.Second }},
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.Query(ctx, QueryRequest{APIName: "daily"})
		done <- err
	}()
	if !clock.WaitForSleep(t, 1) {
		t.Fatal("retry did not enter backoff")
	}
	cancel()
	if err := <-done; err == nil || apperror.CodeOf(err) != apperror.CodeCancelled {
		t.Fatalf("cancelled retry error = %v, want CANCELLED", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP requests = %d, want 1", got)
	}
}

func TestTushareRetryBackoffUsesOneTotalClientTimeout(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		Endpoint: server.URL, Token: "token-secret", Timeout: 20 * time.Millisecond,
		Retry: RetryConfig{MaxAttempts: 3, BaseBackoff: time.Second, MaxBackoff: time.Second, Jitter: func(time.Duration) time.Duration { return time.Second }},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(context.Background(), QueryRequest{APIName: "daily"})
	if err == nil || apperror.CodeOf(err) != apperror.CodeTimeout {
		t.Fatalf("Query() error = %v, want total timeout", err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("HTTP attempts = %d, want no retry after the total deadline", got)
	}
}

func TestTushareRetryReacquiresRateLimitForEachAttempt(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{"fields":["ts_code"],"items":[]}}`)
	}))
	defer server.Close()
	clock := newTestClock(time.Unix(0, 0))
	client, err := newClientWithClock(ClientConfig{
		Endpoint: server.URL, Token: "token-secret",
		RateLimits: RateLimitConfig{GlobalPerMinute: 60, DefaultAPIPerMinute: 60},
		Retry:      RetryConfig{MaxAttempts: 2, BaseBackoff: time.Nanosecond, MaxBackoff: time.Nanosecond, Jitter: func(time.Duration) time.Duration { return 0 }},
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := client.Query(context.Background(), QueryRequest{APIName: "daily"})
		done <- err
	}()
	if !clock.WaitForSleep(t, 1) {
		t.Fatal("retry attempt did not wait for the limiter after backoff")
	}
	clock.Advance(time.Second)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("HTTP requests = %d, want 2", got)
	}
}

func TestTushareObservationContainsOnlyAllowlistedRedactedFields(t *testing.T) {
	var captured Observation
	var observations atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"msg":"token-secret","data":{"fields":["ts_code"],"items":[["000001.SZ"]]}}`)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		Endpoint: server.URL, Token: "token-secret",
		Observer: ObserverFunc(func(event Observation) { captured = event; observations.Add(1) }),
		Retry:    RetryConfig{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Query(context.Background(), QueryRequest{APIName: "daily", Params: map[string]any{
		"trade_date": "20261009", "token": "token-secret", "sensitive": "token-secret",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if observations.Load() != 1 || captured.APIName != "daily" || captured.RowCount != 1 || captured.BusinessDate != "20261009" || captured.ErrorClass != "" || captured.RequestID == "" || captured.Duration < 0 || captured.Attempts != 1 {
		t.Fatalf("unexpected observation: %+v", captured)
	}
	encoded, err := json.Marshal(captured)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "token-secret") || strings.Contains(string(encoded), "sensitive") {
		t.Fatalf("observation exposed a secret or arbitrary params: %s", encoded)
	}
}

func TestTushareObservationRedactsTokenEmbeddedInAPINameAndDates(t *testing.T) {
	var events []Observation
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{
		Endpoint: server.URL, Token: "daily",
		Observer: ObserverFunc(func(event Observation) { mu.Lock(); events = append(events, event); mu.Unlock() }),
		Retry:    RetryConfig{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client.Query(context.Background(), QueryRequest{APIName: "daily", Params: map[string]any{"start_date": "daily", "end_date": "20261009", "other": "secret"}})
	mu.Lock()
	defer mu.Unlock()
	if len(events) != 1 || strings.Contains(events[0].APIName, "daily") || strings.Contains(events[0].BusinessDate, "daily") || events[0].ErrorClass != string(apperror.CodeRateLimited) {
		t.Fatalf("observation was not safely classified: %+v", events)
	}
}

func TestTushareRetryTransientNetworkFailure(t *testing.T) {
	var calls atomic.Int32
	transport := roundTripFunc(func(*http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, temporaryNetworkError{}
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":0,"msg":"ok","data":{"fields":["ts_code"],"items":[]}}`))}, nil
	})
	client, err := NewClient(ClientConfig{
		Endpoint: "https://api.tushare.pro", Token: "token-secret", HTTPClient: &http.Client{Transport: transport},
		Retry: RetryConfig{MaxAttempts: 2, BaseBackoff: time.Nanosecond, MaxBackoff: time.Nanosecond, Jitter: func(time.Duration) time.Duration { return 0 }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Query(context.Background(), QueryRequest{APIName: "daily"}); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("attempts = %d, want transient network failure retried once", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

type temporaryNetworkError struct{}

func (temporaryNetworkError) Error() string   { return "temporary network failure" }
func (temporaryNetworkError) Timeout() bool   { return false }
func (temporaryNetworkError) Temporary() bool { return true }

var _ interface {
	error
	Timeout() bool
	Temporary() bool
} = temporaryNetworkError{}

type testClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []*testClockWaiter
}

type testClockWaiter struct {
	deadline time.Time
	done     chan struct{}
	once     sync.Once
}

func newTestClock(now time.Time) *testClock { return &testClock{now: now} }

func (clock *testClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.now
}

func (clock *testClock) Sleep(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if delay <= 0 {
		return nil
	}
	clock.mu.Lock()
	waiter := &testClockWaiter{deadline: clock.now.Add(delay), done: make(chan struct{})}
	clock.waiters = append(clock.waiters, waiter)
	clock.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waiter.done:
		return ctx.Err()
	}
}

func (clock *testClock) Advance(delta time.Duration) {
	clock.mu.Lock()
	clock.now = clock.now.Add(delta)
	remaining := clock.waiters[:0]
	var ready []*testClockWaiter
	for _, waiter := range clock.waiters {
		if !waiter.deadline.After(clock.now) {
			ready = append(ready, waiter)
		} else {
			remaining = append(remaining, waiter)
		}
	}
	clock.waiters = remaining
	clock.mu.Unlock()
	for _, waiter := range ready {
		waiter.once.Do(func() { close(waiter.done) })
	}
}

func (clock *testClock) WaitForSleep(t *testing.T, count int) bool {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		clock.mu.Lock()
		got := len(clock.waiters)
		clock.mu.Unlock()
		if got >= count {
			return true
		}
		time.Sleep(time.Millisecond)
	}
	return false
}
