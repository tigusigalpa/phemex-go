// Package phemex provides a Go client for the Phemex cryptocurrency exchange REST API.
//
// The Client handles request signing, retries, and error mapping. Subpackages
// expose typed methods for each API domain (market data, spot, USDⓈ-M, etc.).
package phemex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURI    = "https://api.phemex.com"
	defaultTimeout    = 30 * time.Second
	defaultRetries    = 3
	defaultRetryDelay = 1 * time.Second
	defaultExpiry     = 60

	headerAccessToken = "x-phemex-access-token"
	headerExpiry      = "x-phemex-request-expiry"
	headerSignature   = "x-phemex-request-signature"
	headerTracing     = "x-phemex-request-tracing"
)

// Config holds the settings used to configure a Client.
type Config struct {
	// APIKey is the Phemex API key. Optional for public market endpoints.
	APIKey string
	// APISecret is the Phemex API secret. Required for private endpoints.
	APISecret string
	// BaseURI is the base URL for the Phemex REST API.
	BaseURI string
	// HTTPClient allows a custom http.Client to be supplied.
	HTTPClient *http.Client
	// Timeout is the default request timeout.
	Timeout time.Duration
	// Retries is the number of retries for transient failures and rate limits.
	Retries int
	// RetryDelay is the base delay used for exponential backoff.
	RetryDelay time.Duration
	// RequestTracing is an optional trace token sent with signed requests.
	RequestTracing string
}

func (c Config) baseURI() string {
	if c.BaseURI != "" {
		return strings.TrimRight(c.BaseURI, "/")
	}
	return defaultBaseURI
}

func (c Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return defaultTimeout
}

func (c Config) retries() int {
	if c.Retries < 0 {
		return 0
	}
	if c.Retries > 0 {
		return c.Retries
	}
	return defaultRetries
}

func (c Config) retryDelay() time.Duration {
	if c.RetryDelay > 0 {
		return c.RetryDelay
	}
	return defaultRetryDelay
}

func (c Config) hasCredentials() bool {
	return c.APIKey != "" && c.APISecret != ""
}

// Client is the central HTTP client for the Phemex REST API.
type Client struct {
	config Config
	signer *Signer
	http   *http.Client

	dialerOnce sync.Once
}

// NewClient creates a new Phemex REST API client from the provided Config.
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
		signer: NewSigner(cfg.APISecret),
		http:   cfg.HTTPClient,
	}
}

func (c *Client) httpClient() *http.Client {
	if c.http != nil {
		return c.http
	}
	c.dialerOnce.Do(func() {
		c.http = &http.Client{Timeout: c.config.timeout()}
	})
	return c.http
}

// Do sends an HTTP request with the provided method and path. Query and body
// parameters are supplied through params. It returns the raw JSON-decoded
// response envelope, or an error if the request fails.
//
// For GET and DELETE requests params are encoded as query parameters. For
// POST/PUT/PATCH requests params are encoded as JSON in the request body.
func (c *Client) Do(ctx context.Context, method, path string, params map[string]any) (*Response, error) {
	method = strings.ToUpper(method)
	u, queryString, body, err := c.preparePayload(method, path, params)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != "" {
		bodyReader = bytes.NewReader([]byte(body))
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("phemex: create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "tigusigalpa/phemex-go")

	if c.config.hasCredentials() {
		req = c.signRequest(req, method, path, queryString, body)
	}
	if c.config.RequestTracing != "" {
		req.Header.Set(headerTracing, c.config.RequestTracing)
	}

	return c.executeWithRetries(ctx, req)
}

func (c *Client) preparePayload(method, path string, params map[string]any) (string, string, string, error) {
	path = "/" + strings.TrimLeft(path, "/")
	base := c.config.baseURI()

	if method == http.MethodGet || method == http.MethodDelete {
		filtered := filterParams(params)
		qs := buildQuery(filtered)
		if qs != "" {
			path = path + "?" + qs
		}
		return base + path, qs, "", nil
	}

	filtered := filterParams(params)
	body := ""
	if len(filtered) > 0 {
		b, err := json.Marshal(filtered)
		if err != nil {
			return "", "", "", fmt.Errorf("phemex: encode request body: %w", err)
		}
		body = string(b)
	}
	return base + path, "", body, nil
}

func (c *Client) signRequest(req *http.Request, method, path, queryString, body string) *http.Request {
	expiry := c.signer.Expiry(defaultExpiry)
	sig := c.signer.Sign(method, path, queryString, expiry, body)

	req.Header.Set(headerAccessToken, c.config.APIKey)
	req.Header.Set(headerExpiry, strconv.FormatInt(expiry, 10))
	req.Header.Set(headerSignature, sig)

	return req
}

func (c *Client) executeWithRetries(ctx context.Context, req *http.Request) (*Response, error) {
	var lastErr error
	maxAttempts := c.config.retries()

	for attempt := 0; attempt <= maxAttempts; attempt++ {
		resp, err := c.httpClient().Do(req)
		if err != nil {
			return nil, fmt.Errorf("phemex: http request failed: %w", err)
		}

		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("phemex: read response body: %w", readErr)
		}

		status := resp.StatusCode
		if status >= 200 && status < 300 {
			return parseResponse(body)
		}

		if status == http.StatusTooManyRequests {
			lastErr = newRateLimitError(status, body, resp.Header.Get("Retry-After"))
			if attempt < maxAttempts {
				sleepForRetry(ctx, resp.Header.Get("Retry-After"), c.config.retryDelay(), attempt)
				continue
			}
			return nil, lastErr
		}

		if status >= 500 {
			lastErr = newHTTPError(status, body)
			if attempt < maxAttempts {
				sleepForBackoff(ctx, c.config.retryDelay(), attempt)
				continue
			}
			return nil, lastErr
		}

		return nil, classifyHTTPError(status, body)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, newAPIError(0, "unexpected HTTP error")
}

func parseResponse(body []byte) (*Response, error) {
	if len(body) == 0 {
		return &Response{Raw: nil}, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("phemex: decode response: %w", err)
	}

	return &Response{Raw: raw}, nil
}

func sleepForRetry(ctx context.Context, retryAfter string, baseDelay time.Duration, attempt int) {
	delay := parseRetryAfter(retryAfter)
	if delay <= 0 {
		delay = baseDelay * time.Duration(1<<attempt)
	}
	if delay <= 0 {
		delay = baseDelay
	}
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}

func sleepForBackoff(ctx context.Context, baseDelay time.Duration, attempt int) {
	delay := baseDelay * time.Duration(1<<attempt)
	select {
	case <-time.After(delay):
	case <-ctx.Done():
	}
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	if i, err := strconv.Atoi(value); err == nil && i > 0 {
		return time.Duration(i) * time.Second
	}
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}
	return 0
}

func buildQuery(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	values := make(url.Values, len(params))
	for key, raw := range params {
		switch v := raw.(type) {
		case []string:
			for _, item := range v {
				values.Add(key, item)
			}
		case []any:
			for _, item := range v {
				values.Add(key, stringifyValue(item))
			}
		default:
			values.Set(key, stringifyValue(raw))
		}
	}
	return values.Encode()
}

func stringifyValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func filterParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}
	filtered := make(map[string]any, len(params))
	for k, v := range params {
		if v != nil {
			filtered[k] = v
		}
	}
	return filtered
}
