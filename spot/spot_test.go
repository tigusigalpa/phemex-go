package spot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigusigalpa/phemex-go"
)

func TestClientEndpoints(t *testing.T) {
	type received struct {
		method, path, query string
		body                map[string]any
	}
	requests := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		requests <- received{r.Method, r.URL.Path, r.URL.RawQuery, payload}
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()

	client := NewClient(phemex.NewClient(phemex.Config{BaseURI: server.URL}))
	params := map[string]any{"marker": "value"}
	tests := []struct {
		name, method, path, query string
		body                      bool
		call                      func() error
	}{
		{"create", "PUT", "/spot/orders/create", "", true, func() error { _, err := client.CreateOrder(context.Background(), params); return err }},
		{"amend", "PUT", "/spot/orders", "", true, func() error { _, err := client.AmendOrder(context.Background(), params); return err }},
		{"cancel", "DELETE", "/spot/orders", "marker=value", false, func() error { _, err := client.CancelOrder(context.Background(), params); return err }},
		{"cancel all", "DELETE", "/spot/orders/all", "symbol=BTCUSDT", false, func() error { _, err := client.CancelAllOrders(context.Background(), "BTCUSDT"); return err }},
		{"open order", "GET", "/spot/orders/active", "marker=value", false, func() error { _, err := client.QueryOpenOrder(context.Background(), params); return err }},
		{"open orders", "GET", "/spot/orders", "symbol=BTCUSDT", false, func() error { _, err := client.QueryOpenOrders(context.Background(), "BTCUSDT"); return err }},
		{"wallets", "GET", "/spot/wallets", "", false, func() error { _, err := client.Wallets(context.Background()); return err }},
		{"history", "GET", "/api-data/spots/orders", "marker=value", false, func() error { _, err := client.OrderHistory(context.Background(), params); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatal(err)
			}
			got := <-requests
			if got.method != tt.method || got.path != tt.path {
				t.Fatalf("request = %s %s, want %s %s", got.method, got.path, tt.method, tt.path)
			}
			if tt.body && got.body["marker"] != "value" {
				t.Fatalf("body = %#v", got.body)
			}
			if got.query != tt.query {
				t.Fatalf("query = %q, want %q", got.query, tt.query)
			}
		})
	}
}
