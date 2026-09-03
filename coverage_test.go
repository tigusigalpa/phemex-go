package phemex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConfigAndPayloadHelpers(t *testing.T) {
	cfg := Config{}
	if cfg.baseURI() != defaultBaseURI || cfg.timeout() != defaultTimeout || cfg.retries() != defaultRetries || cfg.retryDelay() != defaultRetryDelay {
		t.Fatal("defaults were not applied")
	}
	cfg = Config{BaseURI: "https://example.test///", Timeout: time.Second, Retries: -1, RetryDelay: time.Millisecond, APIKey: "key"}
	if cfg.baseURI() != "https://example.test" || cfg.timeout() != time.Second || cfg.retries() != 0 || cfg.retryDelay() != time.Millisecond || cfg.hasCredentials() {
		t.Fatal("custom configuration was not applied")
	}
	cfg.APISecret = "secret"
	if !cfg.hasCredentials() {
		t.Fatal("credentials should be complete")
	}

	client := NewClient(Config{BaseURI: "https://example.test"})
	u, query, body, err := client.preparePayload(http.MethodGet, "orders", map[string]any{"z": nil, "tags": []string{"a", "b"}, "enabled": true})
	if err != nil || u != "https://example.test/orders?enabled=true&tags=a&tags=b" || query != "enabled=true&tags=a&tags=b" || body != "" {
		t.Fatalf("GET payload = %q, %q, %q, %v", u, query, body, err)
	}
	u, query, body, err = client.preparePayload(http.MethodPut, "/orders", map[string]any{"value": 2})
	if err != nil || u != "https://example.test/orders" || query != "" || body != `{"value":2}` {
		t.Fatalf("PUT payload = %q, %q, %q, %v", u, query, body, err)
	}
	if _, _, _, err := client.preparePayload(http.MethodPost, "/orders", map[string]any{"invalid": func() {}}); err == nil {
		t.Fatal("expected JSON encoding error")
	}
}

