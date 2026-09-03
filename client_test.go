package phemex

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClient_PublicRequestDoesNotSendSignatureHeaders(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":""}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	_, err := client.Do(context.Background(), "GET", "/public/products", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Header.Get("x-phemex-access-token") != "" {
		t.Errorf("expected no access token header")
	}
	if received.Header.Get("x-phemex-request-signature") != "" {
		t.Errorf("expected no signature header")
	}
}

func TestClient_PrivateRequestAddsSignatureHeaders(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":""}`)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURI:   server.URL,
		APIKey:    "api-key",
		APISecret: "api-secret",
	})
	_, err := client.Do(context.Background(), "GET", "/accounts/accountPositions", map[string]any{
		"currency": "BTC",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := received.Header.Get("x-phemex-access-token"); got != "api-key" {
		t.Errorf("access token = %q, want api-key", got)
	}
	if received.Header.Get("x-phemex-request-expiry") == "" {
		t.Errorf("expected expiry header")
	}
	if received.Header.Get("x-phemex-request-signature") == "" {
		t.Errorf("expected signature header")
	}
	if !strings.Contains(received.URL.RawQuery, "currency=BTC") {
		t.Errorf("expected currency=BTC in query, got %q", received.URL.RawQuery)
	}
}

func TestClient_PostRequestBodyIsSigned(t *testing.T) {
	var received *http.Request
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body in handler: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":""}`)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURI:   server.URL,
		APIKey:    "api-key",
		APISecret: "api-secret",
	})
	_, err := client.Do(context.Background(), "POST", "/assets/transfer", map[string]any{
		"currency": "BTC",
		"amount":   "0.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if received.Method != "POST" {
		t.Errorf("method = %q, want POST", received.Method)
	}
	if string(body) != `{"amount":"0.1","currency":"BTC"}` {
		t.Errorf("body = %q", body)
	}
	if received.Header.Get("x-phemex-request-signature") == "" {
		t.Errorf("expected signature header")
	}
}

func TestClient_RepeatedQueryKeys(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":""}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	_, err := client.Do(context.Background(), "GET", "/orders/activeList", map[string]any{
		"ordStatus": []string{"New", "PartiallyFilled", "Untriggered"},
		"symbol":    "BTCUSD",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := received.URL.RawQuery
	if !strings.Contains(q, "ordStatus=New") || !strings.Contains(q, "ordStatus=PartiallyFilled") || !strings.Contains(q, "ordStatus=Untriggered") {
		t.Errorf("expected repeated ordStatus keys, got %q", q)
	}
	if !strings.Contains(q, "symbol=BTCUSD") {
		t.Errorf("expected symbol=BTCUSD, got %q", q)
	}
}

func TestClient_BooleanQueryParamsSerializedAsTrueFalse(t *testing.T) {
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":""}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	_, err := client.Do(context.Background(), "GET", "/md/orderbook", map[string]any{
		"symbol":  "BTCUSD",
		"withFee": true,
		"reverse": false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := received.URL.RawQuery
	if !strings.Contains(q, "withFee=true") {
		t.Errorf("expected withFee=true, got %q", q)
	}
	if !strings.Contains(q, "reverse=false") {
		t.Errorf("expected reverse=false, got %q", q)
	}
}

func TestClient_NotFoundError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"msg":"Not found"}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	_, err := client.Do(context.Background(), "GET", "/unknown", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Fatalf("expected *NotFoundError, got %T", err)
	}
}

func TestClient_RateLimitError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"msg":"Rate limit"}`)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURI:    server.URL,
		Retries:    2,
		RetryDelay: 1,
	})
	_, err := client.Do(context.Background(), "GET", "/public/products", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", calls)
	}
}

func TestClient_ResponseCodeAndData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"code":0,"msg":"ok","data":{"timestamp":1234567890}}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	resp, err := client.Do(context.Background(), "GET", "/public/time", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccess() {
		t.Errorf("expected success")
	}
	if resp.Code() != 0 {
		t.Errorf("code = %d", resp.Code())
	}
	var data struct {
		Timestamp int64 `json:"timestamp"`
	}
	if err := resp.Data(&data); err != nil {
		t.Fatalf("data unmarshal: %v", err)
	}
	if data.Timestamp != 1234567890 {
		t.Errorf("timestamp = %d", data.Timestamp)
	}
}

func TestClient_HTTP200WithAPIFailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code":10001,"msg":"duplicate order"}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL})
	_, err := client.Do(context.Background(), "GET", "/orders", nil)
	if err == nil {
		t.Fatal("expected API error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != 10001 || apiErr.Message != "duplicate order" {
		t.Fatalf("unexpected API error: %+v", apiErr)
	}
}

func TestClient_DoesNotRetryUnsafeRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"msg":"outcome unknown"}`)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURI: server.URL, Retries: 2, RetryDelay: time.Nanosecond})
	_, err := client.Do(context.Background(), "POST", "/orders/create", map[string]any{"symbol": "BTCUSD"})
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if calls != 1 {
		t.Fatalf("unsafe request was sent %d times, want 1", calls)
	}
}
