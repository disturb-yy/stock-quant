package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"stock-quant/internal/shared/apperror"
)

const (
	defaultEndpoint       = "https://api.tushare.pro"
	defaultRequestTimeout = 30 * time.Second
	maxResponseBodyBytes  = 16 << 20
)

// ClientConfig configures the provider endpoint and HTTP transport.
type ClientConfig struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
	Timeout    time.Duration
	RateLimits RateLimitConfig
	Retry      RetryConfig
	Observer   Observer
	Logger     *slog.Logger
}

// Client sends Tushare Pro JSON requests and parses returned field names.
type Client struct {
	endpoint string
	token    string
	http     *http.Client
	timeout  time.Duration
	limiter  *rateLimiter
	retry    RetryConfig
	observer Observer
	logger   *slog.Logger
	clock    clock
}

// QueryRequest describes one API call. RequiredFields are checked against the
// provider's returned fields, while additional returned fields are preserved.
type QueryRequest struct {
	APIName        string
	Params         map[string]any
	Fields         string
	RequiredFields []string
}

// Row maps returned field names to their original JSON cell values.
type Row map[string]json.RawMessage

type wireRequest struct {
	APIName string         `json:"api_name"`
	Token   string         `json:"token"`
	Params  map[string]any `json:"params"`
	Fields  string         `json:"fields"`
}

type wireResponse struct {
	Code *int   `json:"code"`
	Msg  string `json:"msg"`
	Data *struct {
		Fields *[]string            `json:"fields"`
		Items  *[][]json.RawMessage `json:"items"`
	} `json:"data"`
}

// NewClient builds a client with a bounded request timeout and a configurable endpoint.
func NewClient(config ClientConfig) (*Client, error) {
	return newClientWithClock(config, realClock{})
}

func newClientWithClock(config ClientConfig, source clock) (*Client, error) {
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("create Tushare client: endpoint must be an HTTP(S) URL without credentials, query, or fragment")
	}
	if parsed.Scheme == "http" {
		host := strings.ToLower(parsed.Hostname())
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return nil, errors.New("create Tushare client: plain HTTP is allowed only for loopback test endpoints")
		}
	}
	if strings.TrimSpace(config.Token) == "" {
		return nil, errors.New("create Tushare client: token is required")
	}
	if strings.Contains(endpoint, config.Token) {
		return nil, errors.New("create Tushare client: endpoint must not contain the token")
	}
	if err := validateResilienceConfig(config.RateLimits, config.Retry); err != nil {
		return nil, fmt.Errorf("create Tushare client: %w", err)
	}
	if config.Timeout <= 0 {
		config.Timeout = defaultRequestTimeout
	}
	if source == nil {
		source = realClock{}
	}
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	safeHTTPClient := *httpClient
	safeHTTPClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{
		endpoint: endpoint, token: config.Token, http: &safeHTTPClient, timeout: config.Timeout,
		limiter: newRateLimiter(config.RateLimits, source), retry: normalizeRetryConfig(config.Retry),
		observer: config.Observer, logger: logger, clock: source,
	}, nil
}

// Query posts an API request and maps each item using the response's field order.
func (client *Client) Query(ctx context.Context, query QueryRequest) ([]Row, error) {
	if client == nil {
		return nil, errors.New("query Tushare: client is required")
	}
	if ctx == nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("context is required"))
	}
	if strings.TrimSpace(query.APIName) == "" {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("api_name is required"))
	}
	if err := validateRequiredFields(query.RequiredFields); err != nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("required field list is invalid"))
	}
	params := query.Params
	if params == nil {
		params = map[string]any{}
	}
	requestBody, err := json.Marshal(wireRequest{APIName: query.APIName, Token: client.token, Params: params, Fields: query.Fields})
	if err != nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, errors.New("request parameters are not JSON serializable"))
	}

	requestContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	requestID, err := newRequestID()
	if err != nil {
		return nil, apperror.New(apperror.CodeInternal, err)
	}
	started := client.clock.Now()
	observation := Observation{
		RequestID: requestID, APIName: safeProviderMessage(query.APIName, client.token),
		BusinessDate: safeProviderMessage(businessDate(query.Params), client.token),
	}
	rows, attempts, err := client.queryWithRetry(requestContext, query, requestBody)
	observation.Duration = client.clock.Now().Sub(started)
	observation.RowCount = len(rows)
	observation.Attempts = attempts
	if err != nil {
		observation.ErrorClass = string(apperror.CodeOf(err))
	}
	client.logObservation(observation)
	if client.observer != nil {
		client.observer.Observe(observation)
	}
	return rows, err
}