func TestValueAndRetryHelpers(t *testing.T) {
	for value, want := range map[any]string{nil: "", true: "true", false: "false", "x": "x", int(2): "2", int64(3): "3", float64(1.5): "1.5"} {
		if got := stringifyValue(value); got != want {
			t.Fatalf("stringifyValue(%#v) = %q, want %q", value, got, want)
		}
	}
	if got := stringifyValue(struct{ Name string }{"x"}); got != "{x}" {
		t.Fatalf("fallback string = %q", got)
	}
	if got := buildQuery(map[string]any{"items": []any{1, "two"}}); got != "items=1&items=two" {
		t.Fatalf("query = %q", got)
	}
	if got := filterParams(map[string]any{"keep": 1, "drop": nil}); len(got) != 1 || got["keep"] != 1 {
		t.Fatalf("filtered params = %#v", got)
	}

	if parseRetryAfter("") != 0 || parseRetryAfter("bad") != 0 || parseRetryAfter("2") != 2*time.Second {
		t.Fatal("invalid Retry-After parsing")
	}
	if got := parseRetryAfter(time.Now().Add(time.Second).UTC().Format(http.TimeFormat)); got <= 0 || got > 2*time.Second {
		t.Fatalf("date Retry-After = %v", got)
	}
	if !isSafeMethod(http.MethodHead) || !isSafeMethod(http.MethodOptions) || isSafeMethod(http.MethodPost) {
		t.Fatal("safe method classification is wrong")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sleepForRetry(ctx, "", time.Hour, 0)
	sleepForBackoff(ctx, time.Hour, 0)
}

func TestErrorsAndResponseHelpers(t *testing.T) {
	for _, tt := range []struct{ body, want string }{
		{`{"msg":"message"}`, "message"}, {`{"message":"message"}`, "message"}, {`{"error":"message"}`, "message"}, {`{"code":7}`, "code 7"}, {"plain", "plain"}, {"", ""},
	} {
		if got := extractMessage([]byte(tt.body)); got != tt.want {
			t.Fatalf("extractMessage(%q) = %q", tt.body, got)
		}
	}

	if newAPIError(1, "x").Error() != "phemex API error 1: x" {
		t.Fatal("API error string")
	}
	if (&APIError{StatusCode: 2, Code: 3, Message: "x"}).Error() != "phemex API error 2 (code 3): x" {
		t.Fatal("API code error string")
	}
	if (&APIError{StatusCode: 2, Code: 3}).Error() != "phemex API error 2 (code 3)" {
		t.Fatal("API code-only error string")
	}
	if (&AuthenticationError{APIError{StatusCode: 401, Message: "bad"}}).Error() == "" || (&ValidationError{APIError{StatusCode: 400, Message: "bad"}}).Error() == "" || (&NotFoundError{APIError{StatusCode: 404, Message: "bad"}}).Error() == "" || (&RateLimitError{APIError: APIError{StatusCode: 429}}).Error() == "" {
		t.Fatal("typed error string")
	}
	for status, want := range map[int]any{http.StatusUnauthorized: &AuthenticationError{}, http.StatusBadRequest: &ValidationError{}, http.StatusForbidden: &ValidationError{}, http.StatusNotFound: &NotFoundError{}, http.StatusTeapot: &APIError{}} {
		got := classifyHTTPError(status, []byte(`{"code":42,"msg":"no"}`))
		switch want.(type) {
		case *AuthenticationError:
			if _, ok := got.(*AuthenticationError); !ok {
				t.Fatalf("status %d = %T", status, got)
			}
		case *ValidationError:
			if _, ok := got.(*ValidationError); !ok {
				t.Fatalf("status %d = %T", status, got)
			}
		case *NotFoundError:
			if _, ok := got.(*NotFoundError); !ok {
				t.Fatalf("status %d = %T", status, got)
			}
		case *APIError:
			if _, ok := got.(*APIError); !ok {
				t.Fatalf("status %d = %T", status, got)
			}
		}
	}

	response := &Response{Raw: map[string]json.RawMessage{"code": []byte("0"), "msg": []byte(`"ok"`), "data": []byte(`{"value":1}`), "error": []byte("null"), "id": []byte("9"), "result": []byte(`{"value":2}`)}}
	var data, result struct {
		Value int `json:"value"`
	}
	if response.Code() != 0 || response.Msg() != "ok" || response.ErrorMsg() != "" || response.ID() != 9 || !response.IsSuccess() || response.Data(&data) != nil || data.Value != 1 || response.Result(&result) != nil || result.Value != 2 || response.String() == "" {
		t.Fatal("response helpers")
	}
	if (&Response{}).Data(&data) != nil || (&Response{}).Result(&result) != nil || (&Response{Raw: map[string]json.RawMessage{"error": []byte(`"bad"`)}}).IsSuccess() {
		t.Fatal("empty/error response helpers")
	}
	if !strings.Contains(StringJSON(map[string]int{"x": 1}), `"x":1`) || !strings.HasPrefix(StringJSON(make(chan int)), "%!json(") {
		t.Fatal("StringJSON")
	}
}

func TestClientResponseFailuresAndRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/empty":
		case "/invalid":
			_, _ = w.Write([]byte("invalid"))
		case "/retry":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			_, _ = w.Write([]byte(`{"code":0}`))
		}
	}))
	defer server.Close()
	client := NewClient(Config{BaseURI: server.URL, Retries: 1, RetryDelay: time.Nanosecond})
	if response, err := client.Do(context.Background(), http.MethodGet, "/empty", nil); err != nil || response.Raw != nil {
		t.Fatalf("empty response = %#v, %v", response, err)
	}
	if _, err := client.Do(context.Background(), http.MethodGet, "/invalid", nil); err == nil {
		t.Fatal("expected decode error")
	}
	if _, err := client.Do(context.Background(), http.MethodGet, "/retry", nil); err == nil {
		t.Fatal("expected retry error")
	}
	if _, err := NewClient(Config{BaseURI: "://bad"}).Do(context.Background(), http.MethodGet, "/", nil); err == nil {
		t.Fatal("expected request creation error")
	}
}

func TestURLValuesAreStable(t *testing.T) {
	values, err := url.ParseQuery(buildQuery(map[string]any{"a": "x", "b": 2}))
	if err != nil || values.Get("a") != "x" || values.Get("b") != "2" {
		t.Fatal("query should parse")
	}
	if NewSigner("secret").Expiry(0) <= time.Now().Unix() {
		t.Fatal("default expiry should be in future")
	}
}
