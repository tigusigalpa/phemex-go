package margin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigusigalpa/phemex-go"
)

func TestClientEndpoints(t *testing.T) {
	type received struct{ method, path, query string }
	requests := make(chan received, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- received{r.Method, r.URL.Path, r.URL.RawQuery}
		_, _ = w.Write([]byte(`{"code":0}`))
	}))
	defer server.Close()
	client := NewClient(phemex.NewClient(phemex.Config{BaseURI: server.URL}))
	params := map[string]any{"marker": "value"}
	tests := []struct {
		name, method, path, query string
		call                      func() error
	}{
		{"create", "PUT", "/margin-trade/orders/create", "", func() error { _, err := client.CreateOrder(context.Background(), params); return err }},
		{"cancel", "DELETE", "/margin-trade/orders", "marker=value", func() error { _, err := client.CancelOrder(context.Background(), params); return err }},
		{"cancel all", "DELETE", "/margin-trade/orders/all", "symbol=BTCUSDT", func() error { _, err := client.CancelAllOrders(context.Background(), "BTCUSDT"); return err }},
		{"open order", "GET", "/margin-trade/orders/active", "marker=value", func() error { _, err := client.QueryOpenOrder(context.Background(), params); return err }},
		{"borrow history", "GET", "/margin/borrow", "marker=value", func() error { _, err := client.BorrowHistory(context.Background(), params); return err }},
		{"borrow", "POST", "/margin/borrow", "", func() error { _, err := client.Borrow(context.Background(), params); return err }},
		{"payback", "POST", "/margin/payback", "", func() error { _, err := client.Payback(context.Background(), params); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.call(); err != nil {
				t.Fatal(err)
			}
			got := <-requests
			if got.method != tt.method || got.path != tt.path || got.query != tt.query {
				t.Fatalf("request = %+v, want %s %s?%s", got, tt.method, tt.path, tt.query)
			}
		})
	}
}