func (client *Client) logObservation(event Observation) {
	client.logger.Info("Tushare query finished",
		slog.String("request_id", event.RequestID),
		slog.String("api_name", event.APIName),
		slog.Duration("duration", event.Duration),
		slog.Int("row_count", event.RowCount),
		slog.String("business_date", event.BusinessDate),
		slog.String("error_class", event.ErrorClass),
		slog.Int("attempts", event.Attempts),
	)
}

func (client *Client) queryWithRetry(ctx context.Context, query QueryRequest, body []byte) ([]Row, int, error) {
	for attempt := 1; attempt <= client.retry.MaxAttempts; attempt++ {
		if err := client.limiter.Wait(ctx, query.APIName); err != nil {
			return nil, attempt - 1, classifyWaitError(query.APIName, client.token, ctx, err)
		}
		rows, err := client.queryOnce(ctx, query, body)
		if err == nil {
			return rows, attempt, nil
		}
		if !isRetryable(err) || attempt == client.retry.MaxAttempts {
			return nil, attempt, err
		}
		if err := client.clock.Sleep(ctx, client.retry.backoff(attempt)); err != nil {
			return nil, attempt, classifyWaitError(query.APIName, client.token, ctx, err)
		}
	}
	return nil, 0, apperror.New(apperror.CodeInternal, errors.New("retry loop ended unexpectedly"))
}

func (client *Client) queryOnce(ctx context.Context, query QueryRequest, body []byte) ([]Row, error) {
	apiName := query.APIName
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, fmt.Errorf("build Tushare request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return nil, classifyTransportError(apiName, client.token, ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, markRetryable(providerError(apiName, client.token, apperror.CodeRateLimited, fmt.Errorf("HTTP status %d", response.StatusCode)))
	}
	if response.StatusCode >= 500 && response.StatusCode < 600 {
		return nil, markRetryable(providerError(apiName, client.token, apperror.CodeUpstreamUnavailable, fmt.Errorf("HTTP status %d", response.StatusCode)))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, providerError(apiName, client.token, apperror.CodeUpstreamUnavailable, fmt.Errorf("HTTP status %d", response.StatusCode))
	}
	body, err = io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, classifyTransportError(apiName, client.token, ctx, err)
	}
	if len(body) > maxResponseBodyBytes {
		return nil, providerError(apiName, client.token, apperror.CodeDataIncomplete, errors.New("response exceeds maximum size"))
	}
	return client.parseResponse(apiName, query.RequiredFields, body)
}

func (client *Client) parseResponse(apiName string, requiredFields []string, body []byte) ([]Row, error) {
	var decoded wireResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, providerError(apiName, client.token, apperror.CodeUpstreamUnavailable, errors.New("invalid JSON response"))
	}
	if decoded.Code == nil {
		return nil, providerError(apiName, client.token, apperror.CodeUpstreamUnavailable, errors.New("response code is missing"))
	}
	if *decoded.Code != 0 {
		return nil, client.providerResponseError(apiName, *decoded.Code, decoded.Msg)
	}
	if decoded.Data == nil || decoded.Data.Fields == nil || decoded.Data.Items == nil {
		return nil, providerError(apiName, client.token, apperror.CodeDataIncomplete, errors.New("response data, fields, or items are missing"))
	}
	if err := validateResponseFields(apiName, client.token, *decoded.Data.Fields, requiredFields); err != nil {
		return nil, err
	}
	return responseRows(apiName, client.token, *decoded.Data.Fields, *decoded.Data.Items)
}

