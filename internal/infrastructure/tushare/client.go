package tushare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
}

// Client sends Tushare Pro JSON requests and parses returned field names.
type Client struct {
	endpoint string
	token    string
	http     *http.Client
	timeout  time.Duration
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
	if config.Timeout <= 0 {
		config.Timeout = defaultRequestTimeout
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	safeHTTPClient := *httpClient
	safeHTTPClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{endpoint: endpoint, token: config.Token, http: &safeHTTPClient, timeout: config.Timeout}, nil
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
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, client.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, apperror.New(apperror.CodeInvalidArgument, fmt.Errorf("build Tushare request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return nil, classifyTransportError(query.APIName, client.token, requestContext, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, providerError(query.APIName, client.token, apperror.CodeRateLimited, fmt.Errorf("HTTP status %d", response.StatusCode))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, providerError(query.APIName, client.token, apperror.CodeUpstreamUnavailable, fmt.Errorf("HTTP status %d", response.StatusCode))
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return nil, classifyTransportError(query.APIName, client.token, requestContext, err)
	}
	if len(body) > maxResponseBodyBytes {
		return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, errors.New("response exceeds maximum size"))
	}
	var decoded wireResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, providerError(query.APIName, client.token, apperror.CodeUpstreamUnavailable, errors.New("invalid JSON response"))
	}
	if decoded.Code == nil {
		return nil, providerError(query.APIName, client.token, apperror.CodeUpstreamUnavailable, errors.New("response code is missing"))
	}
	if *decoded.Code != 0 {
		message := safeProviderMessage(decoded.Msg, client.token)
		cause := fmt.Errorf("provider code %d", *decoded.Code)
		if message != "" {
			cause = fmt.Errorf("provider code %d: %s", *decoded.Code, message)
		}
		if *decoded.Code == 2002 {
			return nil, providerError(query.APIName, client.token, apperror.CodePermissionDenied, cause)
		}
		return nil, providerError(query.APIName, client.token, apperror.CodeUpstreamUnavailable, cause)
	}
	if decoded.Data == nil || decoded.Data.Fields == nil || decoded.Data.Items == nil {
		return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, errors.New("response data, fields, or items are missing"))
	}
	fields := *decoded.Data.Fields
	items := *decoded.Data.Items
	if len(fields) == 0 {
		return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, errors.New("response fields are empty"))
	}
	fieldIndexes := make(map[string]int, len(fields))
	for index, field := range fields {
		if strings.TrimSpace(field) == "" {
			return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, errors.New("response contains an empty field name"))
		}
		if _, exists := fieldIndexes[field]; exists {
			return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, fmt.Errorf("response contains duplicate field %q", field))
		}
		fieldIndexes[field] = index
	}
	for _, required := range query.RequiredFields {
		if _, exists := fieldIndexes[required]; !exists {
			return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, fmt.Errorf("required field %q is missing", required))
		}
	}
	rows := make([]Row, 0, len(items))
	for rowIndex, item := range items {
		if len(item) != len(fields) {
			return nil, providerError(query.APIName, client.token, apperror.CodeDataIncomplete, fmt.Errorf("row %d has %d cells for %d fields", rowIndex, len(item), len(fields)))
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
	return providerError(apiName, token, apperror.CodeUpstreamUnavailable, errors.New("request transport failed"))
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
