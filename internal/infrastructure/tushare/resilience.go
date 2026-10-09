package tushare

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	mathrand "math/rand"
	"sync"
	"time"
)

const (
	defaultGlobalPerMinute     = 100
	defaultAPIPerMinute        = 100
	defaultStockBasicPerMinute = 40
	defaultMaxAttempts         = 3
	defaultBaseBackoff         = 100 * time.Millisecond
	defaultMaxBackoff          = 2 * time.Second
)

// RateLimitConfig configures the process-local global and per-API request rates.
type RateLimitConfig struct {
	GlobalPerMinute     int
	DefaultAPIPerMinute int
	APIOverrides        map[string]int
}

// RetryConfig bounds retries for explicitly recoverable provider failures.
type RetryConfig struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	Jitter      func(time.Duration) time.Duration
}

// Observation contains only allowlisted, credential-free query metadata.
type Observation struct {
	RequestID    string        `json:"request_id"`
	APIName      string        `json:"api_name"`
	Duration     time.Duration `json:"duration"`
	RowCount     int           `json:"row_count"`
	BusinessDate string        `json:"business_date,omitempty"`
	ErrorClass   string        `json:"error_class,omitempty"`
	Attempts     int           `json:"attempts"`
}

// Observer receives a safe summary that can be adapted to logs and metrics.
// Implementations must support concurrent calls from Query callers.
type Observer interface {
	Observe(Observation)
}

// ObserverFunc adapts a function to the Observer interface.
type ObserverFunc func(Observation)

func (fn ObserverFunc) Observe(event Observation) { fn(event) }

type clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) Sleep(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}

type rateLimiter struct {
	mu                 sync.Mutex
	clock              clock
	globalInterval     time.Duration
	defaultAPIInterval time.Duration
	apiIntervals       map[string]time.Duration
	nextGlobal         time.Time
	nextAPI            map[string]time.Time
}

func newRateLimiter(config RateLimitConfig, source clock) *rateLimiter {
	if config.GlobalPerMinute <= 0 {
		config.GlobalPerMinute = defaultGlobalPerMinute
	}
	if config.DefaultAPIPerMinute <= 0 {
		config.DefaultAPIPerMinute = defaultAPIPerMinute
	}
	if source == nil {
		source = realClock{}
	}
	intervals := make(map[string]time.Duration, len(config.APIOverrides)+1)
	for apiName, perMinute := range config.APIOverrides {
		if perMinute > 0 {
			intervals[apiName] = time.Minute / time.Duration(perMinute)
		}
	}
	if _, exists := intervals["stock_basic"]; !exists {
		intervals["stock_basic"] = time.Minute / defaultStockBasicPerMinute
	}
	return &rateLimiter{
		clock: source, globalInterval: time.Minute / time.Duration(config.GlobalPerMinute),
		defaultAPIInterval: time.Minute / time.Duration(config.DefaultAPIPerMinute),
		apiIntervals:       intervals, nextAPI: make(map[string]time.Time),
	}
}

func (limiter *rateLimiter) Wait(ctx context.Context, apiName string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := limiter.clock.Now()
	limiter.mu.Lock()
	deadline := now
	if limiter.nextGlobal.After(deadline) {
		deadline = limiter.nextGlobal
	}
	if limiter.nextAPI[apiName].After(deadline) {
		deadline = limiter.nextAPI[apiName]
	}
	limiter.nextGlobal = deadline.Add(limiter.globalInterval)
	apiInterval := limiter.defaultAPIInterval
	if configured, exists := limiter.apiIntervals[apiName]; exists {
		apiInterval = configured
	}
	limiter.nextAPI[apiName] = deadline.Add(apiInterval)
	limiter.mu.Unlock()
	return limiter.clock.Sleep(ctx, deadline.Sub(now))
}

type retryableError struct{ err error }

func (err retryableError) Error() string { return err.err.Error() }
func (err retryableError) Unwrap() error { return err.err }

func markRetryable(err error) error { return retryableError{err: err} }

func isRetryable(err error) bool {
	var retryable retryableError
	return errors.As(err, &retryable)
}

func validateResilienceConfig(rate RateLimitConfig, retry RetryConfig) error {
	if rate.GlobalPerMinute < 0 || rate.DefaultAPIPerMinute < 0 {
		return errors.New("rate limits must be positive when configured")
	}
	if int64(rate.GlobalPerMinute) > int64(time.Minute) || int64(rate.DefaultAPIPerMinute) > int64(time.Minute) {
		return errors.New("rate limits exceed timer resolution")
	}
	for apiName, perMinute := range rate.APIOverrides {
		if apiName == "" || perMinute <= 0 || int64(perMinute) > int64(time.Minute) {
			return errors.New("per-api rate overrides require a name and rate within timer resolution")
		}
	}
	if retry.MaxAttempts < 0 || retry.BaseBackoff < 0 || retry.MaxBackoff < 0 {
		return errors.New("retry settings cannot be negative")
	}
	baseBackoff := retry.BaseBackoff
	if baseBackoff == 0 {
		baseBackoff = defaultBaseBackoff
	}
	maxBackoff := retry.MaxBackoff
	if maxBackoff == 0 {
		maxBackoff = defaultMaxBackoff
	}
	if maxBackoff < baseBackoff {
		return errors.New("maximum retry backoff cannot be less than base backoff")
	}
	return nil
}

func normalizeRetryConfig(config RetryConfig) RetryConfig {
	if config.MaxAttempts == 0 {
		config.MaxAttempts = defaultMaxAttempts
	}
	if config.BaseBackoff == 0 {
		config.BaseBackoff = defaultBaseBackoff
	}
	if config.MaxBackoff == 0 {
		config.MaxBackoff = defaultMaxBackoff
	}
	if config.Jitter == nil {
		config.Jitter = func(max time.Duration) time.Duration {
			if max <= 0 {
				return 0
			}
			if max == time.Duration(math.MaxInt64) {
				return time.Duration(mathrand.Int63())
			}
			return time.Duration(mathrand.Int63n(int64(max) + 1))
		}
	}
	return config
}

func (config RetryConfig) backoff(retryNumber int) time.Duration {
	delay := config.BaseBackoff
	for step := 1; step < retryNumber && delay < config.MaxBackoff; step++ {
		if delay > config.MaxBackoff/2 {
			delay = config.MaxBackoff
			break
		}
		delay *= 2
	}
	if delay > config.MaxBackoff {
		delay = config.MaxBackoff
	}
	jittered := config.Jitter(delay)
	if jittered < 0 {
		return 0
	}
	if jittered > delay {
		return delay
	}
	return jittered
}

func newRequestID() (string, error) {
	var value [16]byte
	if _, err := cryptorand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate Tushare request id: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}

func businessDate(params map[string]any) string {
	for _, key := range []string{"trade_date", "start_date", "end_date"} {
		if value, ok := params[key].(string); ok {
			return value
		}
	}
	return ""
}