func (client *Client) providerResponseError(apiName string, code int, message string) error {
	safeMessage := safeProviderMessage(message, client.token)
	cause := fmt.Errorf("provider code %d", code)
	if safeMessage != "" {
		cause = fmt.Errorf("provider code %d: %s", code, safeMessage)
	}
	if code == 2002 {
		return providerError(apiName, client.token, apperror.CodePermissionDenied, cause)
	}
	return providerError(apiName, client.token, apperror.CodeUpstreamUnavailable, cause)
}

func validateResponseFields(apiName, token string, fields, requiredFields []string) error {
	if len(fields) == 0 {
		return providerError(apiName, token, apperror.CodeDataIncomplete, errors.New("response fields are empty"))
	}
	indexes := make(map[string]int, len(fields))
	for index, field := range fields {
		if strings.TrimSpace(field) == "" {
			return providerError(apiName, token, apperror.CodeDataIncomplete, errors.New("response contains an empty field name"))
		}
		if _, exists := indexes[field]; exists {
			return providerError(apiName, token, apperror.CodeDataIncomplete, fmt.Errorf("response contains duplicate field %q", field))
		}
		indexes[field] = index
	}
	for _, required := range requiredFields {
		if _, exists := indexes[required]; !exists {
			return providerError(apiName, token, apperror.CodeDataIncomplete, fmt.Errorf("required field %q is missing", required))
		}
	}
	return nil
}

func responseRows(apiName, token string, fields []string, items [][]json.RawMessage) ([]Row, error) {
	rows := make([]Row, 0, len(items))
	for rowIndex, item := range items {
		if len(item) != len(fields) {
			return nil, providerError(apiName, token, apperror.CodeDataIncomplete, fmt.Errorf("row %d has %d cells for %d fields", rowIndex, len(item), len(fields)))
		}
		row := make(Row, len(fields))
		for fieldIndex, field := range fields {
			row[field] = append(json.RawMessage(nil), item[fieldIndex]...)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func validateRequiredFields(fields []string) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if strings.TrimSpace(field) == "" {
			return errors.New("required field names cannot be empty")
		}
		if _, exists := seen[field]; exists {
			return fmt.Errorf("required field %q is repeated", field)
		}
		seen[field] = struct{}{}
	}
	return nil
}

func classifyTransportError(apiName, token string, ctx context.Context, cause error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(cause, context.Canceled) {
		return providerError(apiName, token, apperror.CodeCancelled, context.Canceled)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(cause, context.DeadlineExceeded) {
		return providerError(apiName, token, apperror.CodeTimeout, context.DeadlineExceeded)
	}
	classified := providerError(apiName, token, apperror.CodeUpstreamUnavailable, errors.New("request transport failed"))
	if isTransientNetworkError(cause) {
		return markRetryable(classified)
	}
	return classified
}

func classifyWaitError(apiName, token string, ctx context.Context, cause error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(cause, context.Canceled) {
		return providerError(apiName, token, apperror.CodeCancelled, context.Canceled)
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(cause, context.DeadlineExceeded) {
		return providerError(apiName, token, apperror.CodeTimeout, context.DeadlineExceeded)
	}
	return providerError(apiName, token, apperror.CodeUpstreamUnavailable, errors.New("request wait failed"))
}

func isTransientNetworkError(cause error) bool {
	var networkError net.Error
	return errors.As(cause, &networkError) && networkError.Temporary() && !networkError.Timeout()
}

type safeCause struct {
	message string
	cause   error
}

func (cause safeCause) Error() string { return cause.message }
func (cause safeCause) Unwrap() error { return cause.cause }

func providerError(apiName, token string, code apperror.Code, cause error) error {
	message := safeProviderMessage(apiName+": "+cause.Error(), token)
	return apperror.New(code, safeCause{message: message, cause: cause})
}

func safeProviderMessage(message, token string) string {
	if token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
	}
	runes := []rune(strings.Join(strings.Fields(message), " "))
	if len(runes) > 256 {
		runes = runes[:256]
	}
	return string(runes)
}
